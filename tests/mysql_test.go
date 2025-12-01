package tests

import (
	"reflect"
	"testing"

	"github.com/biyonik/go-fluent-sql/dialect"
)

// mockBuilder implements QueryBuilder interface for testing
type mockBuilder struct {
	table      string
	tableAlias string
	columns    []string
	distinct   bool
	wheres     []dialect.WhereClause
	orders     []dialect.OrderClause
	joins      []dialect.JoinClause
	groupBy    []string
	having     []dialect.WhereClause
	limit      *int
	offset     *int
}

func (m *mockBuilder) GetTable() string                 { return m.table }
func (m *mockBuilder) GetTableAlias() string            { return m.tableAlias }
func (m *mockBuilder) GetColumns() []string             { return m.columns }
func (m *mockBuilder) IsDistinct() bool                 { return m.distinct }
func (m *mockBuilder) GetWheres() []dialect.WhereClause { return m.wheres }
func (m *mockBuilder) GetOrders() []dialect.OrderClause { return m.orders }
func (m *mockBuilder) GetJoins() []dialect.JoinClause   { return m.joins }
func (m *mockBuilder) GetGroupBy() []string             { return m.groupBy }
func (m *mockBuilder) GetHaving() []dialect.WhereClause { return m.having }
func (m *mockBuilder) GetLimit() *int                   { return m.limit }
func (m *mockBuilder) GetOffset() *int                  { return m.offset }

func intPtr(n int) *int { return &n }

func TestMySQLGrammar_Name(t *testing.T) {
	g := dialect.MySQL()
	if g.Name() != "mysql" {
		t.Errorf("Name() = %q, want %q", g.Name(), "mysql")
	}
}

func TestMySQLGrammar_Wrap(t *testing.T) {
	g := dialect.MySQL()

	tests := []struct {
		name       string
		identifier string
		want       string
		wantErr    bool
	}{
		{"simple", "users", "`users`", false},
		{"with underscore", "user_name", "`user_name`", false},
		{"table.column", "users.id", "`users`.`id`", false},
		{"star", "*", "*", false},
		{"invalid", "users;DROP", "", true},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := g.Wrap(tt.identifier)
			if (err != nil) != tt.wantErr {
				t.Errorf("Wrap(%q) error = %v, wantErr %v", tt.identifier, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Wrap(%q) = %q, want %q", tt.identifier, got, tt.want)
			}
		})
	}
}

func TestMySQLGrammar_WrapTable(t *testing.T) {
	g := dialect.MySQL()

	tests := []struct {
		name    string
		table   string
		want    string
		wantErr bool
	}{
		{"simple", "users", "`users`", false},
		{"with alias AS", "users as u", "`users` AS `u`", false},
		{"with alias space", "users u", "`users` AS `u`", false},
		{"invalid", "users;DROP", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := g.WrapTable(tt.table)
			if (err != nil) != tt.wantErr {
				t.Errorf("WrapTable(%q) error = %v, wantErr %v", tt.table, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("WrapTable(%q) = %q, want %q", tt.table, got, tt.want)
			}
		})
	}
}

func TestMySQLGrammar_Placeholder(t *testing.T) {
	g := dialect.MySQL()

	// MySQL always uses ?
	for i := 0; i < 10; i++ {
		if got := g.Placeholder(i); got != "?" {
			t.Errorf("Placeholder(%d) = %q, want %q", i, got, "?")
		}
	}
}

