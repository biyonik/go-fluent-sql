package fluentsql

import (
	"context"
	"database/sql"

	"github.com/biyonik/go-fluent-sql/dialect"
)

// Builder, SQL sorgularını akıcı bir arayüz (fluent interface) ile oluşturmak için kullanılan ana sınıftır.
//
// Bu sınıf, Laravel ve Symfony'nin Query Builder mantığından esinlenmiştir.
// Sorguların neyi, nasıl ve neden yaptığına dair okunabilir ve bakımı kolay bir yapı sunar.
// Builder örnekleri **concurrent-safe** değildir; paralel kullanımlar için Clone() ile çoğaltılmalıdır.
//
// Genel kullanım örneği:
//
//	var users []User
//	err := db.Table("users").
//	    Select("id", "name", "email").
//	    Where("status", "=", "active").
//	    OrderByDesc("created_at").
//	    Limit(10).
//	    GetContext(ctx, &users)
//
// Bu sınıf, SELECT, INSERT, UPDATE, DELETE işlemlerini destekler ve sorgu bileşenlerini kolayca birleştirmeyi sağlar.
//
// @author Ahmet ALTUN
// @github github.com/biyonik
// @linkedin linkedin.com/in/biyonik
// @email ahmet.altun60@gmail.com
type Builder struct {
	executor QueryExecutor
	grammar  dialect.Grammar
	scanner  Scanner

	// Table name and alias
	table      string
	tableAlias string

	// Selected columns
	columns  []string
	distinct bool

	// Query clauses
	wheres  []dialect.WhereClause
	orders  []dialect.OrderClause
	joins   []dialect.JoinClause
	groupBy []string
	having  []dialect.WhereClause

	// Limits
	limit  *int
	offset *int

	// Accumulated error
	err error
}

// NewBuilder, belirtilen executor, grammar ve scanner ile yeni bir Builder oluşturur.
func NewBuilder(executor QueryExecutor, grammar dialect.Grammar, scanner Scanner) *Builder {
	return &Builder{
		executor: executor,
		grammar:  grammar,
		scanner:  scanner,
		columns:  make([]string, 0),
		wheres:   make([]dialect.WhereClause, 0),
		orders:   make([]dialect.OrderClause, 0),
		joins:    make([]dialect.JoinClause, 0),
		groupBy:  make([]string, 0),
		having:   make([]dialect.WhereClause, 0),
	}
}

// Table, sorguda kullanılacak tablo adını ayarlar.
func (b *Builder) Table(name string) *Builder {
	b.table = name
	return b
}

// TableAs, tablo adı ve aliasını ayarlar.
func (b *Builder) TableAs(name, alias string) *Builder {
	b.table = name
	b.tableAlias = alias
	return b
}

// From, Table için okunabilir alias sağlar.
func (b *Builder) From(name string) *Builder {
	return b.Table(name)
}

// Select, sorguda seçilecek kolonları ayarlar.
func (b *Builder) Select(columns ...string) *Builder {
	b.columns = append(b.columns, columns...)
	return b
}

// SelectRaw, ham SQL select ifadesi ekler.
// Dikkat: SQL injection riskine karşı ifadeyi güvenli şekilde kullanın.
func (b *Builder) SelectRaw(expr string) *Builder {
	b.columns = append(b.columns, expr)
	return b
}

// Distinct, sorguyu DISTINCT olarak işaretler.
func (b *Builder) Distinct() *Builder {
	b.distinct = true
	return b
}

// Where, temel bir WHERE koşulu ekler.
func (b *Builder) Where(column, operator string, value any) *Builder {
	b.wheres = append(b.wheres, dialect.WhereClause{
		Type:     dialect.WhereTypeBasic,
		Boolean:  dialect.WhereBooleanAnd,
		Column:   column,
		Operator: operator,
		Value:    value,
	})
	return b
}

// OrWhere, OR WHERE koşulu ekler.
func (b *Builder) OrWhere(column, operator string, value any) *Builder {
	b.wheres = append(b.wheres, dialect.WhereClause{
		Type:     dialect.WhereTypeBasic,
		Boolean:  dialect.WhereBooleanOr,
		Column:   column,
		Operator: operator,
		Value:    value,
	})
	return b
}

