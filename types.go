package fluentsql

//Bu dosya; SQL sorgularının parçalarını nesne mantığıyla yönetilebilir hale getirmek için tasarlanmış bir çekirdek katmandır.
//WHERE, JOIN, ORDER, PAGINATION, CONFIGURATION gibi SQL’in en temel bloklarını tek bir merkezde toplar, böylece kod yazarken karmaşık dize manipülasyonlarına ihtiyaç kalmaz.
//
//Amaç;
//✔ SQL sorgularını okunabilir kılmak
//✔ Dinamik oluşturulabilen filtre yapısı sunmak
//✔ ORM ve Query Builder mantığını Go ekosistemine taşımak
//✔ Sade, genişletilebilir ve esnek bir altyapı kurmak
//
//Buradaki her tip, Go’nun düşük seviyeli SQL kontrol esnekliği ile Laravel/Symfony tarzı akışkan kullanım hissini aynı potada eritmek üzere düşünülmüştür.
//
//@author Ahmet ALTUN
//@github github.com/biyonik
//@linkedin linkedin.com/in/biyonik
//@email ahmet.altun60@gmail.com

import (
	"database/sql"
	"time"
)

// -----------------------------------------------------------------------------
// WHERE Clause Types
// -----------------------------------------------------------------------------

// WhereType, oluşturulan WHERE şartının hangi yapıda olduğunu ifade eder.
// Temel karşılaştırmalar, IN, BETWEEN, NULL kontrolleri ve ham RAW sorgularını
// birbirinden ayırarak builder’ın esnek şekilde işlenebilmesini sağlar.
// Bu enum yapısı query compiler tarafından yorumlanır ve uygun SQL formatı üretilir.
type WhereType int

const (
	// WhereTypeBasic → column op value formatındaki temel şart objesi.
	WhereTypeBasic WhereType = iota
	// WhereTypeIn → WHERE IN(column, ...) sorgusu için kullanılır.
	WhereTypeIn
	// WhereTypeNotIn → IN’in tam tersi şekilde dışlayan bir filtre oluşturur.
	WhereTypeNotIn
	// WhereTypeBetween → WHERE BETWEEN x AND y şeklindeki aralıklı karşılaştırmalardır.
	WhereTypeBetween
	// WhereTypeNotBetween → BETWEEN şartının ters çevrilmiş halidir.
	WhereTypeNotBetween
	// WhereTypeNull → WHERE column IS NULL koşulu.
	WhereTypeNull
	// WhereTypeNotNull → WHERE column IS NOT NULL koşulu.
	WhereTypeNotNull
	// WhereTypeRaw → ham SQL cümlesi kabul eder, güvenlik hassasiyeti gerektirir.
	WhereTypeRaw
	// WhereTypeNested → Parantezli ( ) gruplanmış alt şart kümeleri taşır.
	WhereTypeNested
	// WhereTypeDate → MySQL DATE(column) = value şeklinde tarih bazlı filtre.
	WhereTypeDate
	// WhereTypeYear → WHERE YEAR(column) = ? için kullanılır.
	WhereTypeYear
	// WhereTypeMonth → WHERE MONTH(column) = ?
	WhereTypeMonth
	// WhereTypeDay → WHERE DAY(column) = ?
	WhereTypeDay
)

// String → enum değerini insan okunabilir SQL ifadelerine dönüştürür.
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

// -----------------------------------------------------------------------------
// WHERE Boolean
// -----------------------------------------------------------------------------

// WhereBoolean → WHERE bloklarının AND / OR olarak zincirlenmesini kontrol eder.
type WhereBoolean int

const (
	// WhereBooleanAnd → Varsayılan bağlayıcıdır.
	WhereBooleanAnd WhereBoolean = iota
	// WhereBooleanOr → Alternatif koşul kullanımına izin verir.
	WhereBooleanOr
)

// String → SQL’de kullanılan gerçek boolean keyword'ünü döner.
func (b WhereBoolean) String() string {
	if b == WhereBooleanOr {
		return "OR"
	}
	return "AND"
}

// -----------------------------------------------------------------------------
// WHERE CLAUSE OBJECT
// -----------------------------------------------------------------------------