func TestMySQLGrammar_CompileSelect(t *testing.T) {
	g := dialect.MySQL()

	tests := []struct {
		name     string
		builder  *mockBuilder
		wantSQL  string
		wantArgs []any
		wantErr  bool
	}{
		{
			name: "simple select all",
			builder: &mockBuilder{
				table: "users",
			},
			wantSQL:  "SELECT * FROM `users`",
			wantArgs: []any{},
		},
		{
			name: "select specific columns",
			builder: &mockBuilder{
				table:   "users",
				columns: []string{"id", "name", "email"},
			},
			wantSQL:  "SELECT `id`, `name`, `email` FROM `users`",
			wantArgs: []any{},
		},
		{
			name: "select with distinct",
			builder: &mockBuilder{
				table:    "users",
				columns:  []string{"status"},
				distinct: true,
			},
			wantSQL:  "SELECT DISTINCT `status` FROM `users`",
			wantArgs: []any{},
		},
		{
			name: "select with where",
			builder: &mockBuilder{
				table: "users",
				wheres: []dialect.WhereClause{
					{Type: dialect.WhereTypeBasic, Boolean: dialect.WhereBooleanAnd, Column: "status", Operator: "=", Value: "active"},
				},
			},
			wantSQL:  "SELECT * FROM `users` WHERE `status` = ?",
			wantArgs: []any{"active"},
		},
		{
			name: "select with multiple where",
			builder: &mockBuilder{
				table: "users",
				wheres: []dialect.WhereClause{
					{Type: dialect.WhereTypeBasic, Boolean: dialect.WhereBooleanAnd, Column: "status", Operator: "=", Value: "active"},
					{Type: dialect.WhereTypeBasic, Boolean: dialect.WhereBooleanAnd, Column: "role", Operator: "=", Value: "admin"},
				},
			},
			wantSQL:  "SELECT * FROM `users` WHERE `status` = ? AND `role` = ?",
			wantArgs: []any{"active", "admin"},
		},
		{
			name: "select with or where",
			builder: &mockBuilder{
				table: "users",
				wheres: []dialect.WhereClause{
					{Type: dialect.WhereTypeBasic, Boolean: dialect.WhereBooleanAnd, Column: "role", Operator: "=", Value: "admin"},
					{Type: dialect.WhereTypeBasic, Boolean: dialect.WhereBooleanOr, Column: "role", Operator: "=", Value: "moderator"},
				},
			},
			wantSQL:  "SELECT * FROM `users` WHERE `role` = ? OR `role` = ?",
			wantArgs: []any{"admin", "moderator"},
		},
		{
			name: "select with where in",
			builder: &mockBuilder{
				table: "users",
				wheres: []dialect.WhereClause{
					{Type: dialect.WhereTypeIn, Boolean: dialect.WhereBooleanAnd, Column: "id", Values: []any{1, 2, 3}},
				},
			},
			wantSQL:  "SELECT * FROM `users` WHERE `id` IN (?, ?, ?)",
			wantArgs: []any{1, 2, 3},
		},
		{
			name: "select with where not in",
			builder: &mockBuilder{
				table: "users",
				wheres: []dialect.WhereClause{
					{Type: dialect.WhereTypeNotIn, Boolean: dialect.WhereBooleanAnd, Column: "status", Values: []any{"banned", "suspended"}},
				},
			},
			wantSQL:  "SELECT * FROM `users` WHERE `status` NOT IN (?, ?)",
			wantArgs: []any{"banned", "suspended"},
		},
		{
			name: "select with where between",
			builder: &mockBuilder{
				table: "users",
				wheres: []dialect.WhereClause{
					{Type: dialect.WhereTypeBetween, Boolean: dialect.WhereBooleanAnd, Column: "age", Values: []any{18, 65}},
				},
			},
			wantSQL:  "SELECT * FROM `users` WHERE `age` BETWEEN ? AND ?",
			wantArgs: []any{18, 65},
		},
		{
			name: "select with where null",
			builder: &mockBuilder{
				table: "users",
				wheres: []dialect.WhereClause{
					{Type: dialect.WhereTypeNull, Boolean: dialect.WhereBooleanAnd, Column: "deleted_at"},
				},
			},
			wantSQL:  "SELECT * FROM `users` WHERE `deleted_at` IS NULL",
			wantArgs: []any{},
		},
		{
			name: "select with where not null",
			builder: &mockBuilder{
				table: "users",
				wheres: []dialect.WhereClause{
					{Type: dialect.WhereTypeNotNull, Boolean: dialect.WhereBooleanAnd, Column: "email_verified_at"},
				},
			},
			wantSQL:  "SELECT * FROM `users` WHERE `email_verified_at` IS NOT NULL",
			wantArgs: []any{},
		},
		{
			name: "select with order by",
			builder: &mockBuilder{
				table: "users",
				orders: []dialect.OrderClause{
					{Column: "created_at", Direction: dialect.OrderDesc},
				},
			},
			wantSQL:  "SELECT * FROM `users` ORDER BY `created_at` DESC",
			wantArgs: []any{},
		},
		{
			name: "select with multiple order by",
			builder: &mockBuilder{
				table: "users",
				orders: []dialect.OrderClause{
					{Column: "status", Direction: dialect.OrderAsc},
					{Column: "created_at", Direction: dialect.OrderDesc},
				},
			},
			wantSQL:  "SELECT * FROM `users` ORDER BY `status` ASC, `created_at` DESC",
			wantArgs: []any{},
		},
		{
			name: "select with limit",
			builder: &mockBuilder{
				table: "users",
				limit: intPtr(10),
			},
			wantSQL:  "SELECT * FROM `users` LIMIT 10",
			wantArgs: []any{},
		},
		{
			name: "select with limit and offset",
			builder: &mockBuilder{
				table:  "users",
				limit:  intPtr(10),
				offset: intPtr(20),
			},
			wantSQL:  "SELECT * FROM `users` LIMIT 10 OFFSET 20",
			wantArgs: []any{},
		},
		{
			name: "select with group by",
			builder: &mockBuilder{
				table:   "orders",
				columns: []string{"status", "COUNT(*) as count"},
				groupBy: []string{"status"},
			},
			wantSQL:  "SELECT `status`, COUNT(*) as count FROM `orders` GROUP BY `status`",
			wantArgs: []any{},
		},
		{
			name: "select with join",
			builder: &mockBuilder{
				table:   "orders",
				columns: []string{"orders.id", "users.name"},
				joins: []dialect.JoinClause{
					{Type: dialect.JoinInner, Table: "users", First: "orders.user_id", Operator: "=", Second: "users.id"},
				},
			},
			wantSQL:  "SELECT `orders`.`id`, `users`.`name` FROM `orders` INNER JOIN `users` ON `orders`.`user_id` = `users`.`id`",
			wantArgs: []any{},
		},
		{
			name: "select with left join",
			builder: &mockBuilder{
				table: "users",
				joins: []dialect.JoinClause{
					{Type: dialect.JoinLeft, Table: "profiles", First: "users.id", Operator: "=", Second: "profiles.user_id"},
				},
			},
			wantSQL:  "SELECT * FROM `users` LEFT JOIN `profiles` ON `users`.`id` = `profiles`.`user_id`",
			wantArgs: []any{},
		},
		{
			name: "select with where date",
			builder: &mockBuilder{
				table: "orders",
				wheres: []dialect.WhereClause{
					{Type: dialect.WhereTypeDate, Boolean: dialect.WhereBooleanAnd, Column: "created_at", Value: "2024-01-15"},
				},
			},
			wantSQL:  "SELECT * FROM `orders` WHERE DATE(`created_at`) = ?",
			wantArgs: []any{"2024-01-15"},
		},
		{
			name: "select with where year",
			builder: &mockBuilder{
				table: "orders",
				wheres: []dialect.WhereClause{
					{Type: dialect.WhereTypeYear, Boolean: dialect.WhereBooleanAnd, Column: "created_at", Value: 2024},
				},
			},
			wantSQL:  "SELECT * FROM `orders` WHERE YEAR(`created_at`) = ?",
			wantArgs: []any{2024},
		},
		{
			name: "select with nested where",
			builder: &mockBuilder{
				table: "users",
				wheres: []dialect.WhereClause{
					{Type: dialect.WhereTypeBasic, Boolean: dialect.WhereBooleanAnd, Column: "status", Operator: "=", Value: "active"},
					{
						Type:    dialect.WhereTypeNested,
						Boolean: dialect.WhereBooleanAnd,
						Nested: []dialect.WhereClause{
							{Type: dialect.WhereTypeBasic, Boolean: dialect.WhereBooleanAnd, Column: "role", Operator: "=", Value: "admin"},
							{Type: dialect.WhereTypeBasic, Boolean: dialect.WhereBooleanOr, Column: "role", Operator: "=", Value: "moderator"},
						},
					},
				},
			},
			wantSQL:  "SELECT * FROM `users` WHERE `status` = ? AND (`role` = ? OR `role` = ?)",
			wantArgs: []any{"active", "admin", "moderator"},
		},
		{
			name: "complex query",
			builder: &mockBuilder{
				table:    "users",
				columns:  []string{"id", "name", "email"},
				distinct: false,
				wheres: []dialect.WhereClause{
					{Type: dialect.WhereTypeBasic, Boolean: dialect.WhereBooleanAnd, Column: "status", Operator: "=", Value: "active"},
					{Type: dialect.WhereTypeIn, Boolean: dialect.WhereBooleanAnd, Column: "role", Values: []any{"admin", "user"}},
				},
				orders: []dialect.OrderClause{
					{Column: "created_at", Direction: dialect.OrderDesc},
				},
				limit:  intPtr(10),
				offset: intPtr(0),
			},
			wantSQL:  "SELECT `id`, `name`, `email` FROM `users` WHERE `status` = ? AND `role` IN (?, ?) ORDER BY `created_at` DESC LIMIT 10 OFFSET 0",
			wantArgs: []any{"active", "admin", "user"},
		},
		{
			name:    "no table error",
			builder: &mockBuilder{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL, gotArgs, err := g.CompileSelect(tt.builder)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompileSelect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if gotSQL != tt.wantSQL {
					t.Errorf("CompileSelect() SQL = %q, want %q", gotSQL, tt.wantSQL)
				}
				if !reflect.DeepEqual(gotArgs, tt.wantArgs) {
					t.Errorf("CompileSelect() args = %v, want %v", gotArgs, tt.wantArgs)
				}
			}
		})
	}
}

