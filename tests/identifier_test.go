package tests

import (
	"testing"

	"github.com/biyonik/go-fluent-sql/internal/validation"
)

func TestValidateIdentifier(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		wantErr    bool
	}{
		// Valid identifiers
		{"simple name", "users", false},
		{"with underscore", "user_name", false},
		{"with numbers", "user123", false},
		{"starts with underscore", "_private", false},
		{"table.column", "users.id", false},
		{"uppercase", "Users", false},
		{"mixed case", "UserName", false},
		{"single char", "a", false},
		{"underscore only", "_", false},

		// Invalid identifiers
		{"empty string", "", true},
		{"starts with number", "123users", true},
		{"contains space", "user name", true},
		{"contains dash", "user-name", true},
		{"contains special char", "user@name", true},
		{"contains semicolon", "users;", true},
		{"contains quote", "users'", true},
		{"contains double quote", `users"`, true},
		{"contains backtick", "users`", true},
		{"contains parenthesis", "users()", true},
		{"sql injection attempt", "users; DROP TABLE users;--", true},
		{"multiple dots", "a.b.c", true},
		{"starts with dot", ".users", true},
		{"ends with dot", "users.", true},
		{"only dot", ".", true},
		{"too long", string(make([]byte, 129)), true},

		// SQL injection attempts
		{"union injection", "users UNION SELECT", true},
		{"comment injection", "users--", true},
		{"or injection", "users OR 1=1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateIdentifier(tt.identifier)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIdentifier(%q) error = %v, wantErr %v", tt.identifier, err, tt.wantErr)
			}
		})
	}
}

func TestValidateTableWithAlias(t *testing.T) {
	tests := []struct {
		name      string
		table     string
		wantName  string
		wantAlias string
		wantErr   bool
	}{
		// Valid tables
		{"simple table", "users", "users", "", false},
		{"with AS alias", "users as u", "users", "u", false},
		{"with AS uppercase", "users AS u", "users", "u", false},
		{"with space alias", "users u", "users", "u", false},
		{"long alias", "users as usr", "users", "usr", false},
		{"underscore table", "user_accounts", "user_accounts", "", false},
		{"underscore alias", "users as user_alias", "users", "user_alias", false},

		// Invalid tables
		{"empty string", "", "", "", true},
		{"invalid table name", "123users", "", "", true},
		{"invalid alias", "users as 123", "", "", true},
		{"sql injection in table", "users; DROP", "", "", true},
		{"sql injection in alias", "users as u; DROP", "", "", true},
		{"multiple AS", "users as u as v", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, alias, err := validation.ValidateTableWithAlias(tt.table)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTableWithAlias(%q) error = %v, wantErr %v", tt.table, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if name != tt.wantName {
					t.Errorf("ValidateTableWithAlias(%q) name = %q, want %q", tt.table, name, tt.wantName)
				}
				if alias != tt.wantAlias {
					t.Errorf("ValidateTableWithAlias(%q) alias = %q, want %q", tt.table, alias, tt.wantAlias)
				}
			}
		})
	}
}

func TestSplitTableColumn(t *testing.T) {
	tests := []struct {
		name       string
		ref        string
		wantTable  string
		wantColumn string
		wantErr    bool
	}{
		// Valid references
		{"column only", "id", "", "id", false},
		{"table.column", "users.id", "users", "id", false},
		{"with underscore", "user_accounts.user_id", "user_accounts", "user_id", false},

		// Invalid references
		{"empty", "", "", "", true},
		{"too many dots", "a.b.c", "", "", true},
		{"invalid column", "users.123id", "", "", true},
		{"invalid table", "123users.id", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table, column, err := validation.SplitTableColumn(tt.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("SplitTableColumn(%q) error = %v, wantErr %v", tt.ref, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if table != tt.wantTable {
					t.Errorf("SplitTableColumn(%q) table = %q, want %q", tt.ref, table, tt.wantTable)
				}
				if column != tt.wantColumn {
					t.Errorf("SplitTableColumn(%q) column = %q, want %q", tt.ref, column, tt.wantColumn)
				}
			}
		})
	}
}

func TestIsReservedWord(t *testing.T) {
	reserved := []string{"select", "SELECT", "from", "FROM", "where", "insert", "update", "delete"}
	notReserved := []string{"users", "id", "name", "email", "foobar"}

	for _, word := range reserved {
		if !validation.IsReservedWord(word) {
			t.Errorf("IsReservedWord(%q) = false, want true", word)
		}
	}

	for _, word := range notReserved {
		if validation.IsReservedWord(word) {
			t.Errorf("IsReservedWord(%q) = true, want false", word)
		}
	}
}

func TestIdentifierError(t *testing.T) {
	err := &validation.IdentifierError{
		Identifier: "bad;name",
		Reason:     "contains invalid characters",
	}

	expected := "fluentsql: invalid identifier 'bad;name': contains invalid characters"
	if err.Error() != expected {
		t.Errorf("IdentifierError.Error() = %q, want %q", err.Error(), expected)
	}

	// Empty identifier
	err2 := &validation.IdentifierError{
		Identifier: "",
		Reason:     "cannot be empty",
	}
	expected2 := "fluentsql: invalid identifier: cannot be empty"
	if err2.Error() != expected2 {
		t.Errorf("IdentifierError.Error() = %q, want %q", err2.Error(), expected2)
	}
}

// Benchmark tests
func BenchmarkValidateIdentifier(b *testing.B) {
	identifiers := []string{"users", "user_accounts", "users.id", "a", "very_long_identifier_name"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, id := range identifiers {
			_ = validation.ValidateIdentifier(id)
		}
	}
}

func BenchmarkValidateTableWithAlias(b *testing.B) {
	tables := []string{"users", "users as u", "user_accounts AS ua"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, t := range tables {
			_, _, _ = validation.ValidateTableWithAlias(t)
		}
	}
}
