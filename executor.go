package fluentsql

import (
	"context"
	"database/sql"

	"github.com/biyonik/go-fluent-sql/dialect"
)

/*
=======================================================================================================================
  💠 FLUENT SQL – Veritabanıyla Konuşan Akıcı Bir Dil 💠
  Bu dosya; Go dilindeki sade, yalın fakat son derece güçlü standart `database/sql` yapısının üzerine,
  tıpkı Laravel veya Symfony'nin dokunsal ORM hissi gibi, daha akışkan ve insan-diline yakın bir sorgu üretim
  deneyimi katmak amacıyla oluşturulmuştur.

  Bu yapı sayesinde:
  - Bir sorgu yazarken `builder.Table("users").Where(...).Get()` benzeri doğal bir ifade gücü kazanırız.
  - İster normal bağlantı (`*sql.DB`), ister transaction (`*sql.Tx`) üzerinde çalışalım,
    aynı interface’i kullanarak iş mantığımızı değiştirmeden kod akışına devam ederiz.
  - DB katmanı yalnızca veri okuyan değil, geliştiriciyle konuşan, hata raporlayan ve transaction yönetimini
    üstlenen bir akıllı iş ortağına dönüşür.

  Bu tasarım yapılırken hedef şuydu:
  🔹 "Neyi yapıyorum?" — SQL kuruyorum.
  🔹 "Nasıl yapıyorum?" — Zincirli (Fluent) builder ile.
  🔹 "Neden böyle yapıyorum?" — Hem *raw power* hem *developer ergonomisi* aynı anda elimde olsun diye.

  Bu nedenle aşağıdaki kod, veritabanıyla kurulan ilişkiyi yalnızca teknik değil, aynı zamanda duygusal,
  yani geliştirici deneyimini önemseyen bir yaklaşımla ele alır.

  @author    Ahmet ALTUN
  @github    github.com/biyonik
  @linkedin  linkedin.com/in/biyonik
  @email     ahmet.altun60@gmail.com
=======================================================================================================================
*/

// QueryExecutor arayüzü; hem *sql.DB hem *sql.Tx yapılarının ortak olarak sağlayabildiği temel veritabanı
// fonksiyonlarını soyutlar. Böylece işlem ister direkt DB'de olsun ister Transaction içinde,
// kod yapısı ve çağrım şekli değişmeden akıcı biçimde çalışabilir.
//
// Bu mimari seçim, "Tek kod → iki farklı çalışma ortamı" yaklaşımının bir sonucudur.
// Özellikle transaction tabanlı finansal hareketlerde büyük esneklik sağlar.
// ---------------------------------------------------------------------
type QueryExecutor interface {

	// ExecContext -> INSERT/UPDATE/DELETE gibi sonuç satırı döndürmeyen komutlar için çalıştırma yöntemidir.
	// Parametre olarak context alır; timeout, cancel vb. durumlarda akış kontrolü sağlanabilir.
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)

	// QueryContext -> Birden fazla satır döndürebilen SELECT sorguları için çağrılır.
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)

	// QueryRowContext -> Tek satır beklenen SELECT işlemlerinde kullanılır.
	// Eğer veri yoksa *sql.Row.Err() ile boş dönebilir, bu durum bilinçli yönetilmelidir.
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Compile-time kontrolü: *sql.DB ve *sql.Tx gerçekten QueryExecutor'ı implement ediyor mu?
// Bu satırlar çalışma zamanında değil, derleme zamanında garanti sağlar.
// ---------------------------------------------------------------------
var (
	_ QueryExecutor = (*sql.DB)(nil)
	_ QueryExecutor = (*sql.Tx)(nil)
)

// DB struct'ı veritabanı bağlantısını sarar ve üzerine grammar, scanner, logging, prefix gibi
// ORM davranışlarını belirleyen özellikler ekler. Böylece DB artık yalnızca bağlanılan yer değil,
// sorguyu şekillendiren ve işleyen ana merkez olur.
//
// Bu yapı; "Salt bağlantı" → "Akıllı ORM çekirdeği" dönüşümünün temel taşıdır.
// ---------------------------------------------------------------------
type DB struct {
	*sql.DB                 // Standart Go DB nesnesi gömülü olarak bulunur.
	grammar dialect.Grammar // SQL cümle yapısını oluşturur (MySQL / PostgreSQL / SQLite vb.)
	scanner Scanner         // DB satırlarını struct'lara tarayıp dönüştüren bileşen.
	logger  Logger          // İsteğe bağlı kayıtlama sistemi, debug durumunda detay sağlar.
	debug   bool            // Sorgular loglansın mı? Geliştirici modu açık mı?
	prefix  string          // Tablo adlarının önüne otomatik eklenebilen global prefix.
}

