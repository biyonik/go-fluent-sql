// Package validation, SQL sorgularında kullanılan operatörlerin doğrulanması ve normalizasyonu
// işlemlerini sağlayan dahili yardımcı fonksiyonları içerir. Bu paket, güvenli SQL sorguları
// oluşturmak için operatörlerin beyaz listeye uygunluğunu denetler ve standart formata çevirir.
//
// Yazar: Ahmet ALTUN
// Github: github.com/biyonik
// LinkedIn: linkedin.com/in/biyonik
// Email: ahmet.altun60@gmail.com
package validation

import "strings"

// allowedOperators, güvenli kabul edilen SQL operatörlerini tanımlar.
// Yalnızca bu operatörler WHERE cümlelerinde kullanılabilir.
var allowedOperators = map[string]bool{
	// Karşılaştırma operatörleri
	"=":  true,
	"!=": true,
	"<>": true,
	"<":  true,
	">":  true,
	"<=": true,
	">=": true,

	// Desen eşleştirme operatörleri
	"LIKE":     true,
	"NOT LIKE": true,

	// NULL kontrolü operatörleri
	"IS":     true,
	"IS NOT": true,

	// Set operatörleri (içeride değerleri ayrıca doğrulanır)
	"IN":          true,
	"NOT IN":      true,
	"BETWEEN":     true,
	"NOT BETWEEN": true,

	// Ek karşılaştırma
	"<=>": true, // MySQL NULL güvenli eşitliği
}

// ValidateOperator, verilen operatörün izin verilen listede olup olmadığını kontrol eder.
// Operatörler, kontrol öncesinde büyük harfe çevrilir ve boşlukları kırpılır.
func ValidateOperator(op string) error {
	normalized := strings.ToUpper(strings.TrimSpace(op))

	if !allowedOperators[normalized] {
		return &OperatorError{
			Operator: op,
			Reason:   "operator not in allowed list",
		}
	}

	return nil
}

// NormalizeOperator, bir operatörü standart biçime normalleştirir.
// Geçersiz operatör girilirse hata döner.
func NormalizeOperator(op string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(op))

	if !allowedOperators[normalized] {
		return "", &OperatorError{
			Operator: op,
			Reason:   "operator not in allowed list",
		}
	}

	return normalized, nil
}

// IsComparisonOperator, operatörün temel karşılaştırma operatörü olup olmadığını döndürür.
func IsComparisonOperator(op string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(op))
	switch normalized {
	case "=", "!=", "<>", "<", ">", "<=", ">=", "<=>":
		return true
	default:
		return false
	}
}

// IsPatternOperator, operatörün desen eşleştirme operatörü (LIKE / NOT LIKE) olup olmadığını döndürür.
func IsPatternOperator(op string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(op))
	return normalized == "LIKE" || normalized == "NOT LIKE"
}

// IsNullOperator, operatörün NULL kontrolü (IS / IS NOT) olup olmadığını döndürür.
func IsNullOperator(op string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(op))
	return normalized == "IS" || normalized == "IS NOT"
}

// AllowedOperators, izin verilen tüm operatörleri döndürür.
// Dokümantasyon veya hata mesajları için faydalıdır.
func AllowedOperators() []string {
	ops := make([]string, 0, len(allowedOperators))
	for op := range allowedOperators {
		ops = append(ops, op)
	}
	return ops
}

// OperatorError, operatör doğrulama hatasını temsil eder.
type OperatorError struct {
	Operator string
	Reason   string
}

// Error, error arayüzünü uygular ve hatayı açıklayıcı string olarak döner.
func (e *OperatorError) Error() string {
	return "fluentsql: invalid operator '" + e.Operator + "': " + e.Reason
}
