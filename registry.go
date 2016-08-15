package aeio

import (
	"errors"
	"reflect"
)

var models = make(map[string]interface{})


//TODO: should check if is already registered and panic (so to not compile).
func RegisterModel(m string, o interface{}) {
	models[m] = o
}

func NewObject(m string) (Object, error) {
	if models[m] == nil {
		return nil, errors.New("Resource " + m + " is not implemented.")
	}

	val := reflect.ValueOf(models[m])
	if val.Kind() == reflect.Ptr {
		val = reflect.Indirect(val)
	}
	new := reflect.New(val.Type()).Interface().(Object)

	return new, nil
}
