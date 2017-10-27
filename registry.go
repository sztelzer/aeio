package aeio

import (
	"errors"
	"fmt"
	"reflect"

	"google.golang.org/appengine/datastore"
	// "google.golang.org/appengine/log"
	"encoding/gob"
)

//models are the backbone of AEIO. They allow AEIO to instantiate new objects.
var models = make(map[string]interface{})

func RegisterModel(m string, o interface{}) {
	if _, dup := models[m]; dup {
		panic("aeio: Register called twice for model " + m)
	}
	gob.Register(o)
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

// children allowed to specific models.
// register them in the init of models, after all models have been registered.
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

func TestKeyChainPaternity(k *datastore.Key) (err error) {
	for {
		kind := k.Kind()
		if k.Parent() != nil {
			k = k.Parent()
			err = TestPaternity(k.Kind(), kind)
			if err != nil {
				return
			}
			continue
		}
		//no more parent, test for ""
		err = TestPaternity("", k.Kind())
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
		panic(fmt.Sprintln("function", f, "is alredy registered on model", m))
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
