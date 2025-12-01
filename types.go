package fluentsql

import (
	"database/sql"
	"time"
)

/*
 * ----------------------------------------------------------------------------
 * FLUENTSQL TYPE DEFINITIONS
 * ----------------------------------------------------------------------------
 *
 * Bu dosya, FluentSQL paketinin veri taşıma ve yapılandırma katmanını oluşturur.
 * Bir veritabanı kütüphanesinin en kritik unsuru, ham SQL dünyası ile Go'nun
 * tip güvenli dünyası (Type-Safety) arasındaki köprüyü kurmaktır.
 *
 * Burada yapılanlar:
 * 1. Abstraction (Soyutlama): Ham `sql.Result` nesneleri sarmalanarak, geliştiricinin
 * nil pointer hatalarıyla veya sürücü uyumsuzluklarıyla uğraşması engellenir.
 * 2. Navigation (Navigasyon): Pagination yapısı ile büyük veri setleri, yönetilebilir
 * ve gezilebilir parçalara bölünür.
 * 3. Configuration (Yapılandırma): Veritabanı bağlantısının sadece "nereye" yapılacağı değil,
 * "nasıl" davranacağı (pooling, timeout, charset) da burada belirlenir.
 *
 * Neden Bu Şekilde?
 * Go'nun `database/sql` paketi güçlüdür ancak ham haliyle kullanıldığında çok fazla
 * "boilerplate" (tekrarlayan kod) gerektirir. Bu tipler, bu karmaşayı bir standart
 * altına alarak kütüphanenin geri kalanının temiz bir API sunmasını sağlar.
 *
 * @author Ahmet ALTUN
 * @github github.com/biyonik
 * @linkedin linkedin.com/in/biyonik
 * @email ahmet.altun60@gmail.com
 * ----------------------------------------------------------------------------
 */

// ----------------------------------------------------------------------------
// Query Result Types
// ----------------------------------------------------------------------------

// QueryResult, bir INSERT, UPDATE veya DELETE işlemi sonucunda veritabanından dönen
// ham yanıtı sarmalayan (wrapper) yapıdır.
//
// Bu yapı, ham `sql.Result` arabirimini doğrudan dışarı sızdırmak yerine,
// üzerinde güvenli erişim metotları sunarak olası çalışma zamanı hatalarını (runtime errors)
// minimize etmeyi hedefler.
type QueryResult struct {
	result sql.Result
}

// NewQueryResult, ham `sql.Result` nesnesinden güvenli bir FluentSQL sonuç nesnesi türetir.
//
// Bu fabrika metodu, veritabanı sürücüsünden dönen sonucu paketleyerek
// kütüphanenin standartlarına uygun hale getirir.
func NewQueryResult(result sql.Result) *QueryResult {
	return &QueryResult{result: result}
}

// LastInsertID, veritabanına son eklenen kaydın benzersiz kimliğini (ID) döndürür.
//
// Genellikle AUTO_INCREMENT (MySQL) veya SERIAL (Postgres) alanlar için kullanılır.
// Eğer altta yatan veritabanı sürücüsü bu özelliği desteklemiyorsa veya işlem başarısızsa
// 0 ve hata döner. Ayrıca sonucun `nil` olup olmadığını kontrol ederek panic durumunu önler.
func (r *QueryResult) LastInsertID() (int64, error) {
	if r.result == nil {
		return 0, ErrNoRows
	}
	return r.result.LastInsertId()
}

// RowsAffected, çalıştırılan sorgudan kaç adet satırın etkilendiğini bildirir.
//
// Özellikle toplu güncellemelerde (UPDATE) veya silme (DELETE) işlemlerinde,
// operasyonun başarısını ve kapsamını doğrulamak için kritik bir metriktir.
func (r *QueryResult) RowsAffected() (int64, error) {
	if r.result == nil {
		return 0, ErrNoRows
	}
	return r.result.RowsAffected()
}