func TestMySQLGrammar_CompileInsert(t *testing.T) {
	g := dialect.MySQL()

	tests := []struct {
		name     string
		builder  *mockBuilder
		data     map[string]any
		wantSQL  string
		wantArgs []any
		wantErr  bool
	}{
		{
			name:    "simple insert",
			builder: &mockBuilder{table: "users"},
			data: map[string]any{
				"name":  "John",
				"email": "john@example.com",
			},
			wantSQL:  "INSERT INTO `users` (`email`, `name`) VALUES (?, ?)",
			wantArgs: []any{"john@example.com", "John"}, // Alphabetically sorted
		},
		{
			name:    "insert with many columns",
			builder: &mockBuilder{table: "users"},
			data: map[string]any{
				"name":       "John",
				"email":      "john@example.com",
				"status":     "active",
				"created_at": "2024-01-15",
			},
			wantSQL:  "INSERT INTO `users` (`created_at`, `email`, `name`, `status`) VALUES (?, ?, ?, ?)",
			wantArgs: []any{"2024-01-15", "john@example.com", "John", "active"},
		},
		{
			name:    "no table",
			builder: &mockBuilder{},
			data:    map[string]any{"name": "John"},
			wantErr: true,
		},
		{
			name:    "no columns",
			builder: &mockBuilder{table: "users"},
			data:    map[string]any{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL, gotArgs, err := g.CompileInsert(tt.builder, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompileInsert() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if gotSQL != tt.wantSQL {
					t.Errorf("CompileInsert() SQL = %q, want %q", gotSQL, tt.wantSQL)
				}
				if !reflect.DeepEqual(gotArgs, tt.wantArgs) {
					t.Errorf("CompileInsert() args = %v, want %v", gotArgs, tt.wantArgs)
				}
			}
		})
	}
}

