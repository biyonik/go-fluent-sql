package fluentsql

// -----------------------------------------------------------------------------
//  Bu dosya; FluentSQL yapısının çekirdek konfigürasyon katmanını oluşturan,
//  esnek ve genişletilebilir *Option* mimarisini içerir. Amaç, veritabanı
//  bağlantısı veya sorgu oluşturucu (Query Builder) üzerinde çalışırken,
//  geliştiricinin davranışı tek noktadan, okunabilir ve akıcı bir şekilde
//  yönetebilmesini sağlamaktır.
//
//  Burada kullanılan tasarım yaklaşımı, modern PHP frameworklerinde sıkça
//  rastladığımız *Fluent Config* modelinin Go dünyasına zarif bir yansımasıdır.
//  Her bir With* fonksiyonu; DB veya Builder üzerinde konfigürasyon
//  yapmayı sağlayan, dışarıdan enjekte edilen küçük fakat etkili dokunuşlardır.
//
//  Bu tasarımın özü şudur:
//  ✔ Bir ayarı değiştirmek istediğinde ek yapılandırma dosyalarına girmezsin,
//    sadece ilgili WithX fonksiyonunu DB kurulumuna eklersin.
//  ✔ Kodun okunabilirliği yükselir, bağımlılıklar sadeleşir.
//  ✔ Geliştirici zihinsel yük yaşamadan sistemi yönetir.
//
//  -- @author   Ahmet ALTUN
//  -- @github   github.com/biyonik
//  -- @linkedin linkedin.com/in/biyonik
//  -- @email    ahmet.altun60@gmail.com
// -----------------------------------------------------------------------------

// Option tipi, bir *DB* örneği üzerinde çalışan yapılandırma fonksiyonlarının
// temel imzasıdır. Bu tasarım sayesinde, yapılandırma parametreleri sabit ve
// katı değil; ihtiyaç oldukça genişletilebilir bir yapıya kavuşur.
//
// Neden böyle?
// Çünkü proje büyüdükçe veritabanı davranışı da evrilir. Option pattern,
// kodu işlemeye dokunmadan yeni ayarlar eklemeye olanak tanır.
type Option func(*DB)

// WithGrammar fonksiyonu, FluentSQL'in derleme aşamasında kullanacağı SQL
// gramerini değiştirmeye yarar. Varsayılan olarak MySQLGrammar kullanılır,
// fakat PostgreSQL, SQLite veya özel türevler kolayca eklenebilir.
//
// Neyi sağlar?
// • Sorgu oluşturma kurallarının değişmesini
// • Farklı veri tabanı motorlarına geçişte minimum maliyet
//
// Nasıl çalışır?
// • Option fonksiyonu olarak DB'ye enjekte edilir ve grammar alanını doldurur.
//
// Örnek:
//
//	db := fluentsql.NewDB(sqlDB, fluentsql.WithGrammar(fluentsql.NewPostgreSQLGrammar()))
func WithGrammar(g Grammar) Option {
	return func(d *DB) {
		d.grammar = g
	}
}

// WithScanner fonksiyonu, veritabanından dönen sonucu struct içerisine
// map eden tarayıcı yapısını değiştirmeye imkân tanır. Varsayılan tarayıcı
// DefaultScanner’dır ancak özel modellemede esneklik sunması için yeniden
// tanımlanabilir.
//
// Neden önemli?
// • Farklı veri tipleri veya dönüş yapılarıyla çalışırken kontrol sağlar
// • Geliştiricinin kendi scan algoritmasını yazmasına olanak tanır
//
// Örnek:
//
//	db := fluentsql.NewDB(sqlDB, fluentsql.WithScanner(customScanner))
func WithScanner(s Scanner) Option {
	return func(d *DB) {
		d.scanner = s
	}
}

// WithDebug fonksiyonu debug modunu aktif veya pasif hâle getirir.
// Debug açık olduğunda, oluşturulan her sorgu loglanır.
//
// Neyi, nasıl değiştirir?
// • Debug = true -> tüm query'ler Logger üzerinden görünür hale gelir
// • Geliştirme aşamasında şeffaflık sağlar, hata analizini hızlandırır
//
// Örnek:
//
//	db := fluentsql.NewDB(sqlDB, fluentsql.WithDebug(true))
func WithDebug(enabled bool) Option {
	return func(d *DB) {
		d.debug = enabled
	}
}

// WithLogger fonksiyonu özel bir logger tanımlamaya yarar.
// Debug moduyla birlikte kullanıldığında sorgularınızı istediğiniz formatta,
// istediğiniz hedefe aktarabilirsiniz.
//
// Kullanım amacı?
// • CLI, file, remote log server gibi farklı log ortamlarını desteklemek
// • Geliştiriciye loglama mimarisini özgürce şekillendirme gücü vermek
//
// Örnek:
//
//	db := fluentsql.NewDB(sqlDB,
//	    fluentsql.WithDebug(true),
//	    fluentsql.WithLogger(myLogger),
//	)
func WithLogger(logger Logger) Option {
	return func(d *DB) {
		d.logger = logger
	}
}

// WithTablePrefix fonksiyonu tüm tablo adlarına otomatik olarak prefix ekler.
// Çok tenantlı sistemlerde her müşterinin verisini ayırmak ya da proje genelinde
// ad çakışmalarını önlemek için oldukça zarif bir yaklaşımdır.
//
// Örnek:
//
//	db := fluentsql.NewDB(sqlDB, fluentsql.WithTablePrefix("app_"))
//	// db.Table("users")  →  "app_users"
func WithTablePrefix(prefix string) Option {
	return func(d *DB) {
		d.prefix = prefix
	}
}

// BuilderOption tipi, yalnızca query bazlı kullanılan yapılandırmalardır.
// DB Option'larından farklıdır çünkü her sorguda ayrı davranışlara izin verir.
//
// Amaç:
// • Daha ince taneli kontrol
// • Bağlam bazlı varyasyon
type BuilderOption func(*Builder)

// -----------------------------------------------------------------------------
// Aşağıdaki yorumlu fonksiyon, Builder’a varsayılan context ekleme amacıyla
// tasarlanmıştır. Bu sayede *Context yöntemleri kullanılmasa dahi builder
// belirli bir çalışma bağlamı ile ilişkilendirilebilir.
//
// Şu an devre dışıdır ancak sistem büyüdüğünde tekrar açılabilir.
// -----------------------------------------------------------------------------

// func WithContext(ctx context.Context) BuilderOption {
// 	return func(b *Builder) {
// 		b.defaultCtx = ctx
// 	}
// }

// applyOptions fonksiyonu, DB oluşturulurken verilen bütün Option'ları
// sırayla işler. Buradaki tasarım prensibi; konfigürasyonu birbirine
// zincirleme bağlamak ve esnek büyüyebilen bir altyapı sunmaktır.
//
// Çalışma prensibi:
// 1) DB oluşturma sırasında tüm Option fonksiyonları döngü ile taranır
// 2) Nil olmayan her Option DB üzerine uygulanır
// 3) Sonuç: Temiz, okunabilir, injectable bir kurulum zinciri
func applyOptions(d *DB, opts []Option) {
	for _, opt := range opts {
		if opt != nil {
			opt(d)
		}
	}
}