// ----------------------------------------------------------------------------
// Pagination Types
// ----------------------------------------------------------------------------

// Pagination, veri listeleme işlemlerinde "sayfalama" mantığını yöneten veri yapısıdır.
//
// Modern web uygulamalarında ve API'lerde, binlerce kaydı tek seferde çekmek yerine
// parçalar halinde sunmak (chunking) performansın anahtarıdır. Bu yapı, hem istemciye
// sunulacak meta veriyi (toplam sayfa, mevcut sayfa vb.) hem de SQL sorgusu için
// gerekli olan LIMIT/OFFSET hesaplamalarını barındırır.
type Pagination struct {
	Page       int   // Mevcut sayfa numarası (1'den başlar)
	PerPage    int   // Sayfa başına gösterilecek kayıt sayısı
	Total      int64 // Veritabanındaki toplam kayıt sayısı
	TotalPages int   // Hesaplanan toplam sayfa sayısı
	HasMore    bool  // Sonraki sayfaların olup olmadığını belirten bayrak
}

// NewPagination, ham sayfalama parametrelerinden zengin bir Pagination nesnesi oluşturur.
//
// Bu kurucu metot (constructor), geçersiz veya eksik parametreleri (örn: negatif sayfa sayısı)
// otomatik olarak "akıllı varsayılanlara" (sensible defaults) dönüştürür.
// Ayrıca toplam sayfa sayısını ve veri setinin devamı olup olmadığını matematiksel olarak hesaplar.
func NewPagination(page, perPage int, total int64) *Pagination {
	if perPage <= 0 {
		perPage = 15 // Varsayılan olarak sayfa başına 15 kayıt
	}
	if page <= 0 {
		page = 1
	}

	// Toplam sayfa sayısının tavan değerini hesapla
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

// Offset, SQL sorgusu için gerekli olan başlangıç noktasını (SKIP miktarını) hesaplar.
//
// Örnek: 3. sayfadasınız ve her sayfada 10 kayıt var.
// Offset = (3 - 1) * 10 = 20. Yani ilk 20 kaydı atla, sonrakileri getir.
func (p *Pagination) Offset() int {
	return (p.Page - 1) * p.PerPage
}

// HasPrev, mevcut sayfadan geriye gidilip gidilemeyeceğini kontrol eder.
// Sayfa 1'de isek geriye gidiş yoktur.
func (p *Pagination) HasPrev() bool {
	return p.Page > 1
}

// HasNext, mevcut sayfadan ileriye gidilip gidilemeyeceğini kontrol eder.
// Eğer elimizdeki veri seti toplam sayfa sayısına ulaşmadıysa true döner.
func (p *Pagination) HasNext() bool {
	return p.HasMore
}

// ----------------------------------------------------------------------------
// Configuration Types
// ----------------------------------------------------------------------------

// Config, veritabanı bağlantısının DNA'sını oluşturan yapılandırma şemasıdır.
//
// Bir veritabanı bağlantısı sadece host ve porttan ibaret değildir.
// Performanslı bir uygulama için Connection Pooling (Havuzlama), Timeout süreleri,
// Karakter setleri (Charset) ve SSL/TLS ayarları hayati önem taşır.
// Bu struct, tüm bu parametreleri tek bir çatı altında toplar.
type Config struct {
	Driver       string        // Kullanılacak sürücü: "mysql", "postgres" vb.
	Host         string        // Veritabanı sunucusunun adresi (IP veya domain)
	Port         int           // Bağlantı portu
	Database     string        // Bağlanılacak veritabanı (schema) adı
	Username     string        // Yetkilendirme için kullanıcı adı
	Password     string        // Yetkilendirme için parola
	Charset      string        // Karakter seti (varsayılan: utf8mb4)
	Collation    string        // Sıralama ve karşılaştırma kuralları (varsayılan: utf8mb4_unicode_ci)
	Prefix       string        // Tablo isimlerinin önüne eklenecek önek (prefix)
	MaxOpenConns int           // Havuzdaki maksimum açık bağlantı sayısı (0 = sınırsız)
	MaxIdleConns int           // Havuzda boşta bekletilecek maksimum bağlantı sayısı
	ConnMaxLife  time.Duration // Bir bağlantının yaşam döngüsü süresi
	ConnMaxIdle  time.Duration // Bir bağlantının boşta kalabileceği maksimum süre
	TLS          bool          // TLS/SSL şifreli bağlantı zorunluluğu
}

// DefaultConfig, üretim ortamına (production) uygun varsayılan ayarlarla
// dolu bir konfigürasyon nesnesi döndürür.
//
// Geliştirici hiçbir ayar yapmasa bile, bu metot sayesinde "çalışan" ve
// belirli bir performans standardına sahip bir yapılandırma elde eder.
func DefaultConfig() *Config {
	return &Config{
		Driver:       "mysql",
		Host:         "localhost",
		Port:         3306,
		Charset:      "utf8mb4",
		Collation:    "utf8mb4_unicode_ci",
		MaxOpenConns: 25,              // Aşırı yüklenmeyi önlemek için makul bir sınır
		MaxIdleConns: 5,               // Ani trafik artışları için hazırda bekleyen bağlantılar
		ConnMaxLife:  5 * time.Minute, // Bayat bağlantıları (stale connections) temizle
		ConnMaxIdle:  5 * time.Minute,
	}
}

// DSN (Data Source Name), veritabanı sürücüsünün anlayacağı formatta bağlantı
// dizesini (connection string) oluşturur.
//
// Özellikle MySQL için gereken karmaşık parametre dizilimini (user:pass@tcp(host:port)/db?param=val)
// dinamik olarak inşa eder. Charset, collation ve parseTime gibi kritik parametreleri
// sorgu dizesine (query string) ekler.
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

	// Bağlantı parametrelerini (Query Params) ayarla
	params := "?"
	if c.Charset != "" {
		params += "charset=" + c.Charset + "&"
	}
	if c.Collation != "" {
		params += "collation=" + c.Collation + "&"
	}
	// Tarih/Saat alanlarının Go'nun time.Time tipine otomatik dönüşümü için gerekli
	params += "parseTime=true&"
	if c.TLS {
		params += "tls=true&"
	}

	// Son eklenen gereksiz "&" karakterini temizle
	if len(params) > 1 {
		dsn += params[:len(params)-1]
	}

	return dsn
}

