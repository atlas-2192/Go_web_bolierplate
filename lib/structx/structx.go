package structx

import (
	"reflect"
)

func Get(object any, field string) any {
	reflectValue := reflect.ValueOf(object)
	reflectFieldValue := reflectValue.FieldByName(field)
	return reflectFieldValue.Interface()
}
