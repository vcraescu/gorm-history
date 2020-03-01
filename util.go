package history

import (
	"errors"
	"fmt"
	"reflect"
)

type Version int64

type FieldNotFoundError struct {
	name string
}

func (e FieldNotFoundError) Error() string {
	return fmt.Sprintf("field not found: %s", e.name)
}

func isHistory(h interface{}) bool {
	_, ok := h.(History)
	if ok {
		return true
	}

	return len(GetHistoryFields(h)) > 0
}

func getFieldsByTag(i interface{}, tag string) []reflect.StructField {
	t := reflect.TypeOf(i)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	fieldsMap := make(map[string]reflect.StructField)
	v := reflect.Indirect(reflect.ValueOf(i))
	for i := 0; i < t.NumField(); i++ {
		ft := t.Field(i)
		fv := v.Field(i)

		if _, ok := fieldsMap[ft.Name]; ok {
			continue
		}

		if _, ok := ft.Tag.Lookup(tag); ok {
			fieldsMap[ft.Name] = ft
			continue
		}

		if fv.Type().Kind() == reflect.Ptr && fv.IsNil() {
			continue
		}

		if entry, ok := reflect.Indirect(fv).Interface().(Entry); ok {
			for _, f := range getFieldsByTag(entry, tag) {
				fieldsMap[f.Name] = f
			}

			continue
		}
	}

	fields := make([]reflect.StructField, len(fieldsMap))
	k := 0
	for _, f := range fieldsMap {
		fields[k] = f
		k++
	}

	return fields
}

func GetHistoryFields(i interface{}) []reflect.StructField {
	return getFieldsByTag(i, Tag)
}

func setHistoryFields(i interface{}, objectID interface{}, version Version, action Action) (err error) {
	h, ok := i.(History)
	if ok {
		h.SetHistoryObjectID(objectID)
		h.SetHistoryVersion(version)
		h.SetHistoryAction(action)

		return nil
	}

	fields := GetHistoryFields(i)
	valuesToSetMap := map[string]interface{}{
		FieldTagObjectID: objectID,
		FieldTagVersion:  version,
		FieldTagAction:   action,
	}

	for _, field := range fields {
		name := field.Tag.Get(Tag)
		value, ok := valuesToSetMap[name]
		if !ok {
			continue
		}

		if err := setStructFieldByName(i, field.Name, value); err != nil {
			if _, ok := err.(FieldNotFoundError); !ok {
				return err
			}
		}
	}

	return nil
}

func setStructFieldByName(i interface{}, name string, value interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf(`error setting field "%s" value: %v`, name, r)
		}
	}()

	v := reflect.ValueOf(i)
	if v.Kind() != reflect.Ptr {
		err = errors.New("not a pointer")
		return
	}

	v = v.Elem()
	if v.Kind() != reflect.Struct {
		err = errors.New("not a struct pointer")
		return
	}

	field := v.FieldByName(name)
	if !field.IsValid() {
		return FieldNotFoundError{name: name}
	}

	if field.Kind() == reflect.Ptr {
		field.Set(reflect.New(field.Type().Elem()))
	}

	vValueToSet := reflect.ValueOf(value)
	if field.Kind() == reflect.Ptr {
		vValueToSet = reflect.New(field.Type().Elem())
		vValueToSet.Elem().Set(reflect.ValueOf(value))
	}

	field.Set(vValueToSet)

	return nil
}
