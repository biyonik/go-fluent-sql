package tests

import (
	"strings"
	"testing"

	"github.com/biyonik/go-fluent-sql/dialect"
)

// TestSQLInjection_Identifiers tests SQL injection via identifiers
func TestSQLInjection_Identifiers(t *testing.T) {
	g := dialect.MySQL()

	// All of these should fail validation
	maliciousIdentifiers := []string{
		// Classic SQL injection
		"users; DROP TABLE users;--",
		"users; DELETE FROM users;--",
		"users'; DROP TABLE users;--",
		`users"; DROP TABLE users;--`,
		"users`; DROP TABLE users;--",

		// Union-based injection
		"users UNION SELECT * FROM passwords",
		"users UNION ALL SELECT * FROM admin",
		"id UNION SELECT password FROM users",

		// Comment injection
		"users--",
		"users#",
		"users/**/",
		"users/* comment */",

		// Boolean-based injection
		"users OR 1=1",
		"users AND 1=1",
		"users OR 'a'='a'",
		"1 OR 1=1",

		// Time-based injection
		"users; WAITFOR DELAY '0:0:10'--",
		"users; SLEEP(10)--",
		"users; BENCHMARK(10000000,SHA1('test'))--",

		// Stacked queries
		"users; INSERT INTO admin VALUES('hacker','password');--",
		"users; UPDATE users SET admin=1;--",

		// Special characters
		"users\x00",
		"users\n",
		"users\r",
		"users\t",

		// Hex/encoded injection
		"0x75736572733b2044524f50205441424c452075736572733b2d2d", // hex encoded

		// Null byte injection
		"users%00",
		"users\x00admin",

		// Path traversal (if somehow used in file operations)
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32",
	}

	for _, identifier := range maliciousIdentifiers {
		t.Run(identifier[:min(len(identifier), 30)], func(t *testing.T) {
			_, err := g.Wrap(identifier)
			if err == nil {
				t.Errorf("Wrap(%q) should have returned error for malicious input", identifier)
			}
		})
	}
}

// TestSQLInjection_TableNames tests SQL injection via table names
func TestSQLInjection_TableNames(t *testing.T) {
	g := dialect.MySQL()

	maliciousTables := []string{
		"users; DROP TABLE users",
		"users AS u; DROP TABLE users",
		"users as u; DELETE FROM admin",
		"(SELECT * FROM passwords) as p",
		"users UNION SELECT * FROM admin",
	}

	for _, table := range maliciousTables {
		t.Run(table[:min(len(table), 30)], func(t *testing.T) {
			_, err := g.WrapTable(table)
			if err == nil {
				t.Errorf("WrapTable(%q) should have returned error", table)
			}
		})
	}
}

// TestSQLInjection_Operators tests SQL injection via operators
func TestSQLInjection_Operators(t *testing.T) {
	g := dialect.MySQL()

	maliciousOperators := []string{
		"= OR 1=1--",
		"=; DROP TABLE users;--",
		"LIKE; DELETE FROM users;--",
		"= UNION SELECT",
		"'; DROP TABLE users;--",
		"=1 OR 1=1",
		"> 0; DROP TABLE users",
	}

	builder := &mockBuilder{
		table: "users",
	}

	for _, op := range maliciousOperators {
		t.Run(op[:min(len(op), 20)], func(t *testing.T) {
			builder.wheres = []dialect.WhereClause{
				{Type: dialect.WhereTypeBasic, Boolean: dialect.WhereBooleanAnd, Column: "id", Operator: op, Value: 1},
			}
			_, _, err := g.CompileSelect(builder)
			if err == nil {
				t.Errorf("CompileSelect with operator %q should have returned error", op)
			}
		})
	}
}

// TestSQLInjection_ValuesArePlaceholders ensures values use placeholders
func TestSQLInjection_ValuesArePlaceholders(t *testing.T) {
	g := dialect.MySQL()

	// These malicious values should NEVER appear in the SQL string
	// They should only be in the args slice as placeholders
	maliciousValues := []any{
		"'; DROP TABLE users;--",
		"1 OR 1=1",
		"admin'--",
		"UNION SELECT * FROM passwords",
		"1; DELETE FROM users",
	}

	for _, value := range maliciousValues {
		t.Run("value_placeholder", func(t *testing.T) {
			builder := &mockBuilder{
				table: "users",
				wheres: []dialect.WhereClause{
					{Type: dialect.WhereTypeBasic, Boolean: dialect.WhereBooleanAnd, Column: "name", Operator: "=", Value: value},
				},
			}

			sql, args, err := g.CompileSelect(builder)
			if err != nil {
				t.Errorf("CompileSelect() error = %v", err)
				return
			}

			// The malicious value should NOT appear in the SQL string
			valueStr, ok := value.(string)
			if ok && strings.Contains(sql, valueStr) {
				t.Errorf("SQL contains raw value %q - should use placeholder!", valueStr)
			}

			// The value should be in args
			if len(args) == 0 {
				t.Error("args should contain the value")
			}
			if args[0] != value {
				t.Errorf("args[0] = %v, want %v", args[0], value)
			}

			// SQL should contain placeholder
			if !strings.Contains(sql, "?") {
				t.Error("SQL should contain placeholder ?")
			}
		})
	}
}