// WhereIn, WHERE IN koşulu ekler.
func (b *Builder) WhereIn(column string, values []any) *Builder {
	b.wheres = append(b.wheres, dialect.WhereClause{
		Type:    dialect.WhereTypeIn,
		Boolean: dialect.WhereBooleanAnd,
		Column:  column,
		Values:  values,
	})
	return b
}

// WhereNotIn, WHERE NOT IN koşulu ekler.
func (b *Builder) WhereNotIn(column string, values []any) *Builder {
	b.wheres = append(b.wheres, dialect.WhereClause{
		Type:    dialect.WhereTypeNotIn,
		Boolean: dialect.WhereBooleanAnd,
		Column:  column,
		Values:  values,
	})
	return b
}

// OrWhereIn, OR WHERE IN koşulu ekler.
func (b *Builder) OrWhereIn(column string, values []any) *Builder {
	b.wheres = append(b.wheres, dialect.WhereClause{
		Type:    dialect.WhereTypeIn,
		Boolean: dialect.WhereBooleanOr,
		Column:  column,
		Values:  values,
	})
	return b
}

// OrWhereNotIn, OR WHERE NOT IN koşulu ekler.
func (b *Builder) OrWhereNotIn(column string, values []any) *Builder {
	b.wheres = append(b.wheres, dialect.WhereClause{
		Type:    dialect.WhereTypeNotIn,
		Boolean: dialect.WhereBooleanOr,
		Column:  column,
		Values:  values,
	})
	return b
}

// WhereBetween, WHERE BETWEEN koşulu ekler.
func (b *Builder) WhereBetween(column string, min, max any) *Builder {
	b.wheres = append(b.wheres, dialect.WhereClause{
		Type:    dialect.WhereTypeBetween,
		Boolean: dialect.WhereBooleanAnd,
		Column:  column,
		Values:  []any{min, max},
	})
	return b
}

// WhereNotBetween, WHERE NOT BETWEEN koşulu ekler.
func (b *Builder) WhereNotBetween(column string, min, max any) *Builder {
	b.wheres = append(b.wheres, dialect.WhereClause{
		Type:    dialect.WhereTypeNotBetween,
		Boolean: dialect.WhereBooleanAnd,
		Column:  column,
		Values:  []any{min, max},
	})
	return b
}

// WhereNull, WHERE IS NULL koşulu ekler.
func (b *Builder) WhereNull(column string) *Builder {
	b.wheres = append(b.wheres, dialect.WhereClause{
		Type:    dialect.WhereTypeNull,
		Boolean: dialect.WhereBooleanAnd,
		Column:  column,
	})
	return b
}

// WhereNotNull, WHERE IS NOT NULL koşulu ekler.
func (b *Builder) WhereNotNull(column string) *Builder {
	b.wheres = append(b.wheres, dialect.WhereClause{
		Type:    dialect.WhereTypeNotNull,
		Boolean: dialect.WhereBooleanAnd,
		Column:  column,
	})
	return b
}

// OrWhereNull, OR WHERE IS NULL koşulu ekler.
func (b *Builder) OrWhereNull(column string) *Builder {
	b.wheres = append(b.wheres, dialect.WhereClause{
		Type:    dialect.WhereTypeNull,
		Boolean: dialect.WhereBooleanOr,
		Column:  column,
	})
	return b
}

// OrWhereNotNull, OR WHERE IS NOT NULL koşulu ekler.
func (b *Builder) OrWhereNotNull(column string) *Builder {
	b.wheres = append(b.wheres, dialect.WhereClause{
		Type:    dialect.WhereTypeNotNull,
		Boolean: dialect.WhereBooleanOr,
		Column:  column,
	})
	return b
}

// WhereLike, WHERE LIKE koşulu ekler.
func (b *Builder) WhereLike(column string, pattern string) *Builder {
	b.wheres = append(b.wheres, dialect.WhereClause{
		Type:     dialect.WhereTypeBasic,
		Boolean:  dialect.WhereBooleanAnd,
		Column:   column,
		Operator: "LIKE",
		Value:    pattern,
	})
	return b
}

// WhereNotLike, WHERE NOT LIKE koşulu ekler.
func (b *Builder) WhereNotLike(column string, pattern string) *Builder {
	b.wheres = append(b.wheres, dialect.WhereClause{
		Type:     dialect.WhereTypeBasic,
		Boolean:  dialect.WhereBooleanAnd,
		Column:   column,
		Operator: "NOT LIKE",
		Value:    pattern,
	})
	return b
}