// WhereClause → tek bir WHERE koşul satırını temsil eder.
// Kompakt ve genişletilebilir olacak şekilde tasarlanmıştır.
// IN, BETWEEN, RAW gibi farklı yapılar Values/Nested/Raw alanlarıyla yönetilir.
// Builder sistemi bu objeyi okuyarak otomatik SQL üretir.
type WhereClause struct {
	Type     WhereType
	Boolean  WhereBoolean
	Column   string
	Operator string
	Value    any
	Values   []any         // Çoklu IN/BETWEEN operasyonlarında kullanılır
	Nested   []WhereClause // Parantezli alt grup sorguları içerir
	Raw      string        // Tam SQL girilmek istenen riskli durumlar için
	Bindings []any         // RAW yazıldığında parametreleri taşır
}

// -----------------------------------------------------------------------------
// ORDER BY Types
// -----------------------------------------------------------------------------

// OrderDirection → ASC / DESC yön bilgisini barındırır.
type OrderDirection string

const (
	// OrderAsc → Küçükten büyüğe sıralama
	OrderAsc OrderDirection = "ASC"
	// OrderDesc → Büyükten küçüğe sıralama
	OrderDesc OrderDirection = "DESC"
)

// IsValid → yön bilgisinin geçerliliğini doğrular.
func (d OrderDirection) IsValid() bool {
	return d == OrderAsc || d == OrderDesc
}

// OrderClause → ORDER BY sütunu ve yönünü temsil eder.
// Raw alanı ile fonksiyonlu sıralamalara (LENGTH(name) gibi) izin verecek yapıdadır.
type OrderClause struct {
	Column    string
	Direction OrderDirection
	Raw       string // Ham sıralama cümlesi (yüksek dikkat gerektirir)
}

// -----------------------------------------------------------------------------
// JOIN TYPES
// -----------------------------------------------------------------------------

// JoinType → JOIN’in LEFT/RIGHT/CROSS/INNER türünü belirler.
type JoinType string

const (
	// JoinInner → INNER JOIN kullanım tipidir
	JoinInner JoinType = "INNER"
	// JoinLeft → Sol birleşim, soldaki kayıtları korur
	JoinLeft JoinType = "LEFT"
	// JoinRight → Sağ birleşim
	JoinRight JoinType = "RIGHT"
	// JoinCross → Kartezyen birleşim
	JoinCross JoinType = "CROSS"
)

// JoinClause → bir JOIN satırının tüm yapısal bilgisini tutar.
// First/Second alanları iki tabloyu birbirine bağlayan sütun eşleşmeleridir.
type JoinClause struct {
	Type     JoinType
	Table    string
	Alias    string // Tablo kısaltması
	First    string // Sol sütun
	Operator string // = , <>, > gibi karşılaştırma operatörleri
	Second   string // Sağ sütun
}

// -----------------------------------------------------------------------------
// QUERY RESULT
// -----------------------------------------------------------------------------

// QueryResult → Exec(), Insert(), Update(), Delete() sonucunu sarmalar.
// Böylece LastInsertID() ve RowsAffected() fonksiyonları daha anlamlı çalışır.
type QueryResult struct {
	result sql.Result
}

// NewQueryResult → sql.Result'ı QueryResult yapısına dönüştürür.
func NewQueryResult(result sql.Result) *QueryResult {
	return &QueryResult{result: result}
}

// LastInsertID → Veritabanının desteklemesi halinde son ID döner.
func (r *QueryResult) LastInsertID() (int64, error) {
	if r.result == nil {
		return 0, ErrNoRows
	}
	return r.result.LastInsertId()
}

// RowsAffected → Kaç satırın etkilendiğini döner.
func (r *QueryResult) RowsAffected() (int64, error) {
	if r.result == nil {
		return 0, ErrNoRows
	}
	return r.result.RowsAffected()
}

// -----------------------------------------------------------------------------
// PAGINATION STRUCT
// -----------------------------------------------------------------------------

// Pagination → sayfalama kontrol yapısıdır. API çıktılarında çok kullanışlıdır.
// Toplam kayıt, sayfa adedi, önce/sonra var mı gibi bilgileri barındırır.
type Pagination struct {
	Page       int   // Şu anki sayfa (1’den başlar)
	PerPage    int   // Sayfa başına veri adedi
	Total      int64 // Toplam kayıt
	TotalPages int   // Toplam sayfa sayısı
	HasMore    bool  // Sonraki sayfa var mı?
}

