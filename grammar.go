package fluentsql

/*
=======================================================================================================================
 💠 GRAMMAR — SQL'in Diline Şekil Veren Zihin Katmanı 💠

 Bu dosya, FluentSQL'in en kritik yapı taşlarından birini temsil eder:
 **Grammar** — yani sorgunun nasıl ifade edilmesi gerektiğine karar veren, SQL’in dilbilgisi.

 Bir sorgu yazdığımızı düşün:
  "Select * from users where id = 5"
 Biz geliştiriciler için bu doğal bir cümledir.
 Fakat farklı veritabanlarının bu cümleyi işleme şekilleri birbirinden ayrılır:

   🔸 MySQL → `SELECT * FROM \`users\` WHERE \`id\` = ?`
   🔸 PostgreSQL → `SELECT * FROM "users" WHERE "id" = $1`

 Aynı anlam, fakat ifade biçimi farklı.
 İşte **Grammar**, bu dönüşümün beynidir.
 Sorgular Builder üzerinde akarken, Grammar onları yakalar, sarar, biçimlendirir
 ve hedeflenen veritabanı motorunun anlayacağı forma çevirir.

 Bu tasarım sayesinde:
   - Kodumuz motor bağımsız kalır.
   - "MySQL mi PostgreSQL mi?" sorusu uygulama katmanından uzaklaşır.
   - Aynı Builder, farklı Grammar'larla bambaşka SQL cümlelerine dönüşebilir.

 Bir sorgu yalnızca çalışmak için değil;
 doğru, güvenli ve deterministik üretilebilmek için Grammar’a ihtiyaç duyar.
 Özellikle kullanıcıdan gelen identifier’ların doğrulanması ve sanitize edilmesi,
 SQL injection tehditlerini engelleyen en önemli savunma hattıdır.

 Aşağıdaki interface ve BaseGrammar yapısı,
 yeni Grammar implementasyonları için iskelet niteliğindedir.
 Bir çatı, bir sözleşme ve aynı zamanda bir rehber.

 @author    Ahmet ALTUN
 @github    github.com/biyonik
 @linkedin  linkedin.com/in/biyonik
 @email     ahmet.altun60@gmail.com
=======================================================================================================================
*/