// itoa, tamsayıyı (int) stringe çeviren hafif siklet bir yardımcı fonksiyondur.
//
// Neden strconv.Itoa değil?
// Kütüphanenin bağımlılıklarını minimumda tutmak ve bu basit işlem için
// büyük bir paketi import etmemek adına, dahili (internal) bir çözüm tercih edilmiştir.
// Recursive (özyineli) yapısıyla negatif sayıları da destekler.
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

// ----------------------------------------------------------------------------
// Logger Interface
// ----------------------------------------------------------------------------

// Logger, sistemin kara kutusudur.
//
// Çalışan SQL sorgularını, parametreleri, sorgunun ne kadar sürdüğünü ve
// olası hataları izlemek (observability) için kullanılan arayüzdür.
// Geliştirici kendi logger'ını enjekte ederek sorgu performansını analiz edebilir.
type Logger interface {
	Log(query string, args []any, duration time.Duration, err error)
}

// NopLogger (No-Operation Logger), "sessiz mod" için kullanılan bir logger uygulamasıdır.
//
// Eğer geliştirici herhangi bir loglama mekanizması belirtmezse, sistemin
// hata vermeden çalışmaya devam etmesi için bu boş (dummy) yapı kullanılır.
// Tüm logları yutar ve hiçbir işlem yapmaz.
type NopLogger struct{}

// Log, NopLogger'ın implementasyonudur. Gelen tüm veriyi yok sayar.
func (NopLogger) Log(string, []any, time.Duration, error) {}
