package fluentsql

import (
	"database/sql"
	"errors"
	"reflect"
	"strings"
	"sync"
)

//
// =====================================================================================
// 📚 FLUENTSQL – SCANNER BİRİMİ
// -------------------------------------------------------------------------------------
// Bu dosya, veritabanından çekilen sonuçların Go struct’larına güvenli, hızlı ve
// otomatik şekilde aktarılmasını sağlayan *Scanner* altyapısını içerir.
// Amaç; veritabanı kayıtlarını manuel scan ve atama yükünden kurtarmak,
// `reflection + struct tag + cache` mantığıyla ORM benzeri otomatik doldurma yapmaktır.
//
// Bu sistemin çalışma biçimi:
//   1. Struct field’ları reflection ile taranır
//   2. `db:"column"` tag’lerine göre kolon–field eşlemesi oluşturulur
//   3. Çıkan sonuç cache’e alınır → tekrar eden scan’lar yüksek hızda çalışır
//   4. Row veya Rows nesnesi okunur, gelen veriler ilgili struct alanlarına yazılır
//
// Bu yapı özellikle ORM geliştirenlerin uzun vadede ihtiyaç duyduğu temel bileşendir.
// Çünkü veri dönüşümü *zor, maliyetli ve hata üretmeye müsaittir*.
// Ancak iyi kurulmuş bir scanner, ORM’in çekirdeği sayılabilir.
//
// Bu dosyada şunlar bulunur:
//   ✔ Scanner Interface          → tarama işlemi için standart kontrat
//   ✔ DefaultScanner             → varsayılan, tag tabanlı tarama sistemi
//   ✔ Struct metadata caching    → yüksek performans için tip analiz cache’i
//   ✔ Tek satır / çoklu satır / tek değer / tek kolon okuma fonksiyonları
//
// YAZAR BİLGİSİ
// @author    Ahmet ALTUN
// @github    github.com/biyonik
// @linkedin  linkedin.com/in/biyonik
// @email     ahmet.altun60@gmail.com
// =====================================================================================
//

// Scanner veritabanından okunan satırları Go modellerine map eden davranış sözleşmesidir.
// Bu interface’i implement eden her yapı, satırları struct’a veya slice’a dönüştürebilir.
type Scanner interface {
	// ScanRow → Tek satırı tek struct’a işler.
	// Burada beklenen davranış; row’dan verileri okuyup struct alanlarına set etmektir.
	ScanRow(row *sql.Row, dest any) error

	// ScanRows → Birden fazla satırı slice içine işler.
	// ORM kullanıyormuş hissi veren ana fonksiyondur.
	ScanRows(rows *sql.Rows, dest any) error
}

// DefaultScanner → Kütüphanenin standart tarama motorudur.
// Reflection kullanır, `db:"field"` tag’i ile eşleme yapar.
// Struct metadata bilgisi cache’de tutulduğu için yüksek performans sağlar.
type DefaultScanner struct {
	cache sync.Map // reflect.Type → structInfo
}

// NewDefaultScanner → Varsayılan scanner oluşturur.
// Dışarıdan bağımlılık gerektirmez, tek satırda çağrılır:
//    scanner := NewDefaultScanner()
func NewDefaultScanner() *DefaultScanner {
	return &DefaultScanner{}
}

// structInfo → Bir struct’ın kolon eşlemeleri ve metadata bilgisi.
// ORM'in beyni diyebileceğimiz tablodur.
type structInfo struct {
	fields  []fieldInfo       // Field listesi
	columns map[string]int    // Kolon adından index eşlemesi → O(1) lookup
}

// fieldInfo → Struct içerisindeki her alanın tarama bilgisi.
// Tag, index path, pk bilgisi gibi detayları taşır.
type fieldInfo struct {
	index     []int
	name      string
	isPK      bool
	omit      bool
	scanType  reflect.Type
	zeroValue reflect.Value
}

// ScanRow → Tek satırı karşılayan scanner fonksiyonudur.
// Struct pointer bekler, alanlar tek tek doldurulur.
// Row yok ise ErrNoRows döner.
func (s *DefaultScanner) ScanRow(row *sql.Row, dest any) error {
	if row == nil {
		return ErrNoRows
	}
	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return ErrNotAPointer
	}
	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return ErrNotAStruct
	}

	info := s.getStructInfo(elem.Type())
	scanDests := make([]any, len(info.fields))

	for i, f := range info.fields {
		if f.omit {
			var ignore any
			scanDests[i] = &ignore
			continue
		}
		fieldVal := elem.FieldByIndex(f.index)
		scanDests[i] = fieldVal.Addr().Interface()
	}

	err := row.Scan(scanDests...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNoRows
		}
		return WrapError("scan row", err)
	}

	return nil
}

