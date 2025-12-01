package fluentsql

import (
	"fmt"
	"sort"
	"strings"

	"github.com/biyonik/go-fluent-sql/internal/validation"
)

// MySQLGrammar implements the Grammar interface for MySQL/MariaDB.
type MySQLGrammar struct {
	BaseGrammar
}

// NewMySQLGrammar creates a new MySQL grammar instance.
func NewMySQLGrammar() *MySQLGrammar {
	return &MySQLGrammar{
		BaseGrammar: BaseGrammar{
			name:       "mysql",
			dateFormat: "2006-01-02 15:04:05",
		},
	}
}

// Wrap wraps an identifier with backticks.
func (g *MySQLGrammar) Wrap(identifier string) (string, error) {
	if identifier == "*" {
		return "*", nil
	}

	if err := validation.ValidateIdentifier(identifier); err != nil {
		return "", err
	}

	// Handle table.column format
	if strings.Contains(identifier, ".") {
		parts := strings.Split(identifier, ".")
		wrapped := make([]string, len(parts))
		for i, part := range parts {
			wrapped[i] = "`" + part + "`"
		}
		return strings.Join(wrapped, "."), nil
	}

	return "`" + identifier + "`", nil
}

// WrapTable wraps a table name, handling aliases.
func (g *MySQLGrammar) WrapTable(table string) (string, error) {
	name, alias, err := validation.ValidateTableWithAlias(table)
	if err != nil {
		return "", err
	}

	wrapped := "`" + name + "`"
	if alias != "" {
		wrapped += " AS `" + alias + "`"
	}

	return wrapped, nil
}

// WrapValue wraps a column value reference.
func (g *MySQLGrammar) WrapValue(value string) (string, error) {
	return g.Wrap(value)
}

// Placeholder returns the MySQL placeholder (?).
func (g *MySQLGrammar) Placeholder(index int) string {
	return "?"
}

// CompileSelect compiles a SELECT query.
func (g *MySQLGrammar) CompileSelect(b *Builder) (string, []any, error) {
	if b.GetTable() == "" {
		return "", nil, ErrNoTable
	}

	var sql strings.Builder
	args := make([]any, 0)

	// SELECT
	sql.WriteString("SELECT ")

	// DISTINCT
	if b.IsDistinct() {
		sql.WriteString("DISTINCT ")
	}

	// Columns
	columns := b.GetColumns()
	if len(columns) == 0 {
		sql.WriteString("*")
	} else {
		wrappedCols := make([]string, len(columns))
		for i, col := range columns {
			// Check if it's a raw expression (contains spaces or special chars)
			if strings.ContainsAny(col, " ()") {
				wrappedCols[i] = col // Raw expression, don't wrap
			} else {
				wrapped, err := g.Wrap(col)
				if err != nil {
					return "", nil, err
				}
				wrappedCols[i] = wrapped
			}
		}
		sql.WriteString(strings.Join(wrappedCols, ", "))
	}

	// FROM
	table, err := g.WrapTable(b.GetTable())
	if err != nil {
		return "", nil, err
	}
	sql.WriteString(" FROM ")
	sql.WriteString(table)

	// JOIN
	joins := b.GetJoins()
	for _, join := range joins {
		joinSQL, err := g.compileJoin(join)
		if err != nil {
			return "", nil, err
		}
		sql.WriteString(" ")
		sql.WriteString(joinSQL)
	}

	// WHERE
	wheres := b.GetWheres()
	if len(wheres) > 0 {
		whereSQL, whereArgs, err := g.compileWheres(wheres)
		if err != nil {
			return "", nil, err
		}
		sql.WriteString(" WHERE ")
		sql.WriteString(whereSQL)
		args = append(args, whereArgs...)
	}

	// GROUP BY
	groupBy := b.GetGroupBy()
	if len(groupBy) > 0 {
		sql.WriteString(" GROUP BY ")
		wrappedGroups := make([]string, len(groupBy))
		for i, col := range groupBy {
			wrapped, err := g.Wrap(col)
			if err != nil {
				return "", nil, err
			}
			wrappedGroups[i] = wrapped
		}
		sql.WriteString(strings.Join(wrappedGroups, ", "))
	}

	// HAVING
	having := b.GetHaving()
	if len(having) > 0 {
		havingSQL, havingArgs, err := g.compileWheres(having)
		if err != nil {
			return "", nil, err
		}
		sql.WriteString(" HAVING ")
		sql.WriteString(havingSQL)
		args = append(args, havingArgs...)
	}

	// ORDER BY
	orders := b.GetOrders()
	if len(orders) > 0 {
		sql.WriteString(" ORDER BY ")
		orderParts := make([]string, len(orders))
		for i, order := range orders {
			if order.Raw != "" {
				orderParts[i] = order.Raw
			} else {
				wrapped, err := g.Wrap(order.Column)
				if err != nil {
					return "", nil, err
				}
				orderParts[i] = wrapped + " " + string(order.Direction)
			}
		}
		sql.WriteString(strings.Join(orderParts, ", "))
	}

	// LIMIT
	if limit := b.GetLimit(); limit != nil {
		sql.WriteString(fmt.Sprintf(" LIMIT %d", *limit))
	}

	// OFFSET
	if offset := b.GetOffset(); offset != nil {
		sql.WriteString(fmt.Sprintf(" OFFSET %d", *offset))
	}

	return sql.String(), args, nil
}

