// Package validation, SQL sorgularında kullanılan tablolar, kolonlar ve alias isimlerini
// güvenli bir şekilde doğrulamak için dahili yardımcı fonksiyonlar sağlar. Bu paket,
// SQL enjeksiyon riskini azaltmak ve uygulamanın fluent query builder mantığıyla uyumlu
// çalışmasını sağlamak amacıyla tasarlanmıştır.
//
// Genel olarak, tablolar ve kolonlar üzerinde yapılan doğrulamalar şu sorulara cevap verir:
// 1. Bu isim geçerli bir SQL identifier mı? (harf, rakam, alt çizgi, noktalar)
// 2. Eğer alias kullanılmışsa, alias da geçerli mi?
// 3. Bu isim SQL rezerv kelimesi mi?
//
// Tüm fonksiyonlar, doğrulama başarısız olduğunda detaylı bir `IdentifierError` döndürür,
// böylece hata ayıklama ve kullanıcı bilgilendirmesi kolaylaşır.
//
// @author Ahmet ALTUN
// @github github.com/biyonik
// @linkedin linkedin.com/in/biyonik
// @email ahmet.altun60@gmail.com
package validation

import (
	"regexp"
	"strings"
)

// identifierRegex, SQL tabloları ve kolonları için geçerli identifier'ları doğrular.
// Geçerli karakterler: harfler, rakamlar, alt çizgi. İlk karakter harf veya alt çizgi olmalıdır.
// Noktalar (.) table.column referanslarını destekler.
var identifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*)?$`)

// aliasRegex, "table as alias" veya "table alias" formatlarını eşler.
var aliasRegex = regexp.MustCompile(`(?i)^([a-zA-Z_][a-zA-Z0-9_]*)\s+(?:as\s+)?([a-zA-Z_][a-zA-Z0-9_]*)$`)

// reservedWords, SQL rezerv kelimelerini içerir. Bu kelimeler, kullanıldığında
// genellikle tırnak içine alınmalıdır.
var reservedWords = map[string]bool{
	"select": true, "from": true, "where": true, "and": true, "or": true,
	"insert": true, "update": true, "delete": true, "into": true, "values": true,
	"set": true, "order": true, "by": true, "asc": true, "desc": true,
	"limit": true, "offset": true, "join": true, "left": true, "right": true,
	"inner": true, "outer": true, "on": true, "as": true, "in": true,
	"between": true, "like": true, "is": true, "null": true, "not": true,
	"group": true, "having": true, "distinct": true, "union": true,
	"create": true, "drop": true, "alter": true, "table": true, "index": true,
	"primary": true, "key": true, "foreign": true, "references": true,
	"default": true, "constraint": true, "unique": true, "check": true,
}

// ValidateIdentifier, verilen identifier'ın geçerli bir SQL identifier olup olmadığını kontrol eder.
// Başarılıysa nil döner, geçersizse açıklayıcı bir hata döner.
func ValidateIdentifier(id string) error {
	if id == "" {
		return &IdentifierError{
			Identifier: id,
			Reason:     "identifier cannot be empty",
		}
	}

	if len(id) > 128 {
		return &IdentifierError{
			Identifier: id,
			Reason:     "identifier exceeds maximum length of 128 characters",
		}
	}

	if !identifierRegex.MatchString(id) {
		return &IdentifierError{
			Identifier: id,
			Reason:     "identifier contains invalid characters; only letters, numbers, underscores, and dots are allowed",
		}
	}

	return nil
}

// ValidateTableWithAlias, bir tablo referansını (alias ile birlikte olabilir) doğrular.
// Desteklenen formatlar: "table", "table alias", "table as alias".
// Döndürür: tablo adı, alias (varsa) ve hata.
func ValidateTableWithAlias(table string) (name, alias string, err error) {
	if table == "" {
		return "", "", &IdentifierError{
			Identifier: table,
			Reason:     "table name cannot be empty",
		}
	}

	matches := aliasRegex.FindStringSubmatch(table)
	if matches != nil {
		name = matches[1]
		alias = matches[2]

		if err := ValidateIdentifier(name); err != nil {
			return "", "", err
		}
		if err := ValidateIdentifier(alias); err != nil {
			return "", "", &IdentifierError{
				Identifier: alias,
				Reason:     "invalid alias: " + err.Error(),
			}
		}

		return name, alias, nil
	}

	if err := ValidateIdentifier(table); err != nil {
		return "", "", err
	}

	return table, "", nil
}

// ValidateColumn, bir kolon referansını doğrular.
// Desteklenen formatlar: "column", "table.column".
func ValidateColumn(column string) error {
	return ValidateIdentifier(column)
}

// IsReservedWord, verilen identifier'ın SQL rezerv kelimesi olup olmadığını kontrol eder.
func IsReservedWord(id string) bool {
	return reservedWords[strings.ToLower(id)]
}

// SplitTableColumn, "table.column" formatındaki referansı parçalar.
// Döndürür: tablo (boşsa ""), kolon ve hata.
func SplitTableColumn(ref string) (table, column string, err error) {
	parts := strings.Split(ref, ".")
	switch len(parts) {
	case 1:
		if err := ValidateIdentifier(parts[0]); err != nil {
			return "", "", err
		}
		return "", parts[0], nil
	case 2:
		if err := ValidateIdentifier(parts[0]); err != nil {
			return "", "", err
		}
		if err := ValidateIdentifier(parts[1]); err != nil {
			return "", "", err
		}
		return parts[0], parts[1], nil
	default:
		return "", "", &IdentifierError{
			Identifier: ref,
			Reason:     "column reference can have at most one dot (table.column)",
		}
	}
}

// IdentifierError, identifier doğrulama hatalarını temsil eder.
type IdentifierError struct {
	Identifier string
	Reason     string
}

// Error, error arayüzünü uygular.
func (e *IdentifierError) Error() string {
	if e.Identifier == "" {
		return "fluentsql: invalid identifier: " + e.Reason
	}
	return "fluentsql: invalid identifier '" + e.Identifier + "': " + e.Reason
}