func TestMySQLGrammar_CompileUpdate(t *testing.T) {
	g := dialect.MySQL()

	tests := []struct {
		name     string
		builder  *mockBuilder
		data     map[string]any
		wantSQL  string
		wantArgs []any
		wantErr  bool
	}{
		{
			name:    "simple update",
			builder: &mockBuilder{table: "users"},
			data: map[string]any{
				"name": "Updated Name",
			},
			wantSQL:  "UPDATE `users` SET `name` = ?",
			wantArgs: []any{"Updated Name"},
		},
		{
			name: "update with where",
			builder: &mockBuilder{
				table: "users",
				wheres: []dialect.WhereClause{
					{Type: dialect.WhereTypeBasic, Boolean: dialect.WhereBooleanAnd, Column: "id", Operator: "=", Value: 1},
				},
			},
			data: map[string]any{
				"name":   "Updated",
				"status": "inactive",
			},
			wantSQL:  "UPDATE `users` SET `name` = ?, `status` = ? WHERE `id` = ?",
			wantArgs: []any{"Updated", "inactive", 1},
		},
		{
			name:    "no table",
			builder: &mockBuilder{},
			data:    map[string]any{"name": "John"},
			wantErr: true,
		},
		{
			name:    "no columns",
			builder: &mockBuilder{table: "users"},
			data:    map[string]any{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL, gotArgs, err := g.CompileUpdate(tt.builder, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompileUpdate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if gotSQL != tt.wantSQL {
					t.Errorf("CompileUpdate() SQL = %q, want %q", gotSQL, tt.wantSQL)
				}
				if !reflect.DeepEqual(gotArgs, tt.wantArgs) {
					t.Errorf("CompileUpdate() args = %v, want %v", gotArgs, tt.wantArgs)
				}
			}
		})
	}
}