// ScanRows → Çoklu sonuç tarayıcı.
// rows sonuç kümesini slice’a aktarır. (Users → []User şeklinde)
//
// ÖNEMLİ NOKTA:
// - Eğer hedef slice pointer değilse çalışmaz
// - Eğer slice element yapısı struct değilse hata döner
func (s *DefaultScanner) ScanRows(rows *sql.Rows, dest any) error {
	if rows == nil {
		return ErrNoRows
	}
	defer rows.Close()

	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return ErrNotAPointer
	}

	sliceVal := v.Elem()
	if sliceVal.Kind() != reflect.Slice {
		return ErrNotASlice
	}

	elemType := sliceVal.Type().Elem()
	isPtr := elemType.Kind() == reflect.Ptr
	if isPtr {
		elemType = elemType.Elem()
	}
	if elemType.Kind() != reflect.Struct {
		return ErrNotAStruct
	}

	columns, err := rows.Columns()
	if err != nil {
		return WrapError("get columns", err)
	}

	info := s.getStructInfo(elemType)
	columnToField := make([]int, len(columns))

	for i, col := range columns {
		col = strings.ToLower(col)
		if idx, ok := info.columns[col]; ok {
			columnToField[i] = idx
		} else {
			columnToField[i] = -1
		}
	}

	for rows.Next() {
		elemVal := reflect.New(elemType).Elem()
		scanDests := make([]any, len(columns))

		for i, fieldIdx := range columnToField {
			if fieldIdx == -1 {
				var ignore any
				scanDests[i] = &ignore
				continue
			}
			f := info.fields[fieldIdx]
			if f.omit {
				var ignore any
				scanDests[i] = &ignore
			} else {
				fieldVal := elemVal.FieldByIndex(f.index)
				scanDests[i] = fieldVal.Addr().Interface()
			}
		}

		if err := rows.Scan(scanDests...); err != nil {
			return WrapError("scan row", err)
		}

		if isPtr {
			sliceVal.Set(reflect.Append(sliceVal, elemVal.Addr()))
		} else {
			sliceVal.Set(reflect.Append(sliceVal, elemVal))
		}
	}

	if err := rows.Err(); err != nil {
		return WrapError("rows iteration", err)
	}

	return nil
}

// ScanValue → Tek kolon tek değer okuma.
// Sayım, tek field sonuçları gibi minimal sorgular için idealdir.
func (s *DefaultScanner) ScanValue(row *sql.Row, dest any) error {
	if row == nil {
		return ErrNoRows
	}
	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return ErrNotAPointer
	}

	err := row.Scan(dest)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNoRows
		}
		return WrapError("scan value", err)
	}

	return nil
}

// ScanColumn → sonuçların tek bir kolon olup slice’a yazıldığı senaryolar içindir.
// Örnek:
//   var ids []int
//   scanner.ScanColumn(rows, &ids)
func (s *DefaultScanner) ScanColumn(rows *sql.Rows, dest any) error {
	if rows == nil {
		return ErrNoRows
	}
	defer rows.Close()

	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return ErrNotAPointer
	}

	sliceVal := v.Elem()
	if sliceVal.Kind() != reflect.Slice {
		return ErrNotASlice
	}

	elemType := sliceVal.Type().Elem()

	for rows.Next() {
		elemPtr := reflect.New(elemType)
		if err := rows.Scan(elemPtr.Interface()); err != nil {
			return WrapError("scan column", err)
		}
		sliceVal.Set(reflect.Append(sliceVal, elemPtr.Elem()))
	}

	if err := rows.Err(); err != nil {
		return WrapError("rows iteration", err)
	}

	return nil
}

// getStructInfo → Struct metadata cache erişim fonksiyonu.
// Daha önce taranmışsa cache’den çeker → yüksek hız sağlar.
func (s *DefaultScanner) getStructInfo(t reflect.Type) *structInfo {
	if cached, ok := s.cache.Load(t); ok {
		return cached.(*structInfo)
	}

	info := &structInfo{
		fields:  make([]fieldInfo, 0),
		columns: make(map[string]int),
	}

	s.parseStruct(t, nil, info)
	s.cache.Store(t, info)

	return info
}

// parseStruct → Struct içindeki tüm alanları tarar.
// Gömülü struct’lar dahil derin tarama yapılır.
func (s *DefaultScanner) parseStruct(t reflect.Type, index []int, info *structInfo) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if !field.IsExported() {
			continue
		}

		fieldIndex := append(append([]int{}, index...), i)

		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			s.parseStruct(field.Type, fieldIndex, info)
			continue
		}

		tag := field.Tag.Get("db")
		if tag == "-" {
			continue
		}

		fi := fieldInfo{
			index:    fieldIndex,
			scanType: field.Type,
		}

		if tag != "" {
			parts := strings.Split(tag, ",")
			fi.name = parts[0]
			for _, part := range parts[1:] {
				if part == "pk" {
					fi.isPK = true
				}
			}
		} else {
			fi.name = strings.ToLower(field.Name)
		}

		idx := len(info.fields)
		info.fields = append(info.fields, fi)
		info.columns[fi.name] = idx
	}
}

// GetFieldNames → Struct içerisinde veritabanına karşılık gelen bütün kolon adlarını döner.
// SELECT * yerine SELECT id,name,email üretmek isteyen sistemler burada beslenir.
func (s *DefaultScanner) GetFieldNames(dest any) ([]string, error) {
	v := reflect.ValueOf(dest)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	var t reflect.Type
	if v.Kind() == reflect.Slice {
		t = v.Type().Elem()
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
	} else {
		t = v.Type()
	}

	if t.Kind() != reflect.Struct {
		return nil, ErrNotAStruct
	}

	info := s.getStructInfo(t)
	names := make([]string, 0, len(info.fields))

	for _, f := range info.fields {
		if !f.omit {
			names = append(names, f.name)
		}
	}

	return names, nil
}

// GetPrimaryKey → struct'ın birincil anahtar kolonunu döner.
// Eğer tanımlı değilse "id" fallback olarak kabul edilir.
func (s *DefaultScanner) GetPrimaryKey(dest any) string {
	v := reflect.ValueOf(dest)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()
	if t.Kind() != reflect.Struct {
		return ""
	}

	info := s.getStructInfo(t)

	for _, f := range info.fields {
		if f.isPK {
			return f.name
		}
	}

	if _, ok := info.columns["id"]; ok {
		return "id"
	}

	return ""
}
