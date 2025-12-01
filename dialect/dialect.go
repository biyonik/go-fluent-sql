// Package dialect, farklı veritabanları için SQL dilbilgisi (grammar) implementasyonlarını sağlar.
// Bu paket, sorgu oluşturucular (QueryBuilder) ve gramerler (Grammar) aracılığıyla
// MySQL, PostgreSQL, SQLite gibi farklı veritabanları için sorguları derler.
//
// Yazar: Ahmet ALTUN
// Github: github.com/biyonik
// LinkedIn: linkedin.com/in/biyonik
// Email: ahmet.altun60@gmail.com
package dialect

// ----------------------------------------------------------------------------
// QueryBuilder Interface (import döngüsünü kırmak için)
// ----------------------------------------------------------------------------

// QueryBuilder, Grammar implementasyonlarının ihtiyaç duyduğu arayüzü tanımlar.
// Bu arayüz, ana paket ile dialect paketi arasındaki import döngüsünü kırmak için kullanılır.
type QueryBuilder interface {
	GetTable() string
	GetTableAlias() string
	GetColumns() []string
	IsDistinct() bool
	GetWheres() []WhereClause
	GetOrders() []OrderClause
	GetJoins() []JoinClause
	GetGroupBy() []string
	GetHaving() []WhereClause
	GetLimit() *int
	GetOffset() *int
}

// ----------------------------------------------------------------------------
// Grammar Interface
// ----------------------------------------------------------------------------

// Grammar, sorgu bileşenlerini veritabanına özgü SQL ifadelerine çevirir.
// Her veritabanı için (MySQL, PostgreSQL, SQLite) ayrı bir Grammar implementasyonu vardır.
type Grammar interface {
	// Name, gramerin kimliğini döndürür (örn. "mysql", "postgres", "sqlite").
	Name() string

	// Wrap, bir sütun veya tablo adını veritabanına özgü tırnaklarla sarar.
	// Geçersiz karakter içeriyorsa hata döner.
	Wrap(identifier string) (string, error)

	// WrapTable, tablo adını sarar ve alias yönetir.
	WrapTable(table string) (string, error)

	// WrapValue, JOINlerde kullanılacak sütun değerini sarar.
	WrapValue(value string) (string, error)

	// Placeholder, verilen indeks için parametre yer tutucusunu döndürür.
	// MySQL: "?", PostgreSQL: "$1", "$2" vb.
	Placeholder(index int) string

	// CompileSelect, SELECT sorgusunu derler.
	CompileSelect(b QueryBuilder) (string, []any, error)

	// CompileInsert, INSERT sorgusunu derler.
	CompileInsert(b QueryBuilder, data map[string]any) (string, []any, error)

	// CompileInsertBatch, toplu INSERT sorgusunu derler.
	CompileInsertBatch(b QueryBuilder, data []map[string]any) (string, []any, error)

	// CompileUpdate, UPDATE sorgusunu derler.
	CompileUpdate(b QueryBuilder, data map[string]any) (string, []any, error)

	// CompileDelete, DELETE sorgusunu derler.
	CompileDelete(b QueryBuilder) (string, []any, error)

	// CompileExists, EXISTS alt sorgusunu derler.
	CompileExists(b QueryBuilder) (string, []any, error)

	// CompileCount, COUNT sorgusunu derler.
	CompileCount(b QueryBuilder, column string) (string, []any, error)

	// CompileAggregate, SUM, AVG, MIN, MAX gibi agregat fonksiyonlarını derler.
	CompileAggregate(b QueryBuilder, fn, column string) (string, []any, error)

	// CompileTruncate, TRUNCATE TABLE ifadesini derler.
	CompileTruncate(b QueryBuilder) (string, error)

	// CompileUpsert, INSERT ... ON DUPLICATE KEY UPDATE sorgusunu derler.
	CompileUpsert(b QueryBuilder, data map[string]any, updateColumns []string) (string, []any, error)

	// SupportsReturning, gramerin RETURNING cümlesini destekleyip desteklemediğini döndürür.
	SupportsReturning() bool

	// DateFormat, veritabanı için tarih formatını döndürür.
	DateFormat() string
}

// ----------------------------------------------------------------------------
// Base Grammar (ortak fonksiyonlar)
// ----------------------------------------------------------------------------

// BaseGrammar, tüm gramer implementasyonları için ortak fonksiyonellik sağlar.
type BaseGrammar struct {
	name       string
	dateFormat string
}

// Name, gramerin adını döndürür.
func (g *BaseGrammar) Name() string {
	return g.name
}