func TestMySQLGrammar_CompileDelete(t *testing.T) {
	g := dialect.MySQL()

	tests := []struct {
		name     string
		builder  *mockBuilder
		wantSQL  string
		wantArgs []any
		wantErr  bool
	}{
		{
			name:     "delete all",
			builder:  &mockBuilder{table: "users"},
			wantSQL:  "DELETE FROM `users`",
			wantArgs: []any{},
		},
		{
			name: "delete with where",
			builder: &mockBuilder{
				table: "users",
				wheres: []dialect.WhereClause{
					{Type: dialect.WhereTypeBasic, Boolean: dialect.WhereBooleanAnd, Column: "id", Operator: "=", Value: 1},
				},
			},
			wantSQL:  "DELETE FROM `users` WHERE `id` = ?",
			wantArgs: []any{1},
		},
		{
			name: "delete with multiple conditions",
			builder: &mockBuilder{
				table: "users",
				wheres: []dialect.WhereClause{
					{Type: dialect.WhereTypeBasic, Boolean: dialect.WhereBooleanAnd, Column: "status", Operator: "=", Value: "inactive"},
					{Type: dialect.WhereTypeBasic, Boolean: dialect.WhereBooleanAnd, Column: "created_at", Operator: "<", Value: "2023-01-01"},
				},
			},
			wantSQL:  "DELETE FROM `users` WHERE `status` = ? AND `created_at` < ?",
			wantArgs: []any{"inactive", "2023-01-01"},
		},
		{
			name:    "no table",
			builder: &mockBuilder{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL, gotArgs, err := g.CompileDelete(tt.builder)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompileDelete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if gotSQL != tt.wantSQL {
					t.Errorf("CompileDelete() SQL = %q, want %q", gotSQL, tt.wantSQL)
				}
				if !reflect.DeepEqual(gotArgs, tt.wantArgs) {
					t.Errorf("CompileDelete() args = %v, want %v", gotArgs, tt.wantArgs)
				}
			}
		})
	}
}

func TestMySQLGrammar_CompileCount(t *testing.T) {
	g := dialect.MySQL()

	tests := []struct {
		name     string
		builder  *mockBuilder
		column   string
		wantSQL  string
		wantArgs []any
		wantErr  bool
	}{
		{
			name:     "count all",
			builder:  &mockBuilder{table: "users"},
			column:   "",
			wantSQL:  "SELECT COUNT(*) FROM `users`",
			wantArgs: []any{},
		},
		{
			name:     "count column",
			builder:  &mockBuilder{table: "users"},
			column:   "email",
			wantSQL:  "SELECT COUNT(`email`) FROM `users`",
			wantArgs: []any{},
		},
		{
			name: "count with where",
			builder: &mockBuilder{
				table: "users",
				wheres: []dialect.WhereClause{
					{Type: dialect.WhereTypeBasic, Boolean: dialect.WhereBooleanAnd, Column: "status", Operator: "=", Value: "active"},
				},
			},
			column:   "",
			wantSQL:  "SELECT COUNT(*) FROM `users` WHERE `status` = ?",
			wantArgs: []any{"active"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL, gotArgs, err := g.CompileCount(tt.builder, tt.column)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompileCount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if gotSQL != tt.wantSQL {
					t.Errorf("CompileCount() SQL = %q, want %q", gotSQL, tt.wantSQL)
				}
				if !reflect.DeepEqual(gotArgs, tt.wantArgs) {
					t.Errorf("CompileCount() args = %v, want %v", gotArgs, tt.wantArgs)
				}
			}
		})
	}
}

