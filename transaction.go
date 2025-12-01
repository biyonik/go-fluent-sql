package fluentsql

import (
	"context"
	"database/sql"
	"sync"
)

// -----------------------------------------------------------------------------
//  Transaction Yapısı — Bir İşlemin Kalbi, Bir Akışın Güvencesi
//
//  Bu dosya, veritabanı işlemlerinin atomik yürütülmesini sağlayan *Transaction*
//  nesnesinin tüm fonksiyonel yeteneklerini barındırır. Amaç yalnızca SQL
//  sorgularını ardışık şekilde çalıştırmak değil; her adımı güvenli, kontrol
//  edilebilir ve gerektiğinde geri alınabilir bir bütün olarak yönetmektir.
//
//  Tıpkı Laravel’in transactions() yapısı veya Doctrine DBAL üzerindeki transactional
//  kontrol gibi; burada da ince düşünülmüş bir akış modeli vardır:
//
//   • Aynı bağlantı üzerinde birden çok sorgu art arda yürütülebilir
//   • İşlem onaylanırsa kalıcı olur (Commit)
//   • İşlem iptal edilirse hiçbir şey olmamış gibi geri alınır (Rollback)
//   • Hatta daha ileri seviye geliştirme için — Savepoint/RollbackTo desteği ile
//     alt seviyede geri dönüş imkânı sunar. (Büyük uygulamalarda hayat kurtarır.)
//
//  Concurrency (eş-zamanlı erişim) kontrollüdür. Aynı transaction birden çok
//  goroutine tarafından kullanılmamalıdır; eğer kullanılacaksa gerekli kilit
//  mekanizmaları senkronizasyonla sağlanmalıdır.
//
//  -- @author   Ahmet ALTUN
//  -- @github   github.com/biyonik
//  -- @linkedin linkedin.com/in/biyonik
//  -- @email    ahmet.altun60@gmail.com
// -----------------------------------------------------------------------------

// Transaction struct'ı, bir SQL transaction’ı temsil eder ve Query Builder entegrasyonu
// ile birlikte çalışır. Aynı *sql.Tx* üzerinden sorgular zincir halinde yürütülür.
// Ancak dikkat: Bu yapı thread-safe değildir. Her goroutine kendi Transaction
// nesnesini kullanmalıdır.
type Transaction struct {
	tx      *sql.Tx
	grammar Grammar
	scanner Scanner
	logger  Logger
	debug   bool
	prefix  string

	mu     sync.Mutex
	closed bool
}

// Table metodu, transaction kapsamında kullanılmak üzere yeni bir Builder üretir.
// Amaç: Transaction içerisinde akıcı query üretmeyi sürdürmek.
//
// Örnek:
//
//	tx, _ := db.Begin()
//	_, err := tx.Table("users").Where("id", "=", 1).Update(data)
func (t *Transaction) Table(name string) *Builder {
	return NewBuilder(t.tx, t.grammar, t.scanner).Table(name)
}

// Commit metodu, yapılan tüm işlemleri kalıcı hale getirir. Bir kez işlendiğinde
// transaction kapanır ve tekrar kullanılmaya çalışılırsa ErrTxAlreadyClosed üretir.
//
// Kullanım Amacı:
// • İşlemlerin başarıyla tamamlandığını onaylamak
// • Sistem bütünlüğünü korumak
func (t *Transaction) Commit() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return ErrTxAlreadyClosed
	}

	t.closed = true
	if err := t.tx.Commit(); err != nil {
		return WrapError("commit transaction", err)
	}
	return nil
}

// Rollback metodu — commit'in tam tersidir. Tüm değişiklikleri geri alır.
// Çok önemli not: Rollback tekrar tekrar çağıldığında hata vermez, idempotenttir.
//
// Kullanım Senaryosu:
// • Hata oluştuğunda işlemi geri almak
// • Bir adım yanlış gittiğinde sistem tutarlılığını korumak
func (t *Transaction) Rollback() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil // Rollback is idempotent
	}

	t.closed = true
	if err := t.tx.Rollback(); err != nil {
		if err == sql.ErrTxDone {
			return nil
		}
		return WrapError("rollback transaction", err)
	}
	return nil
}

