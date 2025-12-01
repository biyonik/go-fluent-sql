package tests

import (
	"testing"

	"github.com/biyonik/go-fluent-sql/internal/validation"
)

func TestValidateOperator(t *testing.T) {
	tests := []struct {
		name     string
		operator string
		wantErr  bool
	}{
		// Valid operators - comparison
		{"equals", "=", false},
		{"not equals", "!=", false},
		{"not equals alt", "<>", false},
		{"less than", "<", false},
		{"greater than", ">", false},
		{"less or equal", "<=", false},
		{"greater or equal", ">=", false},
		{"null safe equals", "<=>", false},

		// Valid operators - pattern
		{"like", "LIKE", false},
		{"like lowercase", "like", false},
		{"like mixed", "Like", false},
		{"not like", "NOT LIKE", false},
		{"not like lowercase", "not like", false},

		// Valid operators - null
		{"is", "IS", false},
		{"is not", "IS NOT", false},
		{"is lowercase", "is", false},
		{"is not lowercase", "is not", false},

		// Valid operators - set
		{"in", "IN", false},
		{"not in", "NOT IN", false},
		{"between", "BETWEEN", false},
		{"not between", "NOT BETWEEN", false},

		// Valid with whitespace
		{"equals with space", " = ", false},
		{"like with space", " LIKE ", false},

		// Invalid operators
		{"empty", "", true},
		{"invalid word", "EQUALS", true},
		{"sql injection", "= OR 1=1", true},
		{"semicolon", ";", true},
		{"drop", "DROP", true},
		{"union", "UNION", true},
		{"comment", "--", true},
		{"random text", "foobar", true},
		{"partial like", "LIK", true},
		{"partial in", "I", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateOperator(tt.operator)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOperator(%q) error = %v, wantErr %v", tt.operator, err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeOperator(t *testing.T) {
	tests := []struct {
		name     string
		operator string
		want     string
		wantErr  bool
	}{
		{"lowercase like", "like", "LIKE", false},
		{"mixed case", "Like", "LIKE", false},
		{"uppercase", "LIKE", "LIKE", false},
		{"with spaces", " like ", "LIKE", false},
		{"not like", "not like", "NOT LIKE", false},
		{"equals", "=", "=", false},
		{"invalid", "invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validation.NormalizeOperator(tt.operator)
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeOperator(%q) error = %v, wantErr %v", tt.operator, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("NormalizeOperator(%q) = %q, want %q", tt.operator, got, tt.want)
			}
		})
	}
}

func TestIsComparisonOperator(t *testing.T) {
	comparison := []string{"=", "!=", "<>", "<", ">", "<=", ">=", "<=>"}
	notComparison := []string{"LIKE", "IN", "BETWEEN", "IS", "NOT LIKE"}

	for _, op := range comparison {
		if !validation.IsComparisonOperator(op) {
			t.Errorf("IsComparisonOperator(%q) = false, want true", op)
		}
	}

	for _, op := range notComparison {
		if validation.IsComparisonOperator(op) {
			t.Errorf("IsComparisonOperator(%q) = true, want false", op)
		}
	}
}

func TestIsPatternOperator(t *testing.T) {
	pattern := []string{"LIKE", "like", "NOT LIKE", "not like"}
	notPattern := []string{"=", "IN", "BETWEEN", "IS"}

	for _, op := range pattern {
		if !validation.IsPatternOperator(op) {
			t.Errorf("IsPatternOperator(%q) = false, want true", op)
		}
	}

	for _, op := range notPattern {
		if validation.IsPatternOperator(op) {
			t.Errorf("IsPatternOperator(%q) = true, want false", op)
		}
	}
}

func TestIsNullOperator(t *testing.T) {
	nullOps := []string{"IS", "is", "IS NOT", "is not"}
	notNullOps := []string{"=", "LIKE", "IN", "BETWEEN"}

	for _, op := range nullOps {
		if !validation.IsNullOperator(op) {
			t.Errorf("IsNullOperator(%q) = false, want true", op)
		}
	}

	for _, op := range notNullOps {
		if validation.IsNullOperator(op) {
			t.Errorf("IsNullOperator(%q) = true, want false", op)
		}
	}
}

func TestAllowedOperators(t *testing.T) {
	ops := validation.AllowedOperators()

	if len(ops) == 0 {
		t.Error("AllowedOperators() returned empty slice")
	}

	// Check that known operators are in the list
	known := map[string]bool{"=": false, "LIKE": false, "IN": false}
	for _, op := range ops {
		if _, ok := known[op]; ok {
			known[op] = true
		}
	}

	for op, found := range known {
		if !found {
			t.Errorf("AllowedOperators() missing %q", op)
		}
	}
}

func TestOperatorError(t *testing.T) {
	err := &validation.OperatorError{
		Operator: "INVALID",
		Reason:   "not in allowed list",
	}

	expected := "fluentsql: invalid operator 'INVALID': not in allowed list"
	if err.Error() != expected {
		t.Errorf("OperatorError.Error() = %q, want %q", err.Error(), expected)
	}
}

// Benchmark tests
func BenchmarkValidateOperator(b *testing.B) {
	operators := []string{"=", "!=", "LIKE", "NOT LIKE", "IN", "BETWEEN"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, op := range operators {
			_ = validation.ValidateOperator(op)
		}
	}
}

func BenchmarkNormalizeOperator(b *testing.B) {
	operators := []string{"like", "LIKE", "not like", "=", ">="}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, op := range operators {
			_, _ = validation.NormalizeOperator(op)
		}
	}
}