// DateFormat, gramerin tarih formatını döndürür.
// Format belirtilmemişse varsayılan "2006-01-02 15:04:05" kullanılır.
func (g *BaseGrammar) DateFormat() string {
	if g.dateFormat == "" {
		return "2006-01-02 15:04:05"
	}
	return g.dateFormat
}

// SupportsReturning, default olarak false döner.
func (g *BaseGrammar) SupportsReturning() bool {
	return false
}

// ----------------------------------------------------------------------------
// WHERE Clause Types
// ----------------------------------------------------------------------------

// WhereType, WHERE koşulunun türünü belirtir.
type WhereType int

const (
	WhereTypeBasic WhereType = iota
	WhereTypeIn
	WhereTypeNotIn
	WhereTypeBetween
	WhereTypeNotBetween
	WhereTypeNull
	WhereTypeNotNull
	WhereTypeRaw
	WhereTypeNested
	WhereTypeDate
	WhereTypeYear
	WhereTypeMonth
	WhereTypeDay
)

// String, WhereType'ın string temsilini döndürür.
func (t WhereType) String() string {
	names := [...]string{
		"Basic", "In", "NotIn", "Between", "NotBetween",
		"Null", "NotNull", "Raw", "Nested",
		"Date", "Year", "Month", "Day",
	}
	if int(t) < len(names) {
		return names[t]
	}
	return "Unknown"
}

// WhereBoolean, AND veya OR bağlacını belirtir.
type WhereBoolean int

const (
	WhereBooleanAnd WhereBoolean = iota
	WhereBooleanOr
)

// String, SQL için boolean kelimesini döndürür.
func (b WhereBoolean) String() string {
	if b == WhereBooleanOr {
		return "OR"
	}
	return "AND"
}

// WhereClause, tek bir WHERE koşulunu temsil eder.
type WhereClause struct {
	Type     WhereType     // Koşul türü
	Boolean  WhereBoolean  // AND/OR bağlacı
	Column   string        // Sütun adı
	Operator string        // Operatör (örn. "=", "IN")
	Value    any           // Tek değer
	Values   []any         // IN, BETWEEN gibi çoklu değerler
	Nested   []WhereClause // İç içe koşullar
	Raw      string        // Raw SQL ifadesi (dikkatli kullanın)
	Bindings []any         // Raw SQL bağlamaları
}

// ----------------------------------------------------------------------------
// ORDER BY Types
// ----------------------------------------------------------------------------

// OrderDirection, sıralama yönünü belirtir.
type OrderDirection string

const (
	OrderAsc  OrderDirection = "ASC"
	OrderDesc OrderDirection = "DESC"
)

// IsValid, yönün geçerli olup olmadığını kontrol eder.
func (d OrderDirection) IsValid() bool {
	return d == OrderAsc || d == OrderDesc
}

// OrderClause, ORDER BY ifadesini temsil eder.
type OrderClause struct {
	Column    string
	Direction OrderDirection
	Raw       string // Raw ifade (dikkatli kullanın)
}

// ----------------------------------------------------------------------------
// JOIN Types
// ----------------------------------------------------------------------------

// JoinType, JOIN türünü belirtir.
type JoinType string

const (
	JoinInner JoinType = "INNER"
	JoinLeft  JoinType = "LEFT"
	JoinRight JoinType = "RIGHT"
	JoinCross JoinType = "CROSS"
)

// JoinClause, JOIN ifadesini temsil eder.
type JoinClause struct {
	Type     JoinType
	Table    string
	Alias    string // Opsiyonel tablo alias
	First    string // Sol sütun
	Operator string // Genellikle "="
	Second   string // Sağ sütun
}

// ----------------------------------------------------------------------------
// Sentinel Errors (dialect-specific)
// ----------------------------------------------------------------------------

// Dialect implementasyonları için ortak hatalar.
// Ana paket ile import döngüsünü önlemek için burada tanımlanmıştır.
var (
	ErrNoTable           = &DialectError{Message: "no table specified"}
	ErrNoColumns         = &DialectError{Message: "no columns specified"}
	ErrEmptyBatch        = &DialectError{Message: "cannot insert empty batch"}
	ErrInconsistentBatch = &DialectError{Message: "inconsistent columns in batch"}
	ErrEmptyWhereIn      = &DialectError{Message: "empty slice passed to WhereIn"}
	ErrInvalidBetween    = &DialectError{Message: "BETWEEN requires exactly 2 values"}
)

// DialectError, dialect'e özgü hataları temsil eder.
type DialectError struct {
	Message string
}

// Error, hatayı string olarak döndürür.
func (e *DialectError) Error() string {
	return "dialect: " + e.Message
}