// IsClosed transaction'ın commit ya da rollback sonrası kapanıp kapanmadığını bildirir.
// Bu, işlem akışını kontrol ederken önemli bir güvenlik kilidi işlevi görür.
func (t *Transaction) IsClosed() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.closed
}

// ExecContext metodu — Builder kullanmaya gerek kalmadan raw SQL çalıştırmak için
// doğrudan erişim sunar.
//
// Kullanım Alanı:
// • Özel SQL ifadeleri
// • Builder’ın desteklemediği query tipleri
func (t *Transaction) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, ErrTxAlreadyClosed
	}
	t.mu.Unlock()

	return t.tx.ExecContext(ctx, query, args...)
}

// QueryContext — result set döndüren SELECT benzeri işlemler için kullanılır.
// Dönen sonuç satır tabanlıdır (*sql.Rows*).
func (t *Transaction) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, ErrTxAlreadyClosed
	}
	t.mu.Unlock()

	return t.tx.QueryContext(ctx, query, args...)
}

// QueryRowContext — tek satır dönen sorgular içindir. Örneğin LIMIT 1,
// COUNT(), MAX() gibi yapılar için tercih edilir.
func (t *Transaction) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return t.tx.QueryRowContext(ctx, "SELECT 1 WHERE 1=0")
	}
	t.mu.Unlock()

	return t.tx.QueryRowContext(ctx, query, args...)
}

// Grammar — transaction seviyesinde kullanılan SQL sözdizimini döndürür.
func (t *Transaction) Grammar() Grammar {
	return t.grammar
}

// Scanner — transaction’ın tarayıcı (struct-bind) bileşenini döndürür.
func (t *Transaction) Scanner() Scanner {
	return t.scanner
}

// Savepoint — büyük transaction blokları arasında güvenli dönüş noktası oluşturur.
// Tüm işlemi bozmak yerine yalnızca belirli bölümü geri almak için kullanılır.
//
// Not: Her veritabanı savepoint desteklemez.
func (t *Transaction) Savepoint(name string) error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return ErrTxAlreadyClosed
	}
	t.mu.Unlock()

	if name == "" {
		return NewValidationError("identifier", name, "savepoint name cannot be empty")
	}

	_, err := t.tx.Exec("SAVEPOINT " + name)
	if err != nil {
		return WrapError("create savepoint", err)
	}
	return nil
}

// RollbackTo — transaction’ı tamamen geri almadan yalnızca belirli savepoint’e
// dönüş sağlar. Büyük sistemlerde geri dönüş maliyetini minimize eder.
func (t *Transaction) RollbackTo(name string) error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return ErrTxAlreadyClosed
	}
	t.mu.Unlock()

	if name == "" {
		return NewValidationError("identifier", name, "savepoint name cannot be empty")
	}

	_, err := t.tx.Exec("ROLLBACK TO SAVEPOINT " + name)
	if err != nil {
		return WrapError("rollback to savepoint", err)
	}
	return nil
}

// ReleaseSavepoint — oluşturulan savepoint’i serbest bırakır.
// Bu işlem transaction’ı bitirmez, yalnızca savepoint'i temizler.
func (t *Transaction) ReleaseSavepoint(name string) error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return ErrTxAlreadyClosed
	}
	t.mu.Unlock()

	if name == "" {
		return NewValidationError("identifier", name, "savepoint name cannot be empty")
	}

	_, err := t.tx.Exec("RELEASE SAVEPOINT " + name)
	if err != nil {
		return WrapError("release savepoint", err)
	}
	return nil
}

// Tx — alttaki *sql.Tx* referansına doğrudan erişim sağlar.
// ⚠ Kullanırken dikkatli olunmalıdır. Transaction metodları güvenlik
// kontrolleri içerirken raw erişim içermez.
func (t *Transaction) Tx() *sql.Tx {
	return t.tx
}
