package aeio

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"regexp"
	
	"cloud.google.com/go/datastore"
)

var validPath = regexp.MustCompile(`^(?:/[a-z-]+/[0-9]+)*(/[a-z-]+)?$`)

// models allow aeio to instantiate new objects based on keys and paths.
var models = make(map[string]interface{})

func RegisterModel(alias string, model interface{}) {
	if _, ok := models[alias]; ok {
		panic("aeio: Register called twice for model " + alias)
	}
	// gob.Register(model)
	models[alias] = model
}

func NewObject(alias string) (interface{}, error) {
	if models[alias] == nil {
		err := errors.New("Model " + alias + " is not implemented/registered.")
		return nil, errorResourceModelNotImplemented.withCause(err).withStack(10).withLog()
	}
	val := reflect.ValueOf(models[alias])
	if val.Kind() == reflect.Ptr {
		val = reflect.Indirect(val)
	}
	newObj := reflect.New(val.Type()).Interface()

	return newObj, nil
}

// patchers allow aeio to instantiate new patcher objects based on keys and paths.
// patchers are useful to allow only some fields of the original model, and use pointers to separate nil and zero values.
var patchers = make(map[string]interface{})

func RegisterPatcher(alias string, patcher interface{}) {
	if _, ok := patchers[alias]; ok {
		panic("aeio: Register called twice for patcher " + alias)
	}
	// gob.Register(patcher)
	patchers[alias] = patcher
}

func NewPatcher(alias string) (interface{}, error) {
	if patchers[alias] == nil {
		err := errors.New("Patcher " + alias + " is not implemented/registered.")
		return nil, errorResourceModelNotImplemented.withCause(err).withStack(10).withLog()
	}
	val := reflect.ValueOf(patchers[alias])
	if val.Kind() == reflect.Ptr {
		val = reflect.Indirect(val)
	}
	newPatcher := reflect.New(val.Type()).Interface()

	return newPatcher, nil
}

// children allowed to specific models.
// register them in the init of models, after all models have been registered.
var children = make(map[string]map[string]struct{})

func RegisterChild(parent string, child string) {
	// if parent != "" && models[parent] == nil {
	// 	panic(fmt.Sprintln("parent model", parent, "is not registered or defined"))
	// }
	// if child == "" || models[child] == nil {
	// 	panic(fmt.Sprintln("model", child, "is not registered or defined"))
	// }
	if children[parent] == nil {
		children[parent] = make(map[string]struct{})
	}
	children[parent][child] = struct{}{}
}

// CheckRegistry verifies if all relationships are for registered objects.
func CheckRegistry(print bool) {
	if print {
		log.Println(models)
		log.Println(children)
	}
	for parent, children := range children {
		for child := range children {
			if parent != "" && models[parent] == nil {
				log.Panicln("parent model", parent, "is not registered or defined")
			}
			if child == "" || models[child] == nil {
				log.Panicln("child model", child, "is not registered or defined")
			}
		}
	}
}

// ValidatePaternity simply verifies that the parent key can have this kind of child.
func ValidatePaternity(p string, c string) error {
	_, ok := children[p][c]
	if !ok {
		err := errors.New("[" + p + "] kind doesn't accept the paternity of [" + c + "] kids. You should register it first.")
		return errorInvalidResourceChild.withCause(err).withStack(10)
	}
	return nil
}

// ValidateKey checks if the key chain is valid, but don't check the existence of parents in datastore.
func ValidateKey(k *datastore.Key) error {
	if k == nil {
		return errorInvalidPath.withStack(10).withLog()
	}
	var level = 0
	for {
		kind := k.Kind
		if level != 0 && k.ID == 0 {
			return errorInvalidPath.withHint(fmt.Sprintf("key id 0 at level %d", level))
		}

		if k.Parent != nil {
			k = k.Parent
			err := ValidatePaternity(k.Kind, kind)
			if err != nil {
				return err
			}
			level--
			continue
		}
		// no more parents, test for root ""
		err := ValidatePaternity("", k.Kind)
		if err != nil {
			return err
		}
		return nil
	}
}

/*// functions allowed to specific models.
// register the allowed functions for the model after registering it.
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
		err := fmt.Errorf("model %s does not have function %s", m, f)
		return errorInvalidModelFunction.withCause(err).withStack(10)
	}
	return nil
}
*/
// actions maps what is being made with the resource so other parts of the code can be aware and take decisions.

const (
	ActionCreate   = "CREATE"
	ActionRead     = "GET"
	ActionReadMany = "GET-MANY"
	ActionReadAny  = "GET-ANY"
	ActionUpdate   = "UPDATE"
	ActionDelete   = "DELETE"
	ActionError    = "ERROR"

	ActionReadManyCount = "GET-MANY-COUNT"
)

var actions = map[string]struct{}{
	ActionCreate:        {},
	ActionRead:          {},
	ActionReadMany:      {},
	ActionReadManyCount: {},
	ActionReadAny:       {},
	ActionUpdate:        {},
	ActionDelete:        {},
	ActionError:         {},
}

// func RegisterAction(action string) {
// 	_, ok := actions[action]
// 	if ok {
// 		panic(fmt.Sprintf("action %s is already registered", action))
// 	}
// 	actions[action] = struct{}{}
// }

func ValidAction(action string) {
	_, ok := actions[action]
	if !ok {
		panic(fmt.Sprintf("invalid action: %s (action is not registered)", action))
	}
}
