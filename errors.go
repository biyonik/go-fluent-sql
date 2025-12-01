package fluentsql

import (
	"errors"
	"fmt"
)

/*
   ================================================================================
   📌 go-fluent-sql HATA YÖNETİM KATMANI
   --------------------------------------------------------------------------------
   Bu dosya, fluent SQL sorgulamayı amaçlayan yapının *en kritik merkezlerinden
   biridir.* Modern ORM mantığında, hataların yalnızca oluşması değil — hangi
   bağlamda, ne sebeple ve nasıl üretildiğinin izlenebilir olması gerekir. Tam da
   bu nedenle; burada hem sabit hata tipleri (sentinel errors) hem de bağlam
   taşıyabilen hata yapıları tanımlanmıştır.

   Bu yaklaşım sayesinde geliştirici;
   - nerede hata aldığını,
   - hangi tablo üzerinde işlem yaptığını,
   - hangi SQL çıktısının üretildiğini,
   - hatanın asıl kaynağının ne olduğunu
   net ve merkezi şekilde izleyebilir.

   Özellikle `QueryError` ve `ValidationError` yapıları; Go’nun `errors.Is`,
   `errors.As`, `Unwrap()` modelleriyle tam uyumlu tasarlanmıştır. Böylece
   FluentSQL yalnızca bir query builder değil, aynı zamanda *profesyonel hata
   izleme mimarisi* sunar.

   Bu doküman şunları açıklar:
   • Neden özel hata tipleri kullanıyoruz?
     → Her sorgu farklı bağlam taşır. Bağlamı kaybetmemek işletimsel teşhisi hızlandırır.
   • Nasıl kullanıyoruz?
     → errors.Is(), errors.As(), context-wrap gibi Go standartlarına dayanarak.
   • Bu tasarım ne kazandırır?
     → Şeffaflık, izlenebilirlik, debug kolaylığı ve kurumsal ölçek sürdürülebilirliği.

   @author    Ahmet ALTUN
   @github    github.com/biyonik
   @linkedin  linkedin.com/in/biyonik
   @email     ahmet.altun60@gmail.com
   ================================================================================
*/

// -------------------------------------------------------------------------------
// 🚨 Sabit Hata Tanımları (Sentinel Errors)
// -------------------------------------------------------------------------------
// Bu bölümde çatı seviyede paket hataları bulunur.
// errors.Is() ile doğrudan tespit edilmesi amaçlanmıştır.
// -------------------------------------------------------------------------------

// Sentinel errors for go-fluent-sql.
// These errors can be checked using errors.Is().
var (
	// ErrNoRows is returned when a query returns no rows.
	ErrNoRows = errors.New("fluentsql: no rows in result set")

	// ErrNoTable is returned when no table is specified.
	ErrNoTable = errors.New("fluentsql: no table specified")

	// ErrNoColumns is returned when no columns are specified for insert/update.
	ErrNoColumns = errors.New("fluentsql: no columns specified")

	// ErrNoExecutor is returned when no database executor is set.
	ErrNoExecutor = errors.New("fluentsql: no database executor")

	// ErrInvalidIdentifier is returned for invalid SQL identifiers.
	ErrInvalidIdentifier = errors.New("fluentsql: invalid identifier")

	// ErrInvalidOperator is returned for disallowed operators.
	ErrInvalidOperator = errors.New("fluentsql: invalid operator")

	// ErrInvalidValue is returned for invalid values.
	ErrInvalidValue = errors.New("fluentsql: invalid value")

	// ErrNotAPointer is returned when dest is not a pointer.
	ErrNotAPointer = errors.New("fluentsql: destination must be a pointer")

	// ErrNotASlice is returned when dest is not a slice.
	ErrNotASlice = errors.New("fluentsql: destination must be a slice")

	// ErrNotAStruct is returned when dest element is not a struct.
	ErrNotAStruct = errors.New("fluentsql: destination element must be a struct")

	// ErrTxAlreadyClosed is returned when transaction is already committed/rolled back.
	ErrTxAlreadyClosed = errors.New("fluentsql: transaction already closed")

	// ErrEmptyBatch is returned when inserting empty batch.
	ErrEmptyBatch = errors.New("fluentsql: cannot insert empty batch")

	// ErrInconsistentBatch is returned when batch rows have different columns.
	ErrInconsistentBatch = errors.New("fluentsql: inconsistent columns in batch")

	// ErrEmptyWhereIn is returned when WhereIn is called with an empty slice.
	ErrEmptyWhereIn = errors.New("fluentsql: empty slice passed to WhereIn")

	// ErrInvalidBetweenValues is returned when WhereBetween doesn't receive exactly 2 values.
	ErrInvalidBetweenValues = errors.New("fluentsql: BETWEEN requires exactly 2 values")

	// ErrConnectionClosed is returned when trying to use a closed connection.
	ErrConnectionClosed = errors.New("fluentsql: connection closed")

	// ErrQueryTimeout is returned when a query exceeds the context deadline.
	ErrQueryTimeout = errors.New("fluentsql: query timeout exceeded")
)