// WhereRaw, ham SQL WHERE ifadesi ekler.
func (b *Builder) WhereRaw(sqlExpr string, bindings ...any) *Builder {
	b.wheres = append(b.wheres, dialect.WhereClause{
		Type:     dialect.WhereTypeRaw,
		Boolean:  dialect.WhereBooleanAnd,
		Raw:      sqlExpr,
		Bindings: bindings,
	})
	return b
}

// OrWhereRaw, ham SQL OR WHERE ifadesi ekler.
func (b *Builder) OrWhereRaw(sqlExpr string, bindings ...any) *Builder {
	b.wheres = append(b.wheres, dialect.WhereClause{
		Type:     dialect.WhereTypeRaw,
		Boolean:  dialect.WhereBooleanOr,
		Raw:      sqlExpr,
		Bindings: bindings,
	})
	return b
}

// WhereNested, iç içe WHERE bloğu ekler.
func (b *Builder) WhereNested(fn func(*Builder)) *Builder {
	nested := NewBuilder(b.executor, b.grammar, b.scanner)
	fn(nested)
	b.wheres = append(b.wheres, dialect.WhereClause{
		Type:    dialect.WhereTypeNested,
		Boolean: dialect.WhereBooleanAnd,
		Nested:  nested.wheres,
	})
	return b
}

// OrWhereNested, OR iç içe WHERE bloğu ekler.
func (b *Builder) OrWhereNested(fn func(*Builder)) *Builder {
	nested := NewBuilder(b.executor, b.grammar, b.scanner)
	fn(nested)
	b.wheres = append(b.wheres, dialect.WhereClause{
		Type:    dialect.WhereTypeNested,
		Boolean: dialect.WhereBooleanOr,
		Nested:  nested.wheres,
	})
	return b
}

// WhereDate, DATE(column) = value koşulu ekler.
func (b *Builder) WhereDate(column string, value string) *Builder {
	b.wheres = append(b.wheres, dialect.WhereClause{
		Type:    dialect.WhereTypeDate,
		Boolean: dialect.WhereBooleanAnd,
		Column:  column,
		Value:   value,
	})
	return b
}

// WhereYear, YEAR(column) = value koşulu ekler.
func (b *Builder) WhereYear(column string, value int) *Builder {
	b.wheres = append(b.wheres, dialect.WhereClause{
		Type:    dialect.WhereTypeYear,
		Boolean: dialect.WhereBooleanAnd,
		Column:  column,
		Value:   value,
	})
	return b
}

// WhereMonth, MONTH(column) = value koşulu ekler.
func (b *Builder) WhereMonth(column string, value int) *Builder {
	b.wheres = append(b.wheres, dialect.WhereClause{
		Type:    dialect.WhereTypeMonth,
		Boolean: dialect.WhereBooleanAnd,
		Column:  column,
		Value:   value,
	})
	return b
}

// WhereDay, DAY(column) = value koşulu ekler.
func (b *Builder) WhereDay(column string, value int) *Builder {
	b.wheres = append(b.wheres, dialect.WhereClause{
		Type:    dialect.WhereTypeDay,
		Boolean: dialect.WhereBooleanAnd,
		Column:  column,
		Value:   value,
	})
	return b
}

// Join, INNER JOIN ekler.
func (b *Builder) Join(table, first, operator, second string) *Builder {
	b.joins = append(b.joins, dialect.JoinClause{
		Type:     dialect.JoinInner,
		Table:    table,
		First:    first,
		Operator: operator,
		Second:   second,
	})
	return b
}

// LeftJoin, LEFT JOIN ekler.
func (b *Builder) LeftJoin(table, first, operator, second string) *Builder {
	b.joins = append(b.joins, dialect.JoinClause{
		Type:     dialect.JoinLeft,
		Table:    table,
		First:    first,
		Operator: operator,
		Second:   second,
	})
	return b
}

// RightJoin, RIGHT JOIN ekler.
func (b *Builder) RightJoin(table, first, operator, second string) *Builder {
	b.joins = append(b.joins, dialect.JoinClause{
		Type:     dialect.JoinRight,
		Table:    table,
		First:    first,
		Operator: operator,
		Second:   second,
	})
	return b
}

// CrossJoin, CROSS JOIN ekler.
func (b *Builder) CrossJoin(table string) *Builder {
	b.joins = append(b.joins, dialect.JoinClause{
		Type:  dialect.JoinCross,
		Table: table,
	})
	return b
}

