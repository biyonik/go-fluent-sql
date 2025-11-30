package fluentsql

import "errors"

// Sentinel errors for go-fluent-sql.
// These errors can be checked using errors.Is().
var (
	// ErrInvalidIdentifier is returned when a table or column name contains invalid characters.
	ErrInvalidIdentifier = errors.New("fluentsql: invalid SQL identifier")

	// ErrInvalidOperator is returned when an unsupported SQL operator is used.
	ErrInvalidOperator = errors.New("fluentsql: invalid SQL operator")

	// ErrNoRows is returned when a query returns no rows.
	ErrNoRows = errors.New("fluentsql: no rows in result set")

	// ErrNoTable is returned when a query is executed without specifying a table.
	ErrNoTable = errors.New("fluentsql: no table specified")

	// ErrNoColumns is returned when an insert/update has no columns.
	ErrNoColumns = errors.New("fluentsql: no columns specified")

	// ErrEmptyWhereIn is returned when WhereIn is called with an empty slice.
	ErrEmptyWhereIn = errors.New("fluentsql: empty slice passed to WhereIn")

	// ErrInvalidBetweenValues is returned when WhereBetween doesn't receive exactly 2 values.
	ErrInvalidBetweenValues = errors.New("fluentsql: BETWEEN requires exactly 2 values")

	// ErrNilDestination is returned when a nil pointer is passed as scan destination.
	ErrNilDestination = errors.New("fluentsql: nil destination pointer")

	// ErrInvalidDestination is returned when the destination is not a pointer to struct/slice.
	ErrInvalidDestination = errors.New("fluentsql: destination must be a pointer to struct or slice")

	// ErrTransactionClosed is returned when trying to use a closed transaction.
	ErrTransactionClosed = errors.New("fluentsql: transaction already closed")

	// ErrConnectionClosed is returned when trying to use a closed connection.
	ErrConnectionClosed = errors.New("fluentsql: connection closed")

	// ErrQueryTimeout is returned when a query exceeds the context deadline.
	ErrQueryTimeout = errors.New("fluentsql: query timeout exceeded")

	// ErrMigrationFailed is returned when a migration fails to execute.
	ErrMigrationFailed = errors.New("fluentsql: migration failed")

	// ErrTableExists is returned when trying to create a table that already exists.
	ErrTableExists = errors.New("fluentsql: table already exists")

	// ErrTableNotFound is returned when a referenced table doesn't exist.
	ErrTableNotFound = errors.New("fluentsql: table not found")
)

// QueryError wraps an error with additional query context.
type QueryError struct {
	Err     error
	Query   string
	Args    []interface{}
	Message string
}

func (e *QueryError) Error() string {
	if e.Message != "" {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Err.Error()
}

func (e *QueryError) Unwrap() error {
	return e.Err
}

// NewQueryError creates a new QueryError with context.
func NewQueryError(err error, query string, args []interface{}, message string) *QueryError {
	return &QueryError{
		Err:     err,
		Query:   query,
		Args:    args,
		Message: message,
	}
}

// ValidationError represents an identifier validation error.
type ValidationError struct {
	Identifier string
	Context    string
	Reason     string
}

func (e *ValidationError) Error() string {
	return "fluentsql: invalid " + e.Context + " '" + e.Identifier + "': " + e.Reason
}

func (e *ValidationError) Is(target error) bool {
	return target == ErrInvalidIdentifier
}

// NewValidationError creates a new ValidationError.
func NewValidationError(identifier, context, reason string) *ValidationError {
	return &ValidationError{
		Identifier: identifier,
		Context:    context,
		Reason:     reason,
	}
}
