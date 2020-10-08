package aeio

import (
	"cloud.google.com/go/datastore"
	"encoding/gob"
	"errors"
	"fmt"
	"reflect"
)

// models allow aeio to instantiate new objects based on keys and paths.
var models = make(map[string]Data)

func RegisterModel(alias string, model Data) {
	if _, ok := models[alias]; ok {
		panic("aeio: Register called twice for model " + alias)
	}
	gob.Register(model)
	models[alias] = model
}

func NewObject(alias string) (Data, error) {
	if models[alias] == nil {
		return nil, errors.New("Resource " + alias + " is not implemented.")
	}
	val := reflect.ValueOf(models[alias])
	if val.Kind() == reflect.Ptr {
		val = reflect.Indirect(val)
	}
	newObj := reflect.New(val.Type()).Interface().(Data)

	return newObj, nil
}

// children allowed to specific models.
// register them in the init of models, after all models have been registered.
var children = make(map[string]map[string]struct{})

func RegisterChild(parent string, child string) {
	if parent != "" && models[parent] == nil {
		panic(fmt.Sprintln("parent model", parent, "is not registered or defined"))
	}
	if child == "" || models[child] == nil {
		panic(fmt.Sprintln("model", child, "is not registered or defined"))
	}
	if children[parent] == nil {
		children[parent] = make(map[string]struct{})
	}
	children[parent][child] = struct{}{}
}

// ValidatePaternity simply verifies that the parent key can have this kind of child.
func ValidatePaternity(p string, c string) (err error) {
	_, ok := children[p][c]
	if !ok {
		err = errors.New("[" + p + "] kind doesn't accept the paternity of [" + c + "] kids. You should register it first.")
	}
	return
}

// ValidateKey checks if the key chain is valid
func ValidateKey(k *datastore.Key) (err error) {
	for {
		kind := k.Kind
		if k.Parent != nil {
			k = k.Parent
			err = ValidatePaternity(k.Kind, kind)
			if err != nil {
				return
			}
			continue
		}
		// no more parent, test for ""
		err = ValidatePaternity("", k.Kind)
		if err != nil {
			return
		}
		return nil
	}
}

// functions allowed to specific models.
// register the allowed functions with the model, after registering it.
var functions = make(map[string]map[string]struct{})

func RegisterFunction(m string, f string) {
	if TestFunction(m, f) == nil {
		panic(fmt.Sprintln("function", f, "is already registered on model", m))
	}
	if functions[m] == nil {
		functions[m] = make(map[string]struct{})
	}
	functions[m][f] = struct{}{}
}

func TestFunction(m string, f string) error {
	_, ok := functions[m][f]
	if !ok {
		return errors.New(fmt.Sprintf("%v", "model"+m+"doesn't have the function"+f))
	}
	return nil
}

// actions maps what is being made with the resource so other parts of the code can be aware and take decisions.

var actions = map[string]struct{}{
	"error":    {},
	"create":   {},
	"read":     {},
	"readall":  {},
	"readany":  {},
	"update":   {},
	"save":     {},
	"hardsave": {},
	"patch":    {},
	"delete":   {},
}

func RegisterAction(action string) {
	_, ok := actions[action]
	if ok {
		panic(fmt.Sprintln("action", action, "is already registered"))
	}
	actions[action] = struct{}{}
}

func ValidAction(action string) {
	_, ok := actions[action]
	if !ok {
		panic(fmt.Sprintf("%v: %v", "invalid_action", action))
	}
}