// OrderBy, ORDER BY ekler.
func (b *Builder) OrderBy(column string, direction dialect.OrderDirection) *Builder {
	b.orders = append(b.orders, dialect.OrderClause{
		Column:    column,
		Direction: direction,
	})
	return b
}

// OrderByAsc, artan sırada ORDER BY ekler.
func (b *Builder) OrderByAsc(column string) *Builder {
	return b.OrderBy(column, dialect.OrderAsc)
}

// OrderByDesc, azalan sırada ORDER BY ekler.
func (b *Builder) OrderByDesc(column string) *Builder {
	return b.OrderBy(column, dialect.OrderDesc)
}

// OrderByRaw, ham ORDER BY ifadesi ekler.
func (b *Builder) OrderByRaw(expr string) *Builder {
	b.orders = append(b.orders, dialect.OrderClause{
		Raw: expr,
	})
	return b
}

// Latest, created_at DESC ile sıralar.
func (b *Builder) Latest() *Builder {
	return b.OrderByDesc("created_at")
}

// Oldest, created_at ASC ile sıralar.
func (b *Builder) Oldest() *Builder {
	return b.OrderByAsc("created_at")
}

// GroupBy, GROUP BY ekler.
func (b *Builder) GroupBy(columns ...string) *Builder {
	b.groupBy = append(b.groupBy, columns...)
	return b
}

// Having, HAVING ekler.
func (b *Builder) Having(column, operator string, value any) *Builder {
	b.having = append(b.having, dialect.WhereClause{
		Type:     dialect.WhereTypeBasic,
		Boolean:  dialect.WhereBooleanAnd,
		Column:   column,
		Operator: operator,
		Value:    value,
	})
	return b
}

// HavingRaw, ham HAVING ifadesi ekler.
func (b *Builder) HavingRaw(sqlExpr string, bindings ...any) *Builder {
	b.having = append(b.having, dialect.WhereClause{
		Type:     dialect.WhereTypeRaw,
		Boolean:  dialect.WhereBooleanAnd,
		Raw:      sqlExpr,
		Bindings: bindings,
	})
	return b
}

// Limit, LIMIT ekler.
func (b *Builder) Limit(n int) *Builder {
	b.limit = &n
	return b
}

// Offset, OFFSET ekler.
func (b *Builder) Offset(n int) *Builder {
	b.offset = &n
	return b
}

// Take, Limit aliasıdır.
func (b *Builder) Take(n int) *Builder {
	return b.Limit(n)
}

// Skip, Offset aliasıdır.
func (b *Builder) Skip(n int) *Builder {
	return b.Offset(n)
}

// ForPage, sayfa bazlı limit ve offset belirler.
func (b *Builder) ForPage(page, perPage int) *Builder {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * perPage
	return b.Limit(perPage).Offset(offset)
}

// ToSQL, sorguyu SQL string ve bindinglerle döndürür.
func (b *Builder) ToSQL() (string, []any, error) {
	return b.ToSelectSQL()
}

// ToSelectSQL, SELECT sorgusunu derler.
func (b *Builder) ToSelectSQL() (string, []any, error) {
	if b.err != nil {
		return "", nil, b.err
	}
	if b.grammar == nil {
		return "", nil, ErrNoExecutor
	}
	return b.grammar.CompileSelect(b)
}

// ToInsertSQL, INSERT sorgusunu derler.
func (b *Builder) ToInsertSQL(data map[string]any) (string, []any, error) {
	if b.err != nil {
		return "", nil, b.err
	}
	if b.grammar == nil {
		return "", nil, ErrNoExecutor
	}
	return b.grammar.CompileInsert(b, data)
}

// ToUpdateSQL, UPDATE sorgusunu derler.
func (b *Builder) ToUpdateSQL(data map[string]any) (string, []any, error) {
	if b.err != nil {
		return "", nil, b.err
	}
	if b.grammar == nil {
		return "", nil, ErrNoExecutor
	}
	return b.grammar.CompileUpdate(b, data)
}

// ToDeleteSQL, DELETE sorgusunu derler.
func (b *Builder) ToDeleteSQL() (string, []any, error) {
	if b.err != nil {
		return "", nil, b.err
	}
	if b.grammar == nil {
		return "", nil, ErrNoExecutor
	}
	return b.grammar.CompileDelete(b)
}