// NewPagination → verilen total değeriyle sayfalama yapısı oluşturur.
func NewPagination(page, perPage int, total int64) *Pagination {
	if perPage <= 0 {
		perPage = 15
	}
	if page <= 0 {
		page = 1
	}

	totalPages := int(total) / perPage
	if int(total)%perPage > 0 {
		totalPages++
	}

	return &Pagination{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
		HasMore:    page < totalPages,
	}
}

// Offset → LIMIT offset değerini üretir.
func (p *Pagination) Offset() int {
	return (p.Page - 1) * p.PerPage
}

// HasPrev → Önceki sayfa var mı?
func (p *Pagination) HasPrev() bool {
	return p.Page > 1
}

// HasNext → Sonraki sayfa var mı?
func (p *Pagination) HasNext() bool {
	return p.HasMore
}

// -----------------------------------------------------------------------------
// CONFIGURATION
// -----------------------------------------------------------------------------

// Config → veritabanı bağlantı ayarlarını taşır.
// Amacı; DSN üretmek, connection pool limitlerini belirlemek ve güvenli bağlanmaktır.
type Config struct {
	Driver       string
	Host         string
	Port         int
	Database     string
	Username     string
	Password     string
	Charset      string
	Collation    string
	Prefix       string
	MaxOpenConns int
	MaxIdleConns int
	ConnMaxLife  time.Duration
	ConnMaxIdle  time.Duration
	TLS          bool
}

// DefaultConfig → MySQL tabanlı bağlantı için önerilen başlangıç değerlerini döner.
func DefaultConfig() *Config {
	return &Config{
		Driver:       "mysql",
		Host:         "localhost",
		Port:         3306,
		Charset:      "utf8mb4",
		Collation:    "utf8mb4_unicode_ci",
		MaxOpenConns: 25,
		MaxIdleConns: 5,
		ConnMaxLife:  5 * time.Minute,
		ConnMaxIdle:  5 * time.Minute,
	}
}

// DSN → MySQL bağlanma stringi oluşturur.
// user:pass@tcp(host:port)/db?params formatına göre çalışır.
func (c *Config) DSN() string {
	dsn := ""
	if c.Username != "" {
		dsn += c.Username
		if c.Password != "" {
			dsn += ":" + c.Password
		}
		dsn += "@"
	}

	dsn += "tcp(" + c.Host
	if c.Port > 0 {
		dsn += ":" + itoa(c.Port)
	}
	dsn += ")/" + c.Database

	params := "?"
	if c.Charset != "" {
		params += "charset=" + c.Charset + "&"
	}
	if c.Collation != "" {
		params += "collation=" + c.Collation + "&"
	}
	params += "parseTime=true&"
	if c.TLS {
		params += "tls=true&"
	}

	if len(params) > 1 {
		dsn += params[:len(params)-1]
	}

	return dsn
}

// itoa → strconv kullanmadan int → string dönüşümü.
// Mikro maliyetli bir fonksiyondur. Harici import’u azaltır.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// -----------------------------------------------------------------------------
// LOGGING
// -----------------------------------------------------------------------------

// Logger → Sorgu çalıştıktan sonra tetiklenen standart logging arabirimidir.
// Query, parametreler, süre ve hata bilgisi gönderilir.
type Logger interface {
	Log(query string, args []any, duration time.Duration, err error)
}

// NopLogger → Loglama istemeyen sistemler için boş implementasyon.
// Üretim ortamında log kapatma seçeneği oluşturur.
type NopLogger struct{}

// Log → hiçbir işlem yapmaz.
func (NopLogger) Log(string, []any, time.Duration, error) {}

// StdLogger → Temel stdout loglayıcı.
// Not: Üretimde gelişmiş logging altyapısı ile geliştirilmelidir.
type StdLogger struct{}

// Log → Query çalışmasının sonucunu standart çıktıya yazar.
func (StdLogger) Log(query string, args []any, duration time.Duration, err error) {
	status := "OK"
	if err != nil {
		status = "ERR: " + err.Error()
	}
	_ = query
	_ = args
	_ = duration
	_ = status
}