// -------------------------------------------------------------------------------
// 🏷 QueryError
// -------------------------------------------------------------------------------
// - Amaç: Sorgu işlemlerinde bağlam kaybı olmadan hata taşımak.
// - Neden var?: Hatanın "hangi tablo", "hangi operasyon", "hangi SQL çıktısı"
//   ile ilişkili olduğunun tek bakışta anlaşılması gerekir.
// - Kullanım: `return &QueryError{ ... }` şeklinde veya `NewQueryError()` ile üretilir.
//   errors.Unwrap() ile alt hata geri alınabilir.
// ------------------------------------------------------------------------------
type QueryError struct {
	Op    string // Operation: "select", "insert", "update", "delete", "compile"
	Table string // Table name
	SQL   string // Generated SQL (sanitized, no actual values)
	Err   error  // Underlying error
}

// Error implements the error interface.
// Bu fonksiyon hata mesajını okunabilir formatta döndürür.
// Eğer tablo adı mevcut ise → "select on table X" şeklinde detaylı yazılır.
func (e *QueryError) Error() string {
	if e.Table != "" {
		return fmt.Sprintf("fluentsql: %s on table %q: %v", e.Op, e.Table, e.Err)
	}
	return fmt.Sprintf("fluentsql: %s: %v", e.Op, e.Err)
}

// Unwrap returns the underlying error.
// Amaç: Go'nun error chain mekanizması ile hatanın köküne ulaşabilmek.
func (e *QueryError) Unwrap() error {
	return e.Err
}

// NewQueryError creates a new QueryError with context.
// Bu yardımcı fonksiyon hata üretimini standartlaştırır; proje genelinde
// tek tip format ve izlenebilirlik sağlar.
func NewQueryError(op, table, sql string, err error) *QueryError {
	return &QueryError{
		Op:    op,
		Table: table,
		SQL:   sql,
		Err:   err,
	}
}


// -------------------------------------------------------------------------------
// 🏷 ValidationError
// -------------------------------------------------------------------------------
// - Amaç: Identifier, operator veya value geçersiz olduğunda anlamlı geri dönüş üretmek.
// - Neden özel struct?: Çünkü bir hatanın yalnızca oluşması değil, *neden* oluştuğu
//   da önemlidir. Örn: "identifier geçersiz" vs "value geçersiz" → farklı kök sebepler.
// - errors.Is() override edilmiştir, böylece ErrInvalidIdentifier gibi sabit
//   hatalarla eşleştirilebilir.
// ------------------------------------------------------------------------------
type ValidationError struct {
	Type   string // "identifier", "operator", "value"
	Value  string // The invalid value
	Reason string // Why it's invalid
}

// Error implements the error interface.
// Hatanın insan tarafından anlaşılabilir string halini döndürür.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("fluentsql: invalid %s %q: %s", e.Type, e.Value, e.Reason)
}

// Is allows errors.Is() to match against sentinel errors.
// Böylece errors.Is(err, ErrInvalidIdentifier) → true olabilir.
func (e *ValidationError) Is(target error) bool {
	switch e.Type {
	case "identifier":
		return target == ErrInvalidIdentifier
	case "operator":
		return target == ErrInvalidOperator
	case "value":
		return target == ErrInvalidValue
	default:
		return false
	}
}

// NewValidationError creates a new ValidationError.
// Kullanımı basitleştirilmiş factory fonksiyondur.
func NewValidationError(typ, value, reason string) *ValidationError {
	return &ValidationError{
		Type:   typ,
		Value:  value,
		Reason: reason,
	}
}


// -------------------------------------------------------------------------------
// 🔄 WrapError
// -------------------------------------------------------------------------------
// Amaç: Operasyon ismiyle birlikte hata zinciri oluşturmak.
// Kullanım: return WrapError("insert", err)
// Sonuç: `fluentsql: insert: <wrapped error>` şeklinde takip edilebilir output üretir.
// ------------------------------------------------------------------------------
func WrapError(op string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("fluentsql: %s: %w", op, err)
}
