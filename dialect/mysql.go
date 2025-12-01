package dialect

import (
	"fmt"
	"sort"
	"strings"

	"github.com/biyonik/go-fluent-sql/internal/validation"
)

/*
 * ----------------------------------------------------------------------------
 * MYSQL GRAMMAR IMPLEMENTATION
 * ----------------------------------------------------------------------------
 *
 * Bu dosya, FluentSQL'in soyut sorgu yapısını (Query Builder) saf ve çalıştırılabilir
 * MySQL/MariaDB SQL dizelerine dönüştüren "çevirmen" (translator) katmanıdır.
 *
 * SQL dünyasında standartlar olsa da (ANSI SQL), her veritabanı motoru kendi
 * kurallarına, tırnaklama stillerine ve özel fonksiyonlarına sahiptir.
 * Örneğin; PostgreSQL çift tırnak (") kullanırken, MySQL backtick (`) kullanır.
 *
 * Bu sınıfın sorumlulukları:
 * 1. Sanitization (Temizleme): Tablo ve kolon isimlerini rezerve kelimelerle
 * (örn: "order", "group") çakışmaması için sarmalar.
 * 2. Compilation (Derleme): SELECT, INSERT, UPDATE, DELETE gibi operasyonları
 * doğru sözdizimi sırasıyla (SELECT -> FROM -> WHERE -> ORDER) inşa eder.
 * 3. Optimization (Optimizasyon): Batch Insert ve Upsert gibi MySQL'e özgü
 * performans özelliklerini destekler.
 *
 * @author Ahmet ALTUN
 * @github github.com/biyonik
 * @linkedin linkedin.com/in/biyonik
 * @email ahmet.altun60@gmail.com
 * ----------------------------------------------------------------------------
 */

// MySQLGrammar, Grammar arayüzünü MySQL ve MariaDB veritabanları için implemente eder.
//
// Bu yapı, temel dilbilgisi kurallarını (BaseGrammar) devralır ve üzerine
// MySQL'e özgü davranışları (parametre yer tutucuları, tırnaklama stili vb.) ekler.
type MySQLGrammar struct {
	BaseGrammar
}

// MySQL, yeni bir MySQL dilbilgisi örneği oluşturur.
//
// Varsayılan tarih formatı ve sürücü isimlendirmesi burada yapılandırılır.
// Bu metot genellikle Driver Factory tarafından çağrılır.
func MySQL() *MySQLGrammar {
	return &MySQLGrammar{
		BaseGrammar: BaseGrammar{
			name:       "mysql",
			dateFormat: "2006-01-02 15:04:05",
		},
	}
}

// NewMySQLGrammar, geriye dönük uyumluluk (backward compatibility) için
// MySQL() kurucusuna (constructor) verilen bir takma addır.
func NewMySQLGrammar() *MySQLGrammar {
	return MySQL()
}

// Wrap, bir veritabanı tanımlayıcısını (tablo veya kolon adı) MySQL standartlarına
// uygun kaçış karakterleriyle (backtick) sarmalar.
//
// Bu işlem iki kritik sorunu çözer:
//  1. SQL Injection güvenliği sağlar.
//  2. Rezerve edilmiş kelimelerin (örn: `key`, `index`) kolon adı olarak
//     kullanılabilmesine olanak tanır.
//
// Örnek: "users.name" -> "`users`.`name`"
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

// WrapTable, tablo ismini ve varsa takma adını (alias) güvenli bir şekilde sarmalar.
//
// Query Builder içinde "users as u" şeklinde tanımlanan tabloları
// MySQL'in anlayacağı "`users` AS `u`" formatına dönüştürür.
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

// WrapValue, bir kolon değer referansını sarmalar.
// MySQLGrammar içinde Wrap metodu ile aynı işlevi görür.
func (g *MySQLGrammar) WrapValue(value string) (string, error) {
	return g.Wrap(value)
}

// Placeholder, sorgu parametreleri için kullanılan yer tutucuyu döndürür.
//
// PostgreSQL ($1, $2) aksine, MySQL sıralı soru işareti (?) kullanır.
// Index parametresi MySQL için önemsizdir ancak arayüz uyumluluğu için tutulur.
func (g *MySQLGrammar) Placeholder(index int) string {
	return "?"
}