// Clone, Builder'ın derin kopyasını oluşturur.
func (b *Builder) Clone() *Builder {
	clone := &Builder{
		executor:   b.executor,
		grammar:    b.grammar,
		scanner:    b.scanner,
		table:      b.table,
		tableAlias: b.tableAlias,
		distinct:   b.distinct,
		limit:      b.limit,
		offset:     b.offset,
		err:        b.err,
	}

	clone.columns = make([]string, len(b.columns))
	copy(clone.columns, b.columns)

	clone.wheres = make([]dialect.WhereClause, len(b.wheres))
	copy(clone.wheres, b.wheres)

	clone.orders = make([]dialect.OrderClause, len(b.orders))
	copy(clone.orders, b.orders)

	clone.joins = make([]dialect.JoinClause, len(b.joins))
	copy(clone.joins, b.joins)

	clone.groupBy = make([]string, len(b.groupBy))
	copy(clone.groupBy, b.groupBy)

	clone.having = make([]dialect.WhereClause, len(b.having))
	copy(clone.having, b.having)

	return clone
}

// Reset, tüm sorgu durumunu temizler (connection ve grammar hariç).
func (b *Builder) Reset() *Builder {
	b.table = ""
	b.tableAlias = ""
	b.columns = make([]string, 0)
	b.distinct = false
	b.wheres = make([]dialect.WhereClause, 0)
	b.orders = make([]dialect.OrderClause, 0)
	b.joins = make([]dialect.JoinClause, 0)
	b.groupBy = make([]string, 0)
	b.having = make([]dialect.WhereClause, 0)
	b.limit = nil
	b.offset = nil
	b.err = nil
	return b
}

// Err, birikmiş hatayı döndürür.
func (b *Builder) Err() error {
	return b.err
}

// When, koşullu olarak callback uygular.
func (b *Builder) When(condition bool, fn func(*Builder)) *Builder {
	if condition {
		fn(b)
	}
	return b
}

// Unless, When’in tersidir.
func (b *Builder) Unless(condition bool, fn func(*Builder)) *Builder {
	return b.When(!condition, fn)
}

// GetTable, tablo adını döndürür.
func (b *Builder) GetTable() string {
	return b.table
}

// GetTableAlias, tablo aliasını döndürür.
func (b *Builder) GetTableAlias() string {
	return b.tableAlias
}

// GetColumns, seçilen kolonları döndürür.
func (b *Builder) GetColumns() []string {
	return b.columns
}

// IsDistinct, DISTINCT kullanılıp kullanılmadığını döndürür.
func (b *Builder) IsDistinct() bool {
	return b.distinct
}

// GetWheres, WHERE koşullarını döndürür.
func (b *Builder) GetWheres() []dialect.WhereClause {
	return b.wheres
}

// GetOrders, ORDER BY koşullarını döndürür.
func (b *Builder) GetOrders() []dialect.OrderClause {
	return b.orders
}

// GetJoins, JOIN koşullarını döndürür.
func (b *Builder) GetJoins() []dialect.JoinClause {
	return b.joins
}

// GetGroupBy, GROUP BY kolonlarını döndürür.
func (b *Builder) GetGroupBy() []string {
	return b.groupBy
}

// GetHaving, HAVING koşullarını döndürür.
func (b *Builder) GetHaving() []dialect.WhereClause {
	return b.having
}

// GetLimit, LIMIT değerini döndürür.
func (b *Builder) GetLimit() *int {
	return b.limit
}

// GetOffset, OFFSET değerini döndürür.
func (b *Builder) GetOffset() *int {
	return b.offset
}

// GetContext, sorguyu çalıştırır ve sonuçları dest içine tarar.
func (b *Builder) GetContext(ctx context.Context, dest any) error {
	if b.executor == nil {
		return ErrNoExecutor
	}

	sqlStr, args, err := b.ToSelectSQL()
	if err != nil {
		return err
	}

	rows, err := b.executor.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return NewQueryError("select", b.table, sqlStr, err)
	}
	defer rows.Close()

	return b.scanner.ScanRows(rows, dest)
}

// Get, GetContext’in context.Background() versiyonudur.
func (b *Builder) Get(dest any) error {
	return b.GetContext(context.Background(), dest)
}

