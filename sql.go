package goption

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"time"
)

// Scan implements sql.Scanner for Options
func (o *Option[T]) Scan(src any) error {
	if src == nil {
		*o = None[T]()
		return nil
	}

	// Try scanning
	var maybeScanner any = &o.t
	if scanner, isScanner := maybeScanner.(sql.Scanner); isScanner {
		o.ok = true
		return scanner.Scan(src)
	}

	// Try reflecting
	srcVal := reflect.ValueOf(src)
	tType := reflect.TypeOf(o.t)
	if srcVal.CanConvert(tType) {
		reflect.ValueOf(&o.t).Elem().Set(srcVal.Convert(tType))
		o.ok = true
		return nil
	}

	return ErrNotAScanner
}

type errNotAScanner struct{}

func (errNotAScanner) Error() string {
	return "Not a scanner"
}

var ErrNotAScanner errNotAScanner

func (o Option[T]) Value() (driver.Value, error) {
	if !o.ok {
		return nil, nil
	}

	var maybeValuer any = o.t
	if valuer, isValuer := maybeValuer.(driver.Valuer); isValuer {
		return valuer.Value()
	}

	tVal := reflect.ValueOf(o.t)
	switch tVal.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return tVal.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return int64(tVal.Uint()), nil
	case reflect.Uint64:
		u64 := tVal.Uint()
		if u64 >= 1<<63 {
			return nil, fmt.Errorf("uint64 values with high bit set are not supported")
		}
		return int64(u64), nil
	case reflect.Float32, reflect.Float64:
		return tVal.Float(), nil
	case reflect.Bool:
		return tVal.Bool(), nil
	case reflect.Slice:
		ek := tVal.Type().Elem().Kind()
		if ek == reflect.Uint8 {
			return tVal.Bytes(), nil
		}
	case reflect.String:
		return tVal.String(), nil
	}

	int64Type := reflect.TypeOf(int64(0))
	if tVal.CanConvert(int64Type) {
		return tVal.Convert(int64Type).Interface(), nil
	}
	f64Type := reflect.TypeOf(float64(0))
	if tVal.CanConvert(f64Type) {
		return tVal.Convert(f64Type).Interface(), nil
	}
	boolType := reflect.TypeOf(false)
	if tVal.CanConvert(boolType) {
		return tVal.Convert(boolType).Interface(), nil
	}
	bytesType := reflect.TypeOf([]byte(nil))
	if tVal.CanConvert(bytesType) {
		return tVal.Convert(bytesType).Interface(), nil
	}
	stringType := reflect.TypeOf("")
	if tVal.CanConvert(stringType) {
		return tVal.Convert(stringType).Interface(), nil
	}
	timeType := reflect.TypeOf(time.Time{})
	if tVal.CanConvert(timeType) {
		return tVal.Convert(timeType).Interface(), nil
	}

	return o.t, nil
}