// CompileSelect, bir SELECT sorgusunu parçalarından (components) birleştirerek inşa eder.
//
// Bu metot bir montaj hattı gibi çalışır; her bir SQL parçası (columns, joins, wheres)
// sırasıyla işlenir ve string builder üzerinde birleştirilir.
// Karmaşık mantık (örn: Raw expression kontrolü) burada yönetilir.
func (g *MySQLGrammar) CompileSelect(b QueryBuilder) (string, []any, error) {
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

// CompileInsert, tekil bir kayıt ekleme sorgusu oluşturur.
//
// Map yapısındaki veriyi alır, anahtarları alfabetik sıralar (deterministik test edilebilirlik için)
// ve "INSERT INTO table (col1, col2) VALUES (?, ?)" formatında hazırlar.
func (g *MySQLGrammar) CompileInsert(b QueryBuilder, data map[string]any) (string, []any, error) {
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

// CompileInsertBatch, tek bir sorguda çoklu kayıt (bulk insert) ekleme işlemi oluşturur.
//
// Veritabanı ile yapılan "round-trip" sayısını azalttığı için performans açısından
// kritik bir metottur. 1000 kayıt için 1000 ayrı sorgu atmak yerine,
// tek bir sorguda hepsini gönderir.
//
// Dikkat: Tüm satırların aynı anahtarlara (kolonlara) sahip olduğu varsayılır.
func (g *MySQLGrammar) CompileInsertBatch(b QueryBuilder, data []map[string]any) (string, []any, error) {
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

// CompileUpdate, mevcut kayıtları güncellemek için UPDATE sorgusu oluşturur.
//
// SET bloğunu oluştururken, parametrik yapı (prepared statements) kullanılarak
// SQL Injection riski elimine edilir.
func (g *MySQLGrammar) CompileUpdate(b QueryBuilder, data map[string]any) (string, []any, error) {
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

// CompileDelete, kayıt silme sorgusu (DELETE) oluşturur.
//
// WHERE koşulları eklenerek, tüm tablonun yanlışlıkla silinmesi (truncate etkisi)
// engellenir (tabii geliştirici WHERE eklemeyi unutmazsa).
func (g *MySQLGrammar) CompileDelete(b QueryBuilder) (string, []any, error) {
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

// CompileExists, bir kaydın varlığını kontrol etmek için optimize edilmiş bir sorgu oluşturur.
//
// "SELECT * FROM" yerine "SELECT 1 ... LIMIT 1" yapısını kullanarak veritabanı
// motorunun tüm veriyi okumasını engeller ve performansı artırır.
func (g *MySQLGrammar) CompileExists(b QueryBuilder) (string, []any, error) {
	if b.GetTable() == "" {
		return "", nil, ErrNoTable
	}

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

// CompileCount, satır sayısını öğrenmek için COUNT sorgusu oluşturur.
//
// Eğer belirli bir kolon verilirse NULL olmayan satırları sayar,
// verilmezse COUNT(*) kullanarak tüm satırları sayar.
func (g *MySQLGrammar) CompileCount(b QueryBuilder, column string) (string, []any, error) {
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

// CompileAggregate, SUM, AVG, MIN, MAX gibi toplama fonksiyonlarını işler.
func (g *MySQLGrammar) CompileAggregate(b QueryBuilder, fn, column string) (string, []any, error) {
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

// CompileTruncate, bir tabloyu tamamen boşaltmak için TRUNCATE komutunu oluşturur.
//
// DELETE FROM'dan farklı olarak, TRUNCATE DDL (Data Definition Language) komutudur
// ve genellikle geri alınamaz (transaction-safe değildir). Auto-increment sayacını sıfırlar.
func (g *MySQLGrammar) CompileTruncate(b QueryBuilder) (string, error) {
	if b.GetTable() == "" {
		return "", ErrNoTable
	}

	table, err := g.WrapTable(b.GetTable())
	if err != nil {
		return "", err
	}

	return "TRUNCATE TABLE " + table, nil
}

// CompileUpsert, MySQL'in "ON DUPLICATE KEY UPDATE" özelliğini kullanarak
// "varsa güncelle, yoksa ekle" (update or insert) mantığını uygular.
//
// Modern uygulama geliştirmede idempotent işlemler için kritik bir fonksiyondur.
func (g *MySQLGrammar) CompileUpsert(b QueryBuilder, data map[string]any, updateColumns []string) (string, []any, error) {
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

// ----------------------------------------------------------------------------
// Internal helpers
// ----------------------------------------------------------------------------

// compileWheres, çoklu WHERE koşullarını (AND/OR mantığıyla) birleştirir.
// Recursive (özyineli) yapısı sayesinde iç içe parantez gruplarını yönetebilir.
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

// compileWhere, tekil bir WHERE koşul tipini uygun SQL parçasına dönüştürür.
// Strateji deseni (Strategy Pattern) benzeri bir yapı ile farklı where tiplerini yönetir.
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

// compileWhereBasic, standart "col = val" veya "col > val" karşılaştırmalarını derler.
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

// compileWhereIn, "col IN (1, 2, 3)" yapısını oluşturur.
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

// compileWhereBetween, "col BETWEEN x AND y" yapısını oluşturur.
func (g *MySQLGrammar) compileWhereBetween(where WhereClause, not bool) (string, []any, error) {
	if len(where.Values) != 2 {
		return "", nil, ErrInvalidBetween
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

// compileWhereNull, "col IS NULL" kontrolünü oluşturur.
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

// compileWhereNested, iç içe geçmiş parantezli sorguları "(...)" içine alır.
// Örn: WHERE a=1 AND (b=2 OR c=3)
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

// compileWhereDate, tarih bazlı özel sorguları derler.
func (g *MySQLGrammar) compileWhereDate(where WhereClause, fn string) (string, []any, error) {
	column, err := g.Wrap(where.Column)
	if err != nil {
		return "", nil, err
	}

	return fn + "(" + column + ") = ?", []any{where.Value}, nil
}

// compileJoin, tablolar arası ilişki kuran JOIN ifadelerini derler.
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