// TestSQLInjection_WhereIn tests SQL injection via WhereIn values
func TestSQLInjection_WhereIn(t *testing.T) {
	g := dialect.MySQL()

	maliciousValues := []any{
		"1); DROP TABLE users;--",
		"1 OR 1=1",
		"'); DELETE FROM users;--",
	}

	builder := &mockBuilder{
		table: "users",
		wheres: []dialect.WhereClause{
			{Type: dialect.WhereTypeIn, Boolean: dialect.WhereBooleanAnd, Column: "id", Values: maliciousValues},
		},
	}

	sql, args, err := g.CompileSelect(builder)
	if err != nil {
		t.Errorf("CompileSelect() error = %v", err)
		return
	}

	// Check that malicious strings don't appear in SQL
	for _, v := range maliciousValues {
		if str, ok := v.(string); ok {
			if strings.Contains(sql, str) {
				t.Errorf("SQL contains raw malicious value: %q", str)
			}
		}
	}

	// Check that we have the right number of placeholders
	placeholderCount := strings.Count(sql, "?")
	if placeholderCount != len(maliciousValues) {
		t.Errorf("Expected %d placeholders, got %d", len(maliciousValues), placeholderCount)
	}

	// Check args
	if len(args) != len(maliciousValues) {
		t.Errorf("Expected %d args, got %d", len(maliciousValues), len(args))
	}
}

// TestSQLInjection_Insert tests that INSERT values use placeholders
func TestSQLInjection_Insert(t *testing.T) {
	g := dialect.MySQL()

	builder := &mockBuilder{table: "users"}
	data := map[string]any{
		"name":  "'; DROP TABLE users;--",
		"email": "admin@example.com' OR '1'='1",
	}

	sql, args, err := g.CompileInsert(builder, data)
	if err != nil {
		t.Errorf("CompileInsert() error = %v", err)
		return
	}

	// Check that malicious values don't appear in SQL
	for _, v := range data {
		if str, ok := v.(string); ok {
			if strings.Contains(sql, str) {
				t.Errorf("SQL contains raw malicious value: %q", str)
			}
		}
	}

	// Check placeholders
	placeholderCount := strings.Count(sql, "?")
	if placeholderCount != len(data) {
		t.Errorf("Expected %d placeholders, got %d", len(data), placeholderCount)
	}

	// Values should be in args
	if len(args) != len(data) {
		t.Errorf("Expected %d args, got %d", len(data), len(args))
	}
}

// TestSQLInjection_Update tests that UPDATE values use placeholders
func TestSQLInjection_Update(t *testing.T) {
	g := dialect.MySQL()

	builder := &mockBuilder{
		table: "users",
		wheres: []dialect.WhereClause{
			{Type: dialect.WhereTypeBasic, Boolean: dialect.WhereBooleanAnd, Column: "id", Operator: "=", Value: "1; DROP TABLE users;--"},
		},
	}
	data := map[string]any{
		"name": "hacker'; UPDATE users SET admin=1;--",
	}

	sql, _, err := g.CompileUpdate(builder, data)
	if err != nil {
		t.Errorf("CompileUpdate() error = %v", err)
		return
	}

	// Check that malicious values don't appear in SQL
	maliciousStrings := []string{
		"'; DROP TABLE users;--",
		"UPDATE users SET admin=1",
		"1; DROP TABLE",
	}

	for _, s := range maliciousStrings {
		if strings.Contains(sql, s) {
			t.Errorf("SQL contains malicious string: %q", s)
		}
	}

	// Should have 2 placeholders (one for SET, one for WHERE)
	placeholderCount := strings.Count(sql, "?")
	if placeholderCount != 2 {
		t.Errorf("Expected 2 placeholders, got %d", placeholderCount)
	}
}

