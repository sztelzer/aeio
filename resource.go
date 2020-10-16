package aeio

import (
	"cloud.google.com/go/datastore"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Resource is the main structure holding meta data of connection, the data itself, and many methods of control.
// The is simple, typical lifecycle of a resource:
// For each connection a selected handler initializes a RootResource that, if has a numerical id, will try to initialize
// a complete datastore.Key, or if is a kind request, an incomplete datastore.Key.
// Being complete, it can retrieve it's data from datastore and other services.
// Being incomplete, it can use the request data to build the data and store, giving back the complete key.
type Resource struct {
	Key            *datastore.Key `datastore:"-" json:"-"`
	Data           interface{}    `datastore:"-" json:"data,omitempty"`
	error          error          `datastore:"-"`
	CreatedAt      time.Time      `datastore:"-" json:"created_at,omitempty"`
	Access         *Access        `datastore:"-" json:"-"`
	ActionsStack   []string       `datastore:"-" json:"-"`
	ActionsHistory []string       `datastore:"-" json:"-"`
	Resources      []*Resource    `datastore:"-" json:"resources,omitempty"`
	ResourcesCount int            `datastore:"-" json:"resources_count,omitempty"`
	Next           string         `datastore:"-" json:"next,omitempty"`
	TimeElapsed    int64          `datastore:"-" json:"time_elapsed,omitempty"`
}

type DataBeforeSave interface {
	BeforeSave(*Resource) error
}

type DataAfterSave interface {
	AfterSave(*Resource) error
}

type DataBeforeLoad interface {
	BeforeLoad(*Resource) error
}

type DataAfterLoad interface {
	AfterLoad(*Resource) error
}

type DataBeforeDelete interface {
	BeforeDelete(*Resource) error
}

type DataAfterDelete interface {
	AfterDelete(*Resource) error
}

// NewResourceFromRequest initializes a base resource with information from the request.
func NewResourceFromRequest(writer *http.ResponseWriter, request *http.Request) (*Resource, error) {
	r := &Resource{}
	r.Access = newAccess(writer, request)
	r.Key = Key(r.Access.Request.URL.Path)
	if r.Key == nil {
		return r, errorInvalidPath.withStack()
	}
	return r, nil
}

// InitResource uses a Key to initialize a specific resource beyond the root resource. The resource returned is ready to be actioned.
// Examples are returning sub resources or even something outside the scope of root.
func InitResource(access *Access, key *datastore.Key) (r *Resource) {
	return &Resource{Key: key, Access: access}
}

// NewResource is used to create empty children resources. Parent path may be the "" string (root). It returns a resource with an incompleteKey.
// It initializes an object of type kind.
// TODO: swap access and parentKey for parentResource
func NewResource(access *Access, parentKey *datastore.Key, kind string) (*Resource, error) {
	r := &Resource{Access: access}

	err := ValidatePaternity(parentKey.Kind, kind)
	if err != nil {
		return nil, err
	}

	r.Key = Key(Path(parentKey) + "/" + kind)
	if r.Key == nil {
		return nil, errorInvalidPath.withStack()
	}
	r.Data, err = NewObject(kind)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// // NewList return a new resource with the list type added to the key.
// func NewListResource(parentResource *Resource, listKind string) (r *Resource) {
// 	return NewResource(parentResource.Access, parentResource.Key, listKind)
// }

// Save puts the object into datastore, inlining CreatedAt and Parent in the object.
func (r *Resource) Save() (ps []datastore.Property, err error) {
	r.CreatedAt = NoZeroTime(r.CreatedAt)
	ps, err = datastore.SaveStruct(r.Data)
	if err != nil {
		return nil, err
	}
	ps = append(ps, datastore.Property{Name: "CreatedAt", Value: r.CreatedAt})
	if r.Key != nil {
		ps = append(ps, datastore.Property{Name: "Parent", Value: r.Key.Parent})
	}
	return ps, nil
}

// Load extracts the datastore data in an object, taking CreatedAt and Parent off the object.
func (r *Resource) Load(ps []datastore.Property) (err error) {
	var ps2 []datastore.Property
	for _, p := range ps {
		switch p.Name {
		case "CreatedAt":
			r.CreatedAt = p.Value.(time.Time)
		case "Parent":
		default:
			ps2 = append(ps2, p)
		}
	}
	err = datastore.LoadStruct(r.Data, ps2)
	return
}

// Key transforms a path in a datastore key using an access.
func Key(path string) (k *datastore.Key) {
	var kd string
	var id int64
	var p []string

	if path == "" {
		log.Print("path to key from empty path")
		return nil
	}

	if validPath.MatchString(path) != true {
		log.Print("invalid_path")
		return nil
	}

	p = strings.SplitN(path, "/", 2)
	p = strings.Split(p[1], "/")
	for i := 0; i < len(p); i = i + 2 {
		kd = p[i]
		if i < len(p)-1 {
			id, _ = strconv.ParseInt(p[i+1], 10, 64)
			k = datastore.IDKey(kd, id, k)
		} else {
			k = datastore.IncompleteKey(kd, k)
		}
	}
	return k
}

// Path transforms a datastore Key into a Path
func Path(k *datastore.Key) (p string) {
	if k.Incomplete() == false {
		p = "/" + strconv.FormatInt(k.ID, 10)
	}
	p = "/" + k.Kind + p
	for {
		k = k.Parent
		if k == nil {
			break
		}
		p = "/" + k.Kind + "/" + strconv.FormatInt(k.ID, 10) + p
	}
	return p
}

func (r *Resource) NewData(kind string) error {
	if models[kind] == nil {
		return errorResourceModelNotImplemented.withStack()
	}
	val := reflect.ValueOf(models[kind])
	if val.Kind() == reflect.Ptr {
		val = reflect.Indirect(val)
	}
	r.Data = reflect.New(val.Type()).Interface()
	return nil
}

func (r *Resource) BindRequestData() error {
	var err error
	if r.Data == nil {
		err = r.NewData(r.Key.Kind)
		if err != nil {
			return err
		}
	}

	if r.Access.Request.ContentLength < 2 {
		return errorEmptyRequestBody.withStack()
	}

	bodyContent, err := ioutil.ReadAll(r.Access.Request.Body)
	if err != nil {
		return errorRequestBodyRead.withCause(err).withStack()
	}

	err = json.Unmarshal(bodyContent, &r.Data)
	if err != nil {
		return errorRequestUnmarshal.withCause(err).withStack()
	}
	return nil
}

func (r *Resource) CopyData(dst *Resource) error {
	var p datastore.PropertyList
	var err error

	p, err = r.Save()
	if err != nil {
		return err
	}

	err = dst.Load(p)
	return err
}

func (r *Resource) MarshalJSON() ([]byte, error) {
	type Alias Resource
	return json.Marshal(&struct {
		Path  string `json:"path"`
		Error error  `json:"error"`
		*Alias
	}{
		Path:  Path(r.Key),
		Error: r.error,
		Alias: (*Alias)(r),
	})
}

// CheckAncestors verifies the path for full validity. First it checks for chain paternity issues. Second it check if ancestors really exist in the datastore.
// This means that change to paternity rules or deleted ancestors will block the creation of a new child.
// TODO: Should use BatchQuery.
func (r *Resource) CheckAncestors() error {
	var err error
	var k = r.Key
	err = ValidateKey(r.Key)
	if err != nil {
		return err
	}
	// test existence
	for {
		if k.Parent != nil {
			k = k.Parent
			q := datastore.NewQuery(k.Kind).Filter("__key__ =", k).KeysOnly()
			var c int
			c, err = DatastoreClient.Count(r.Access.Request.Context(), q)
			if err != nil {
				return errorDatastoreCount.withCause(err).withStack()
			}
			if c == 0 {
				err = fmt.Errorf(Path(k))
				return errorDatastoreAncestorNotFound.withCause(err).withStack()
			}
		} else {
			return nil
		}
	}
}

// func (r *Resource) Cross(key **datastore.Key, path *string) (ok bool) {
// 	if *path != "" && *key == nil {
// 		*key = Key(*path)
// 		if *key != nil {
// 			return true
// 		}
// 		r.complexError("invalid_path", *path)
// 		return false
// 	}
//
// 	if *key != nil {
// 		*path = Path(*key)
// 		if *path != "" {
// 			return true
// 		}
// 		k := *key
// 		r.complexError("invalid_key", k.String())
// 		return false
// 	}
//
// 	r.complexError("invalid_path", *path)
// 	return false
// }

func (r *Resource) AssertAction(action string) (ok bool) {
	if len(r.ActionsStack) > 0 {
		return r.ActionsStack[len(r.ActionsStack)-1] == action
	}
	return false
}

func (r *Resource) EnterAction(action string) {
	r.ActionsHistory = append(r.ActionsHistory, action)
	if r.AssertAction("complexError") {
		return
	}

	ValidAction(action)
	if len(r.ActionsStack) > 0 {
		if r.ActionsStack[len(r.ActionsStack)-1] == action {
			panic(fmt.Sprint("repeated_action", action))
		}
	}
	r.ActionsStack = append(r.ActionsStack, action)
}

func (r *Resource) ExitAction(action string) {
	if r.AssertAction("complexError") && action != "complexError" {
		return
	}

	ValidAction(action)
	if len(r.ActionsStack) > 0 {
		if r.ActionsStack[len(r.ActionsStack)-1] != action {
			panic(fmt.Sprintln("exiting_wrong_action"))
		}
		r.ActionsStack = r.ActionsStack[:len(r.ActionsStack)-1]
	} else {
		panic(fmt.Sprintln("nothing_to_exit"))
	}
}

// func (r *Resource) PreviousAction(action string) (ok bool) {
// 	if len(r.ActionsStack) > 1 {
// 		return r.ActionsStack[len(r.ActionsStack)-2] == action
// 	}
// 	return
// }

// type resourceGob struct {
//	Object    Data
//	CreatedAt time.TimeElapsed
// }

// func (r *Resource) SetMem() {
//	memItem := &memcache.Item{
//		Key:        r.Key.Encode(),
//		Expiration: time.Duration(24*7) * time.Hour,
//		Object: &resourceGob{
//			Object:    r.Data,
//			CreatedAt: r.CreatedAt,
//		},
//	}
//
//	if r.PreviousAction("create") {
//		withCause := memcache.Gob.Add(r.Access.Request.Context(), memItem)
//		if withCause != nil {
//			r.Log("memcache_set", withCause)
//		}
//		return
//	}
//
//	withCause := memcache.Gob.Set(r.Access.Request.Context(), memItem)
//	if withCause != nil {
//		r.Log("memcache_set", withCause)
//	}
// }

// func (r *Resource) GetMem() (withCause error) {
//	data := new(resourceGob)
//	_, withCause = memcache.Gob.Get(r.Access.Request.Context(), r.Key.Encode(), data)
//	if withCause != nil {
//		r.Log("memcache_get", withCause)
//		return
//	}
//	r.Data = data.Object
//	r.CreatedAt = data.CreatedAt
//	return
// }

// func (r *Resource) DelMem() {
//	withCause := memcache.Delete(r.Access.Request.Context(), r.Key.Encode())
//	if withCause != nil {
//		r.Log("memcache_del", withCause)
//	}
// }

// type Doc []search.Field

// Load loads all of the provided fields into d.
// It does not first reset *d to an empty slice.
// func (d *Doc) Load(f []search.Field, _ *search.DocumentMetadata) error {
//	*d = append(*d, f...)
//	return nil
// }

// Save returns all of d's fields as a slice of Fields.
// func (d *Doc) Save() ([]search.Field, *search.DocumentMetadata, error) {
//	return *d, nil, nil
// }

// func (d *Doc) Add(o interface{}) {
//	fl, withCause := search.SaveStruct(o)
//	if withCause != nil {
//		panic(withCause)
//	}
//
//	withCause = (search.FieldLoadSaver)(d).Load(fl, nil)
//	if withCause != nil {
//		panic(withCause)
//	}
// }

// func (d *Doc) Append(o *Doc) {
//	*d = append(*d, *o...)
// }

// func (r *Resource) Index(name string) {
//	index, withCause := search.Open(name)
//	if withCause != nil {
//		r.Log("index_open", withCause)
//		return
//	}
//
//	_, withCause = index.Put(r.Access.Request.Context(), r.Key.Encode(), (search.FieldLoadSaver)(r.Doc))
//	if withCause != nil {
//		r.Log("index_put", withCause)
//	}
//
//	return
// }

// type ChainMeta struct {
// 	Owner string `datastore:"-" json:"-" search:"resource_owner"`
// 	Kind  string `datastore:"-" json:"-" search:"resource_kind"`
// }

// func (r *Resource) Document(deep int) {
//	if r.Doc == nil {
//		r.Doc = &Doc{}
//	}
//
//	if r.Data == nil {
//		r.Read()
//		if r.HasErrors() {
//			return
//		}
//	}
//
//	r.Data.Index(r)
//
//	if deep <= 1 {
//		r.Doc.Add(&ChainMeta{
//			Kind:  r.Key.Kind,
//			Owner: OwnerKey(r.Key).Encode(),
//		})
//	}
//
//	if deep > 0 {
//		if r.Key.Parent != nil {
//			r.Previous = append(r.Previous, r.Key.Parent)
//		}
//		for _, pk := range r.Previous {
//			pr := InitResource(r.Access, pk)
//			pr.Document(deep + 1)
//			r.Doc.Append(pr.Doc)
//		}
//	}
// }

// type FieldJson struct {
// 	Value string `json:"value"`
// }

// d Document
// l FieldList
//
// func (d *Doc) MarshalJSON() ([]byte, error) {
//	jd := map[string]interface{}{}
//	for _, f := range ([]search.Field)(*d) {
//		jd[f.Name] = f.Value
//	}
//
//	return json.Marshal(jd)
// }

// func (r *Resource) MarshalJSON() ([]byte, complexError) {
// 	type Alias Resource
// 	return json.Marshal(&struct {
// 		Path string `json:"path"`
// 		*Alias
// 	}{
// 		Path:  Path(r.Key),
// 		Alias: (*Alias)(r),
// 	})
// }
