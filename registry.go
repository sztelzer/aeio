package aeio

import (
	"errors"
	"reflect"
)

var ObjectsRegistry = map[string]reflect.Type{}

func RegisterModel(kind string, objInt interface{}) {
	ObjectsRegistry[kind] = reflect.TypeOf(objInt)
}

func NewObjectKind(kind string) (Object, error) {
	if ObjectsRegistry[kind] != nil {
		return nil, errors.New("Resource " + kind + " not implemented.")
	}
	object := reflect.New(ObjectsRegistry[kind]).Elem()
	concreteObject, _ := reflect.ValueOf(object).Interface().(Object)

	return concreteObject, nil
}