// CompileInsert compiles an INSERT query.
func (g *MySQLGrammar) CompileInsert(b *Builder, data map[string]any) (string, []any, error) {
	if b.GetTable() == "" {
		return "", nil, ErrNoTable
	}
	if len(data) == 0 {
		return "", nil, ErrNoColumns
	}

	var sql strings.Builder
	args := make([]any, 0, len(data))

	// Get sorted keys for deterministic output
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// INSERT INTO table
	table, err := g.WrapTable(b.GetTable())
	if err != nil {
		return "", nil, err
	}
	sql.WriteString("INSERT INTO ")
	sql.WriteString(table)

	// Columns
	sql.WriteString(" (")
	wrappedCols := make([]string, len(keys))
	for i, key := range keys {
		wrapped, err := g.Wrap(key)
		if err != nil {
			return "", nil, err
		}
		wrappedCols[i] = wrapped
		args = append(args, data[key])
	}
	sql.WriteString(strings.Join(wrappedCols, ", "))
	sql.WriteString(")")

	// VALUES
	sql.WriteString(" VALUES (")
	placeholders := make([]string, len(keys))
	for i := range keys {
		placeholders[i] = g.Placeholder(i)
	}
	sql.WriteString(strings.Join(placeholders, ", "))
	sql.WriteString(")")

	return sql.String(), args, nil
}

