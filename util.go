package history

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"reflect"
)

func makePtr(i interface{}) interface{} {
	v := reflect.ValueOf(i)
	if v.Kind() == reflect.Ptr {
		return i
	}

	p := reflect.New(v.Type())
	p.Elem().Set(v)

	return p.Interface()
}

func unsetStructField(i interface{}, name string) error {
	v := reflect.ValueOf(i)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("expected a ptr got %T", i)
	}

	iv := reflect.Indirect(v)
	if iv.Kind() != reflect.Struct {
		return fmt.Errorf("expected a struct got %T", i)
	}

	field := iv.FieldByName(name)
	if !field.IsValid() {
		return fmt.Errorf(`struct %s does not have field "%s"`, iv.Type(), name)
	}

	field.Set(reflect.Zero(field.Type()))

	return nil
}

func getPrimaryKeyValue(db *gorm.DB, value reflect.Value) (*primaryKeyField, error) {
	if !value.IsValid() {
		return nil, errors.New("primary key field cannot be detected on invalid value")
	}

	if db.Statement.Schema == nil {
		return nil, errors.New("primary key field cannot be detected because schema is not set")
	}

	value = reflect.Indirect(value)
	pkField := db.Statement.Schema.PrioritizedPrimaryField
	if pkField == nil {
		return nil, fmt.Errorf("primary key field could not be determined for %T", value.Interface())
	}

	pkValue := value.FieldByName(pkField.Name)

	isZero := pkValue.IsZero()
	if v, ok := pkValue.Interface().(IsZeroer); ok {
		isZero = v.IsZero()
	}

	return &primaryKeyField{
		name:   pkField.Name,
		value:  pkValue.Interface(),
		isZero: isZero,
	}, nil
}
