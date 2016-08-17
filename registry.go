package aeio

import (
	"errors"
	"fmt"
	"reflect"
)

//TODO: should check if is already registered and panic (so to not compile).
var models = make(map[string]interface{})

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

//TODO: should check if models exist.
//use empty structs so the search is trivial: children[p][c] using ok idiom.
var children = make(map[string]map[string]struct{})

func RegisterChild(p string, c string) {
	if p != "" && models[p] == nil {
		panic(fmt.Sprintln("parent model", p, "is not registered"))
	}

	if c == "" || models[c] == nil {
		panic(fmt.Sprintln("model", c, "is not registered"))
	}

	if children[p] == nil {
		children[p] = make(map[string]struct{})
	}

	children[p][c] = struct{}{}
}

func TestPaternity(p string, c string) (err error) {
	_, ok := children[p][c]
	if !ok {
		err = errors.New("[" + p + "] kind doesn't accept the paternity of [" + c + "] kids. You should register it first.")
	}
	return
}