// CompileInsertBatch compiles a batch INSERT query.
func (g *MySQLGrammar) CompileInsertBatch(b *Builder, data []map[string]any) (string, []any, error) {
	if b.GetTable() == "" {
		return "", nil, ErrNoTable
	}
	if len(data) == 0 {
		return "", nil, ErrEmptyBatch
	}

	// Get columns from first row
	firstRow := data[0]
	if len(firstRow) == 0 {
		return "", nil, ErrNoColumns
	}

	keys := make([]string, 0, len(firstRow))
	for k := range firstRow {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sql strings.Builder
	args := make([]any, 0, len(data)*len(keys))

	// INSERT INTO table
	table, err := g.WrapTable(b.GetTable())
	if err != nil {
		return "", nil, err
	}
	sql.WriteString("INSERT INTO ")
	sql.WriteString(table)

	// Columns
	sql.WriteString(" (")
	wrappedCols := make([]string, len(keys))
	for i, key := range keys {
		wrapped, err := g.Wrap(key)
		if err != nil {
			return "", nil, err
		}
		wrappedCols[i] = wrapped
	}
	sql.WriteString(strings.Join(wrappedCols, ", "))
	sql.WriteString(") VALUES ")

	// Values for each row
	rowPlaceholders := make([]string, len(data))
	for i, row := range data {
		// Verify same columns
		if len(row) != len(keys) {
			return "", nil, ErrInconsistentBatch
		}

		placeholders := make([]string, len(keys))
		for j, key := range keys {
			val, ok := row[key]
			if !ok {
				return "", nil, ErrInconsistentBatch
			}
			placeholders[j] = g.Placeholder(len(args))
			args = append(args, val)
		}
		rowPlaceholders[i] = "(" + strings.Join(placeholders, ", ") + ")"
	}
	sql.WriteString(strings.Join(rowPlaceholders, ", "))

	return sql.String(), args, nil
}

// CompileUpdate compiles an UPDATE query.
func (g *MySQLGrammar) CompileUpdate(b *Builder, data map[string]any) (string, []any, error) {
	if b.GetTable() == "" {
		return "", nil, ErrNoTable
	}
	if len(data) == 0 {
		return "", nil, ErrNoColumns
	}

	var sql strings.Builder
	args := make([]any, 0)

	// Get sorted keys for deterministic output
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// UPDATE table
	table, err := g.WrapTable(b.GetTable())
	if err != nil {
		return "", nil, err
	}
	sql.WriteString("UPDATE ")
	sql.WriteString(table)

	// SET
	sql.WriteString(" SET ")
	setParts := make([]string, len(keys))
	for i, key := range keys {
		wrapped, err := g.Wrap(key)
		if err != nil {
			return "", nil, err
		}
		setParts[i] = wrapped + " = " + g.Placeholder(i)
		args = append(args, data[key])
	}
	sql.WriteString(strings.Join(setParts, ", "))

	// WHERE
	wheres := b.GetWheres()
	if len(wheres) > 0 {
		whereSQL, whereArgs, err := g.compileWheres(wheres)
		if err != nil {
			return "", nil, err
		}
		sql.WriteString(" WHERE ")
		sql.WriteString(whereSQL)
		args = append(args, whereArgs...)
	}

	return sql.String(), args, nil
}

// CompileDelete compiles a DELETE query.
func (g *MySQLGrammar) CompileDelete(b *Builder) (string, []any, error) {
	if b.GetTable() == "" {
		return "", nil, ErrNoTable
	}

	var sql strings.Builder
	args := make([]any, 0)

	// DELETE FROM table
	table, err := g.WrapTable(b.GetTable())
	if err != nil {
		return "", nil, err
	}
	sql.WriteString("DELETE FROM ")
	sql.WriteString(table)

	// WHERE
	wheres := b.GetWheres()
	if len(wheres) > 0 {
		whereSQL, whereArgs, err := g.compileWheres(wheres)
		if err != nil {
			return "", nil, err
		}
		sql.WriteString(" WHERE ")
		sql.WriteString(whereSQL)
		args = append(args, whereArgs...)
	}

	return sql.String(), args, nil
}

// CompileExists compiles an EXISTS query.
func (g *MySQLGrammar) CompileExists(b *Builder) (string, []any, error) {
	if b.GetTable() == "" {
		return "", nil, ErrNoTable
	}

	// Build a SELECT 1 query with same conditions
	var sql strings.Builder
	args := make([]any, 0)

	table, err := g.WrapTable(b.GetTable())
	if err != nil {
		return "", nil, err
	}

	sql.WriteString("SELECT EXISTS(SELECT 1 FROM ")
	sql.WriteString(table)

	// WHERE
	wheres := b.GetWheres()
	if len(wheres) > 0 {
		whereSQL, whereArgs, err := g.compileWheres(wheres)
		if err != nil {
			return "", nil, err
		}
		sql.WriteString(" WHERE ")
		sql.WriteString(whereSQL)
		args = append(args, whereArgs...)
	}

	sql.WriteString(" LIMIT 1)")

	return sql.String(), args, nil
}

// CompileCount compiles a COUNT query.
func (g *MySQLGrammar) CompileCount(b *Builder, column string) (string, []any, error) {
	if b.GetTable() == "" {
		return "", nil, ErrNoTable
	}

	var sql strings.Builder
	args := make([]any, 0)

	// COUNT expression
	countExpr := "COUNT(*)"
	if column != "" {
		wrapped, err := g.Wrap(column)
		if err != nil {
			return "", nil, err
		}
		countExpr = "COUNT(" + wrapped + ")"
	}

	table, err := g.WrapTable(b.GetTable())
	if err != nil {
		return "", nil, err
	}

	sql.WriteString("SELECT ")
	sql.WriteString(countExpr)
	sql.WriteString(" FROM ")
	sql.WriteString(table)

	// WHERE
	wheres := b.GetWheres()
	if len(wheres) > 0 {
		whereSQL, whereArgs, err := g.compileWheres(wheres)
		if err != nil {
			return "", nil, err
		}
		sql.WriteString(" WHERE ")
		sql.WriteString(whereSQL)
		args = append(args, whereArgs...)
	}

	return sql.String(), args, nil
}

// CompileAggregate compiles an aggregate function query.
func (g *MySQLGrammar) CompileAggregate(b *Builder, fn, column string) (string, []any, error) {
	if b.GetTable() == "" {
		return "", nil, ErrNoTable
	}
	if column == "" {
		return "", nil, ErrNoColumns
	}

	var sql strings.Builder
	args := make([]any, 0)

	wrapped, err := g.Wrap(column)
	if err != nil {
		return "", nil, err
	}

	table, err := g.WrapTable(b.GetTable())
	if err != nil {
		return "", nil, err
	}

	sql.WriteString("SELECT ")
	sql.WriteString(strings.ToUpper(fn))
	sql.WriteString("(")
	sql.WriteString(wrapped)
	sql.WriteString(") FROM ")
	sql.WriteString(table)

	// WHERE
	wheres := b.GetWheres()
	if len(wheres) > 0 {
		whereSQL, whereArgs, err := g.compileWheres(wheres)
		if err != nil {
			return "", nil, err
		}
		sql.WriteString(" WHERE ")
		sql.WriteString(whereSQL)
		args = append(args, whereArgs...)
	}

	return sql.String(), args, nil
}

// CompileTruncate compiles a TRUNCATE TABLE statement.
func (g *MySQLGrammar) CompileTruncate(b *Builder) (string, error) {
	if b.GetTable() == "" {
		return "", ErrNoTable
	}

	table, err := g.WrapTable(b.GetTable())
	if err != nil {
		return "", err
	}

	return "TRUNCATE TABLE " + table, nil
}

// CompileUpsert compiles an INSERT ... ON DUPLICATE KEY UPDATE query.
func (g *MySQLGrammar) CompileUpsert(b *Builder, data map[string]any, updateColumns []string) (string, []any, error) {
	// First compile the INSERT part
	insertSQL, args, err := g.CompileInsert(b, data)
	if err != nil {
		return "", nil, err
	}

	// Add ON DUPLICATE KEY UPDATE
	var sql strings.Builder
	sql.WriteString(insertSQL)
	sql.WriteString(" ON DUPLICATE KEY UPDATE ")

	// Determine which columns to update
	if len(updateColumns) == 0 {
		// Update all columns
		updateColumns = make([]string, 0, len(data))
		for k := range data {
			updateColumns = append(updateColumns, k)
		}
		sort.Strings(updateColumns)
	}

	updateParts := make([]string, len(updateColumns))
	for i, col := range updateColumns {
		wrapped, err := g.Wrap(col)
		if err != nil {
			return "", nil, err
		}
		updateParts[i] = wrapped + " = VALUES(" + wrapped + ")"
	}
	sql.WriteString(strings.Join(updateParts, ", "))

	return sql.String(), args, nil
}

// compileWheres compiles WHERE clauses.
func (g *MySQLGrammar) compileWheres(wheres []WhereClause) (string, []any, error) {
	if len(wheres) == 0 {
		return "", nil, nil
	}

	var sql strings.Builder
	args := make([]any, 0)

	for i, where := range wheres {
		// Add boolean connector
		if i > 0 {
			sql.WriteString(" ")
			sql.WriteString(where.Boolean.String())
			sql.WriteString(" ")
		}

		clauseSQL, clauseArgs, err := g.compileWhere(where)
		if err != nil {
			return "", nil, err
		}
		sql.WriteString(clauseSQL)
		args = append(args, clauseArgs...)
	}

	return sql.String(), args, nil
}

// compileWhere compiles a single WHERE clause.
func (g *MySQLGrammar) compileWhere(where WhereClause) (string, []any, error) {
	switch where.Type {
	case WhereTypeBasic:
		return g.compileWhereBasic(where)
	case WhereTypeIn:
		return g.compileWhereIn(where, false)
	case WhereTypeNotIn:
		return g.compileWhereIn(where, true)
	case WhereTypeBetween:
		return g.compileWhereBetween(where, false)
	case WhereTypeNotBetween:
		return g.compileWhereBetween(where, true)
	case WhereTypeNull:
		return g.compileWhereNull(where, false)
	case WhereTypeNotNull:
		return g.compileWhereNull(where, true)
	case WhereTypeRaw:
		return where.Raw, where.Bindings, nil
	case WhereTypeNested:
		return g.compileWhereNested(where)
	case WhereTypeDate:
		return g.compileWhereDate(where, "DATE")
	case WhereTypeYear:
		return g.compileWhereDate(where, "YEAR")
	case WhereTypeMonth:
		return g.compileWhereDate(where, "MONTH")
	case WhereTypeDay:
		return g.compileWhereDate(where, "DAY")
	default:
		return "", nil, fmt.Errorf("unknown where type: %v", where.Type)
	}
}

func (g *MySQLGrammar) compileWhereBasic(where WhereClause) (string, []any, error) {
	column, err := g.Wrap(where.Column)
	if err != nil {
		return "", nil, err
	}

	if err := validation.ValidateOperator(where.Operator); err != nil {
		return "", nil, err
	}

	return column + " " + strings.ToUpper(where.Operator) + " ?", []any{where.Value}, nil
}

func (g *MySQLGrammar) compileWhereIn(where WhereClause, not bool) (string, []any, error) {
	if len(where.Values) == 0 {
		return "", nil, ErrEmptyWhereIn
	}

	column, err := g.Wrap(where.Column)
	if err != nil {
		return "", nil, err
	}

	placeholders := make([]string, len(where.Values))
	for i := range where.Values {
		placeholders[i] = "?"
	}

	op := "IN"
	if not {
		op = "NOT IN"
	}

	return column + " " + op + " (" + strings.Join(placeholders, ", ") + ")", where.Values, nil
}

func (g *MySQLGrammar) compileWhereBetween(where WhereClause, not bool) (string, []any, error) {
	if len(where.Values) != 2 {
		return "", nil, ErrInvalidBetweenValues
	}

	column, err := g.Wrap(where.Column)
	if err != nil {
		return "", nil, err
	}

	op := "BETWEEN"
	if not {
		op = "NOT BETWEEN"
	}

	return column + " " + op + " ? AND ?", where.Values, nil
}

func (g *MySQLGrammar) compileWhereNull(where WhereClause, not bool) (string, []any, error) {
	column, err := g.Wrap(where.Column)
	if err != nil {
		return "", nil, err
	}

	op := "IS NULL"
	if not {
		op = "IS NOT NULL"
	}

	return column + " " + op, nil, nil
}

func (g *MySQLGrammar) compileWhereNested(where WhereClause) (string, []any, error) {
	if len(where.Nested) == 0 {
		return "", nil, nil
	}

	nestedSQL, args, err := g.compileWheres(where.Nested)
	if err != nil {
		return "", nil, err
	}

	return "(" + nestedSQL + ")", args, nil
}

func (g *MySQLGrammar) compileWhereDate(where WhereClause, fn string) (string, []any, error) {
	column, err := g.Wrap(where.Column)
	if err != nil {
		return "", nil, err
	}

	return fn + "(" + column + ") = ?", []any{where.Value}, nil
}

func (g *MySQLGrammar) compileJoin(join JoinClause) (string, error) {
	table, err := g.WrapTable(join.Table)
	if err != nil {
		return "", err
	}

	if join.Type == JoinCross {
		return "CROSS JOIN " + table, nil
	}

	first, err := g.Wrap(join.First)
	if err != nil {
		return "", err
	}

	second, err := g.Wrap(join.Second)
	if err != nil {
		return "", err
	}

	if err := validation.ValidateOperator(join.Operator); err != nil {
		return "", err
	}

	return string(join.Type) + " JOIN " + table + " ON " + first + " " + join.Operator + " " + second, nil
}
