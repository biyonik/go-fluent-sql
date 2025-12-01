// Package fluentsql, Go dilinde akıcı SQL sorguları oluşturmayı ve yönetmeyi sağlayan bir kütüphanedir.
// Bu paket, hem bağlantılı hem de bağlantısız sorgu oluşturmayı destekler, raw SQL ifadelerini
// güvenli biçimde kullanmayı sağlar ve çeşitli konfigürasyon seçenekleri sunar.
//
// Yazar: Ahmet ALTUN
// Github: github.com/biyonik
// LinkedIn: linkedin.com/in/biyonik
// Email: ahmet.altun60@gmail.com
package fluentsql

import (
	"database/sql"

	"github.com/biyonik/go-fluent-sql/dialect"
)

// Version, go-fluent-sql kütüphanesinin mevcut sürümünü belirtir.
const Version = "0.1.0-alpha"

// Connect, verilen driver ve veri kaynağıyla yeni bir veritabanı bağlantısı oluşturur ve DB örneğini döndürür.
// Bağlantının doğruluğunu kontrol eder ve hata oluşursa WrapError ile sarar.
//
// Örnek:
//
//	db, err := fluentsql.Connect("mysql", "user:pass@tcp(localhost:3306)/dbname")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer db.Close()
func Connect(driverName, dataSourceName string, opts ...Option) (*DB, error) {
	sqlDB, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, WrapError("connect", err)
	}

	// Bağlantıyı doğrula
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, WrapError("ping", err)
	}

	return NewDB(sqlDB, opts...), nil
}

// ConnectWithConfig, Config yapısı kullanarak yeni bir veritabanı bağlantısı oluşturur.
// Bağlantı havuz ayarlarını uygular ve DSN oluşturur.
//
// Örnek:
//
//	cfg := &fluentsql.Config{
//	    Driver:   "mysql",
//	    Host:     "localhost",
//	    Port:     3306,
//	    Database: "mydb",
//	    Username: "user",
//	    Password: "pass",
//	}
//	db, err := fluentsql.ConnectWithConfig(cfg)
func ConnectWithConfig(cfg *Config, opts ...Option) (*DB, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	dsn := cfg.DSN()
	db, err := Connect(cfg.Driver, dsn, opts...)
	if err != nil {
		return nil, err
	}

	// Bağlantı havuz ayarlarını uygula
	if cfg.MaxOpenConns > 0 {
		db.DB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.DB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLife > 0 {
		db.DB.SetConnMaxLifetime(cfg.ConnMaxLife)
	}
	if cfg.ConnMaxIdle > 0 {
		db.DB.SetConnMaxIdleTime(cfg.ConnMaxIdle)
	}

	return db, nil
}

// New, veritabanı bağlantısı olmadan yeni bir Builder oluşturur.
// SQL stringleri oluşturmak ve sorgu yürütmeden hazırlık yapmak için kullanılır.
//
// Örnek:
//
//	qb := fluentsql.New()
//	sql, args, err := qb.Table("users").
//	    Select("id", "name").
//	    Where("status", "=", "active").
//	    ToSQL()
func New(opts ...Option) *Builder {
	// Sadece seçenekleri çıkarmak için geçici DB oluştur
	d := &DB{
		grammar: dialect.NewMySQLGrammar(),
		scanner: NewDefaultScanner(),
		logger:  NopLogger{},
	}

	for _, opt := range opts {
		opt(d)
	}

	return NewBuilder(nil, d.grammar, d.scanner)
}

// Table, yeni bir Builder oluşturup tablo adını ayarlamak için kısayoldur.
//
// Örnek:
//
//	sql, args, err := fluentsql.Table("users").
//	    Where("status", "=", "active").
//	    ToSQL()
func Table(name string) *Builder {
	return New().Table(name)
}

// Raw, kaçış yapılmayacak ham SQL ifadesini temsil eder.
// Sadece güvenli ve kontrol edilen girdi için kullanın.
//
// Bu tip, düzenli değerlerden ayırt edilebilmesi için bir işaretleyicidir.
type Raw struct {
	SQL      string
	Bindings []any
}

// NewRaw, yeni bir Raw SQL ifadesi oluşturur.
//
// Örnek:
//
//	qb.Select(fluentsql.NewRaw("COUNT(*) as total"))
//	qb.WhereRaw("YEAR(created_at) = ?", 2024)
func NewRaw(sql string, bindings ...any) Raw {
	return Raw{
		SQL:      sql,
		Bindings: bindings,
	}
}

// String, ham SQL ifadesini string olarak döndürür.
func (r Raw) String() string {
	return r.SQL
}