// NewDB -> DB sarmalayıcısının oluşturulduğu yerdir.
// Varsayılan Grammar ve Scanner atanır, opsiyonlar ile davranış şekillendirilebilir.
// Geliştirici, yalnızca sql.DB verip gerisini bu wrapper'a teslim eder → Akıcı yapı başlar.
// ---------------------------------------------------------------------
func NewDB(db *sql.DB, opts ...Option) *DB {
	d := &DB{
		DB:      db,
		grammar: nil,
		scanner: nil,
		logger:  NopLogger{},
		debug:   false,
		prefix:  "",
	}

	// Kullanıcı tarafından verilen opsiyonlar DB yapılandırmasını değiştirir.
	for _, opt := range opts {
		opt(d)
	}

	// Defaults — Eğer kullanıcı grammar/scanner belirtmediyse MySQL grammar ve default scanner atanır.
	if d.grammar == nil {
		d.grammar = dialect.NewMySQLGrammar()
	}
	if d.scanner == nil {
		d.scanner = NewDefaultScanner()
	}

	return d
}

// Grammar -> Aktif SQL cümle oluşturma motorunu döndürür.
func (d *DB) Grammar() dialect.Grammar {
	return d.grammar
}

// Scanner -> Satır–>struct tarama mekanizmasını verir.
func (d *DB) Scanner() Scanner {
	return d.scanner
}

// Logger -> Sorgu izleme/raporlama sistemine dışarıdan erişim sağlar.
func (d *DB) Logger() Logger {
	return d.logger
}

// TablePrefix -> Tüm tabloların başında kullanılacak ön-ek (prefix) değerini döndürür.
func (d *DB) TablePrefix() string {
	return d.prefix
}

// IsDebug -> Debug modunda mıyız? Sorgular loglanacak mı? Bilgi verir.
func (d *DB) IsDebug() bool {
	return d.debug
}

// Table -> Yeni bir Query Builder oluşturur ve belirtilen tablo üzerinde çalışmaya başlar.
// Bu fonksiyon, sorgu yazımının ilk adımıdır. Zincirin başlangıç halkasıdır.
// ---------------------------------------------------------------------
func (d *DB) Table(name string) *Builder {
	return NewBuilder(d.DB, d.grammar, d.scanner).Table(name)
}

// BeginTx -> Manuel transaction başlatır. Bağlantıya güvenip işi tek adımda yapmak yerine,
// adım adım ilerlemek isteyen geliştiriciler için kontrollü güç sunar.
// ---------------------------------------------------------------------
func (d *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Transaction, error) {
	tx, err := d.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, WrapError("begin transaction", err)
	}
	return &Transaction{
		tx:      tx,
		grammar: d.grammar,
		scanner: d.scanner,
		logger:  d.logger,
		debug:   d.debug,
		prefix:  d.prefix,
		closed:  false,
	}, nil
}

// Begin -> Varsayılan ayarlarla transaction başlatır. Hızlı kullanım için kısayoldur.
// ---------------------------------------------------------------------
func (d *DB) Begin() (*Transaction, error) {
	return d.BeginTx(context.Background(), nil)
}

// Transaction -> Verilen fonksiyon içerisinde otomatik transaction yönetimi sağlar.
// Başarılı olursa commit, hata veya panic durumunda rollback yapar.
// Laravel `DB::transaction()` davranışına doğrudan bir karşılıktır.
// ---------------------------------------------------------------------
func (d *DB) Transaction(ctx context.Context, fn func(*Transaction) error) error {
	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// Panic güvenliği → Transaction içi kod hata fırlatırsa rollback yapılır.
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	// Kullanıcı fonksiyonunu çalıştır
	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return WrapError("rollback after error", rbErr)
		}
		return err
	}

	// Başarı → commit
	return tx.Commit()
}

// Close -> Veritabanı bağlantısını kapatır.
func (d *DB) Close() error {
	return d.DB.Close()
}

// Ping -> Bağlantı canlı mı? Kontrol eder.
func (d *DB) Ping(ctx context.Context) error {
	return d.DB.PingContext(ctx)
}