// FirstContext, LIMIT 1 ile sorguyu çalıştırır.
func (b *Builder) FirstContext(ctx context.Context, dest any) error {
	b.Limit(1)

	if b.executor == nil {
		return ErrNoExecutor
	}

	sqlStr, args, err := b.ToSelectSQL()
	if err != nil {
		return err
	}

	row := b.executor.QueryRowContext(ctx, sqlStr, args...)
	return b.scanner.ScanRow(row, dest)
}

// First, FirstContext’in context.Background() versiyonudur.
func (b *Builder) First(dest any) error {
	return b.FirstContext(context.Background(), dest)
}

// InsertContext, INSERT sorgusu çalıştırır.
func (b *Builder) InsertContext(ctx context.Context, data map[string]any) (*QueryResult, error) {
	if b.executor == nil {
		return nil, ErrNoExecutor
	}

	sqlStr, args, err := b.ToInsertSQL(data)
	if err != nil {
		return nil, err
	}

	result, err := b.executor.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, NewQueryError("insert", b.table, sqlStr, err)
	}

	return NewQueryResult(result), nil
}

// Insert, InsertContext’in context.Background() versiyonudur.
func (b *Builder) Insert(data map[string]any) (*QueryResult, error) {
	return b.InsertContext(context.Background(), data)
}

// UpdateContext, UPDATE sorgusu çalıştırır.
func (b *Builder) UpdateContext(ctx context.Context, data map[string]any) (*QueryResult, error) {
	if b.executor == nil {
		return nil, ErrNoExecutor
	}

	sqlStr, args, err := b.ToUpdateSQL(data)
	if err != nil {
		return nil, err
	}

	result, err := b.executor.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, NewQueryError("update", b.table, sqlStr, err)
	}

	return NewQueryResult(result), nil
}

// Update, UpdateContext’in context.Background() versiyonudur.
func (b *Builder) Update(data map[string]any) (*QueryResult, error) {
	return b.UpdateContext(context.Background(), data)
}

// DeleteContext, DELETE sorgusu çalıştırır.
func (b *Builder) DeleteContext(ctx context.Context) (*QueryResult, error) {
	if b.executor == nil {
		return nil, ErrNoExecutor
	}

	sqlStr, args, err := b.ToDeleteSQL()
	if err != nil {
		return nil, err
	}

	result, err := b.executor.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, NewQueryError("delete", b.table, sqlStr, err)
	}

	return NewQueryResult(result), nil
}

// Delete, DeleteContext’in context.Background() versiyonudur.
func (b *Builder) Delete() (*QueryResult, error) {
	return b.DeleteContext(context.Background())
}

// CountContext, sorgu için toplam satır sayısını döndürür.
func (b *Builder) CountContext(ctx context.Context) (int64, error) {
	if b.executor == nil {
		return 0, ErrNoExecutor
	}

	sqlStr, args, err := b.grammar.CompileCount(b, "")
	if err != nil {
		return 0, err
	}

	var count int64
	row := b.executor.QueryRowContext(ctx, sqlStr, args...)
	if err := row.Scan(&count); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, NewQueryError("count", b.table, sqlStr, err)
	}

	return count, nil
}

// Count, CountContext’in context.Background() versiyonudur.
func (b *Builder) Count() (int64, error) {
	return b.CountContext(context.Background())
}

// ExistsContext, sorguda herhangi bir satır var mı kontrol eder.
func (b *Builder) ExistsContext(ctx context.Context) (bool, error) {
	if b.executor == nil {
		return false, ErrNoExecutor
	}

	sqlStr, args, err := b.grammar.CompileExists(b)
	if err != nil {
		return false, err
	}

	var exists bool
	row := b.executor.QueryRowContext(ctx, sqlStr, args...)
	if err := row.Scan(&exists); err != nil {
		return false, NewQueryError("exists", b.table, sqlStr, err)
	}

	return exists, nil
}

// Exists, ExistsContext’in context.Background() versiyonudur.
func (b *Builder) Exists() (bool, error) {
	return b.ExistsContext(context.Background())
}

// DoesntExistContext, sorguda hiçbir satır yoksa true döner.
func (b *Builder) DoesntExistContext(ctx context.Context) (bool, error) {
	exists, err := b.ExistsContext(ctx)
	return !exists, err
}

// DoesntExist, DoesntExistContext’in context.Background() versiyonudur.
func (b *Builder) DoesntExist() (bool, error) {
	return b.DoesntExistContext(context.Background())
}