// TestSQLInjection_JoinConditions tests that JOIN conditions are validated
func TestSQLInjection_JoinConditions(t *testing.T) {
	g := dialect.MySQL()

	maliciousJoins := []dialect.JoinClause{
		{Type: dialect.JoinInner, Table: "admin", First: "users.id; DROP TABLE users", Operator: "=", Second: "admin.user_id"},
		{Type: dialect.JoinInner, Table: "admin", First: "users.id", Operator: "= OR 1=1;--", Second: "admin.user_id"},
		{Type: dialect.JoinInner, Table: "admin'; DROP TABLE admin;--", First: "users.id", Operator: "=", Second: "admin.user_id"},
	}

	for i, join := range maliciousJoins {
		t.Run(string(rune('A'+i)), func(t *testing.T) {
			builder := &mockBuilder{
				table: "users",
				joins: []dialect.JoinClause{join},
			}

			_, _, err := g.CompileSelect(builder)
			if err == nil {
				t.Error("CompileSelect with malicious JOIN should have returned error")
			}
		})
	}
}

// TestSQLInjection_OrderBy tests that ORDER BY columns are validated
func TestSQLInjection_OrderBy(t *testing.T) {
	g := dialect.MySQL()

	maliciousOrders := []dialect.OrderClause{
		{Column: "name; DROP TABLE users;--", Direction: dialect.OrderAsc},
		{Column: "1,1); DROP TABLE users;--", Direction: dialect.OrderDesc},
		{Column: "name UNION SELECT password FROM admin", Direction: dialect.OrderAsc},
	}

	for _, order := range maliciousOrders {
		t.Run(order.Column[:min(len(order.Column), 20)], func(t *testing.T) {
			builder := &mockBuilder{
				table:  "users",
				orders: []dialect.OrderClause{order},
			}

			_, _, err := g.CompileSelect(builder)
			if err == nil {
				t.Errorf("CompileSelect with ORDER BY %q should have returned error", order.Column)
			}
		})
	}
}

// TestSQLInjection_GroupBy tests that GROUP BY columns are validated
func TestSQLInjection_GroupBy(t *testing.T) {
	g := dialect.MySQL()

	maliciousGroups := []string{
		"status; DROP TABLE users;--",
		"status UNION SELECT password FROM admin",
		"1,1); DELETE FROM users;--",
	}

	for _, group := range maliciousGroups {
		t.Run(group[:min(len(group), 20)], func(t *testing.T) {
			builder := &mockBuilder{
				table:   "users",
				groupBy: []string{group},
			}

			_, _, err := g.CompileSelect(builder)
			if err == nil {
				t.Errorf("CompileSelect with GROUP BY %q should have returned error", group)
			}
		})
	}
}

// TestSQLInjection_ColumnSelection tests that SELECT columns are validated
func TestSQLInjection_ColumnSelection(t *testing.T) {
	g := dialect.MySQL()

	// These should be rejected
	maliciousColumns := []string{
		"id; DROP TABLE users;--",
		"id UNION SELECT * FROM passwords",
		"id,(SELECT password FROM admin)",
	}

	for _, col := range maliciousColumns {
		t.Run(col[:min(len(col), 20)], func(t *testing.T) {
			builder := &mockBuilder{
				table:   "users",
				columns: []string{col},
			}

			_, _, err := g.CompileSelect(builder)
			if err == nil {
				t.Errorf("CompileSelect with column %q should have returned error", col)
			}
		})
	}
}

// TestSecureOutputFormat tests that output SQL is properly formatted
func TestSecureOutputFormat(t *testing.T) {
	g := dialect.MySQL()

	builder := &mockBuilder{
		table:   "users",
		columns: []string{"id", "name"},
		wheres: []dialect.WhereClause{
			{Type: dialect.WhereTypeBasic, Boolean: dialect.WhereBooleanAnd, Column: "status", Operator: "=", Value: "active"},
		},
	}

	sql, args, err := g.CompileSelect(builder)
	if err != nil {
		t.Fatalf("CompileSelect() error = %v", err)
	}

	// Verify identifiers are properly wrapped
	if !strings.Contains(sql, "`id`") {
		t.Error("Column 'id' should be wrapped in backticks")
	}
	if !strings.Contains(sql, "`name`") {
		t.Error("Column 'name' should be wrapped in backticks")
	}
	if !strings.Contains(sql, "`users`") {
		t.Error("Table 'users' should be wrapped in backticks")
	}
	if !strings.Contains(sql, "`status`") {
		t.Error("WHERE column 'status' should be wrapped in backticks")
	}

	// Verify placeholder is used
	if !strings.Contains(sql, "= ?") {
		t.Error("Value should use placeholder")
	}

	// Verify value is in args, not SQL
	if strings.Contains(sql, "active") {
		t.Error("Value 'active' should not appear in SQL string")
	}
	if len(args) != 1 || args[0] != "active" {
		t.Errorf("args = %v, want [active]", args)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