func TestMySQLGrammar_CompileExists(t *testing.T) {
	g := dialect.MySQL()

	builder := &mockBuilder{
		table: "users",
		wheres: []dialect.WhereClause{
			{Type: dialect.WhereTypeBasic, Boolean: dialect.WhereBooleanAnd, Column: "email", Operator: "=", Value: "test@example.com"},
		},
	}

	gotSQL, gotArgs, err := g.CompileExists(builder)
	if err != nil {
		t.Errorf("CompileExists() error = %v", err)
		return
	}

	wantSQL := "SELECT EXISTS(SELECT 1 FROM `users` WHERE `email` = ? LIMIT 1)"
	if gotSQL != wantSQL {
		t.Errorf("CompileExists() SQL = %q, want %q", gotSQL, wantSQL)
	}

	wantArgs := []any{"test@example.com"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Errorf("CompileExists() args = %v, want %v", gotArgs, wantArgs)
	}
}

func TestMySQLGrammar_CompileTruncate(t *testing.T) {
	g := dialect.MySQL()

	builder := &mockBuilder{table: "users"}
	gotSQL, err := g.CompileTruncate(builder)
	if err != nil {
		t.Errorf("CompileTruncate() error = %v", err)
		return
	}

	wantSQL := "TRUNCATE TABLE `users`"
	if gotSQL != wantSQL {
		t.Errorf("CompileTruncate() SQL = %q, want %q", gotSQL, wantSQL)
	}
}

func TestMySQLGrammar_CompileUpsert(t *testing.T) {
	g := dialect.MySQL()

	builder := &mockBuilder{table: "users"}
	data := map[string]any{
		"email": "john@example.com",
		"name":  "John",
	}

	gotSQL, gotArgs, err := g.CompileUpsert(builder, data, []string{"name"})
	if err != nil {
		t.Errorf("CompileUpsert() error = %v", err)
		return
	}

	wantSQL := "INSERT INTO `users` (`email`, `name`) VALUES (?, ?) ON DUPLICATE KEY UPDATE `name` = VALUES(`name`)"
	if gotSQL != wantSQL {
		t.Errorf("CompileUpsert() SQL = %q, want %q", gotSQL, wantSQL)
	}

	wantArgs := []any{"john@example.com", "John"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Errorf("CompileUpsert() args = %v, want %v", gotArgs, wantArgs)
	}
}

// Benchmark tests
func BenchmarkMySQLGrammar_CompileSelect(b *testing.B) {
	g := dialect.MySQL()
	builder := &mockBuilder{
		table:   "users",
		columns: []string{"id", "name", "email"},
		wheres: []dialect.WhereClause{
			{Type: dialect.WhereTypeBasic, Boolean: dialect.WhereBooleanAnd, Column: "status", Operator: "=", Value: "active"},
		},
		orders: []dialect.OrderClause{
			{Column: "created_at", Direction: dialect.OrderDesc},
		},
		limit: intPtr(10),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = g.CompileSelect(builder)
	}
}

func BenchmarkMySQLGrammar_CompileInsert(b *testing.B) {
	g := dialect.MySQL()
	builder := &mockBuilder{table: "users"}
	data := map[string]any{
		"name":   "John",
		"email":  "john@example.com",
		"status": "active",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = g.CompileInsert(builder, data)
	}
}

func BenchmarkMySQLGrammar_Wrap(b *testing.B) {
	g := dialect.MySQL()
	identifiers := []string{"users", "user_name", "users.id", "a"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, id := range identifiers {
			_, _ = g.Wrap(id)
		}
	}
}