// Grammar arayüzü SQL cümlelerinin *dil kurallarını* belirler.
// Her veritabanı MySQL, PostgreSQL veya SQLite için ayrı bir Grammar yazılabilir
// ve query builder bu sayede motor bağımsız şekilde çalışabilir.
//
// Bu interface’in sorumlulukları:
//
//	✔ Identifier sarmalama (`users` → `\`users\`` veya `"users"`)
//	✔ Parametre placeholder üretimi (`?`, `$1`, `$2` ...)
//	✔ SELECT / INSERT / UPDATE / DELETE gibi sorguları derleme
//	✔ Güvenli SQL üretmek için kullanıcı girdilerini doğrulama
//
// Security Notu:
// Kullanıcıdan gelen identifier’lar mutlaka doğrulanmalıdır,
// aksi hâlde SQL injection’a açık bir kanal oluşabilir.
// Bu yüzden Wrap, WrapTable, WrapValue hata üretebilir.
type Grammar interface {

	// Name → Grammar'ın kimliğini döndürür. (mysql | postgres | sqlite ...)
	// Amaç: Builder veya log sistemleri hangi dil kurallarıyla çalıştığını bilir.
	Name() string

	// Wrap → Sütun veya tablo isimlerini veritabanına uygun quote formatıyla sarar.
	// Güvenlik kontrolü içerir, hatalı identifier yakalanır ve hata döner.
	//
	// Örnek:
	//   MySQL:      "users"     → "`users`"
	//   MySQL:      "u.name"    → "`u`.`name`"
	//   PostgreSQL: "users"     → `"users"`
	Wrap(identifier string) (string, error)

	// WrapTable → Tablo adlarını sarmalar, gerekirse alias üretir.
	// "users", "users as u", "users u" formatlarını destekler.
	WrapTable(table string) (string, error)

	// WrapValue → JOIN yapılırken kullanılan kolon referanslarını dönüştürür.
	//
	// Örnek:
	//   MySQL: "users.id" → "`users`.`id`"
	WrapValue(value string) (string, error)

	// Placeholder → Query parametrelerinin yerine geçecek placeholder’ı döndürür.
	// MySQL: "?" — PostgreSQL: "$1", "$2", ...
	Placeholder(index int) string

	// CompileSelect → SELECT sorgusu derleyici.
	// Tüm query bileşenlerini nihai SQL stringine dönüştürür.
	CompileSelect(b *Builder) (string, []any, error)

	// CompileInsert → Tekli INSERT sorgusu üretir.
	// Kolon isimleri sıralanarak deterministik çıktı sağlanır.
	CompileInsert(b *Builder, data map[string]any) (string, []any, error)

	// CompileInsertBatch → Çok satırlı INSERT işlemlerini üretir.
	// Tüm satırların aynı kolonlara sahip olması beklenir.
	CompileInsertBatch(b *Builder, data []map[string]any) (string, []any, error)

	// CompileUpdate → UPDATE sorgusu derler.
	// WHERE dahil edilir, aksi hâlde tüm tablo güncellenebilir—bu nedenle dikkat gerekir.
	CompileUpdate(b *Builder, data map[string]any) (string, []any, error)

	// CompileDelete → DELETE sorgusu üretir.
	// WHERE yoksa tüm tablo silinebilir. Çok tehlikeli! — bilinçli kullanılmalıdır.
	CompileDelete(b *Builder) (string, []any, error)

	// CompileExists → EXISTS alt-sorgusunu oluşturur.
	CompileExists(b *Builder) (string, []any, error)

	// CompileCount → COUNT(*) veya COUNT(column) sorgusu.
	CompileCount(b *Builder, column string) (string, []any, error)

	// CompileAggregate → SUM, AVG, MIN, MAX gibi aggregate fonksiyonlarını üretir.
	CompileAggregate(b *Builder, fn, column string) (string, []any, error)

	// CompileTruncate → TRUNCATE TABLE sorgusu.
	CompileTruncate(b *Builder) (string, error)

	// CompileUpsert → On Duplicate/On Conflict sorgularını üretir.
	// MySQL ve PostgreSQL’de farklıdır.
	CompileUpsert(b *Builder, data map[string]any, updateColumns []string) (string, []any, error)

	// SupportsReturning → RETURNING desteği var mı? PostgreSQL: evet, MySQL: hayır.
	SupportsReturning() bool

	// DateFormat → Veritabanının tarih formatı.
	// Varsayılan: "2006-01-02 15:04:05"
	DateFormat() string
}

// BaseGrammar → Farklı Grammar implementasyonlarının temel gövdesidir.
// Ortak davranışlar burada bulunur, alt sınıflar override edebilir.
// Bir iskelet, ama içinde kan dolaşan bir yapı değil — ruhu implementasyon verir.
type BaseGrammar struct {
	name        string // Grammar adı
	placeholder string // Placeholder formatı
	dateFormat  string // Tarih formatı
}

// Name → Grammar adını döndürür (*mysql*, *postgres*, *sqlite* ...).
// Loglama ve debug aşamasında tanımlayıcıdır.
func (g *BaseGrammar) Name() string {
	return g.name
}

// DateFormat → Eğer özel tanım yoksa standart format döner.
// Yazılımsal deterministik tarih formatı için önemlidir.
func (g *BaseGrammar) DateFormat() string {
	if g.dateFormat == "" {
		return "2006-01-02 15:04:05"
	}
	return g.dateFormat
}

// SupportsReturning → Varsayılan davranış *false*.
// Eğer Grammar RETURNING destekliyorsa override edilir.
func (g *BaseGrammar) SupportsReturning() bool {
	return false
}
