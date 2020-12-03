package aeio

import (
	"cloud.google.com/go/datastore"
	"encoding/json"
	"fmt"
	patchstruct "github.com/sztelzer/structpatch"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
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
	CreatedAt      time.Time      `datastore:"-" json:"createdAt,omitempty"`
	Access         *Access        `datastore:"-" json:"-"`
	ActionsStack   []string       `datastore:"-" json:"-"`
	ActionsHistory []string       `datastore:"-" json:"-"`
	Resources      []*Resource    `datastore:"-" json:"resources,omitempty"`
	ResourcesCount *int            `datastore:"-" json:"resourcesCount,omitempty"`
	Next           string         `datastore:"-" json:"-"`
	TimeElapsed    int64          `datastore:"-" json:"timeElapsed,omitempty"`
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

type DataBind interface {
	Bind(*Resource, interface{}) error
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

// Load extracts the datastore data into an object, taking CreatedAt and Parent off the object.
func (r *Resource) Load(ps []datastore.Property) (err error) {
	var ps2 []datastore.Property
	for _, p := range ps {
		switch p.Name {
		case "CreatedAt":
			r.CreatedAt = NoZeroTime(p.Value.(time.Time))
		case "Parent":
		default:
			ps2 = append(ps2, p)
		}
	}
	err = datastore.LoadStruct(r.Data, ps2)
	return
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


// BindRequestData takes the request data to the object data it will always respect the json tag of fields,
// and on specifically action UPDATE will bind only allowed fields. This is useful for locking fields on
// the original state.
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

	if !r.AssertAction(ActionUpdate) {
		// load directly into r.Data
		err = json.Unmarshal(bodyContent, &r.Data)
		if err != nil {
			return errorRequestUnmarshal.withCause(err).withStack()
		}
		return nil
	}

	// ok, it's UPDATE
	// Let's load into a temporary Data, and only copy
	// non empty, non locked fields

	// create patcher temporary object
	var patcher interface{}
	patcher, err = NewPatcher(r.Key.Kind)
	if err != nil {
		patcher, err = NewObject(r.Key.Kind)
		if err != nil {
			return errorRequestUnmarshal.withCause(err).withStack().withLog()
		}
	}

	// load data from request into temporary requestData
	err = json.Unmarshal(bodyContent, &patcher)
	if err != nil {
		return errorRequestUnmarshal.withCause(err).withStack().withLog()
	}

	err = patchstruct.Patch(patcher, r.Data, "ignored")
	if err != nil {
		return errorRequestUnmarshal.withCause(err).withStack().withLog()
	}

	return nil
}

// patchFields copy recursively
// func patchFields(src interface{}, dst interface{}) interface{} {
// 	srcStructValue := reflect.ValueOf(src)
// 	dstStructValue := reflect.ValueOf(dst)
// 	dstStructType := reflect.TypeOf(dst)
//
// 	for i := 0; i < srcStructValue.NumField(); i++ {
// 		srcField := srcStructValue.Field(i)
// 		dstField := dstStructValue.Field(i)
// 		if _, locked := dstStructType.Field(i).Tag.Lookup("lock"); locked {
// 			continue
// 		}
//
// 		if !dstField.IsNil() && srcField.IsValid() && !srcField.IsZero() {
//
//
// 			dstField.Set(srcField)
// 		}
// 	}
//
// 	return dst
// }



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
		Path      string    `json:"key"`
		Error     error     `json:"error"`
		CreatedAt time.Time `json:"createdAt"`
		*Alias
	}{
		Path:      Path(r.Key),
		Error:     r.error,
		CreatedAt: NoZeroTime(r.CreatedAt),
		Alias:     (*Alias)(r),
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
	log.Println("entering action:", action)
	r.ActionsHistory = append(r.ActionsHistory, action)
	if r.AssertAction("complexError") {
		return
	}

	ValidAction(action)
	// if len(r.ActionsStack) > 0 {
	// 	if r.ActionsStack[len(r.ActionsStack)-1] == action {
	// 		panic(fmt.Sprint("repeated_action", action))
	// 	}
	// }
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
//	_, withCause = index.Create(r.Access.Request.Context(), r.Key.Encode(), (search.FieldLoadSaver)(r.Doc))
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
//		r.Get()
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

// Respond writes to the resource writer with the selected status and headers. After calling Respond, the connection
// will be closed, and so it's context. Any process on the request stack may continue after it, but any process using
// the request.Context will be finished.
//
// If the passed status is not http valid, it will be responded http.StatusInternalServerError (500) and the resource will be
// receive the error reference "invalid_status".
//
// If the resource errors contains the reference "not_authorized", the status will be http.StatusForbidden (403) independently
// of the status passed to Respond.
func (r *Resource) Respond(err error) {
	var status = http.StatusOK

	if err != nil {
		switch err.(type) {
		case complexError:
			break
		case error:
			err = errorUnknown.withCause(err)
			break
		}

		r.error = err
		status = err.(complexError).Code
		if http.StatusText(status) == "" {
			r.error = errorInvalidHttpStatusCode.withCause(err).withStack()
		}
	}

	if r.Access.Writer.Header().Get("Content-Type") == "" {
		r.Access.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	}

	log.Println(status, r.Access.Request.Method, r.Access.Request.URL.Path, err)

	j, err := json.Marshal(r)
	if err != nil {
		_ = errorResponseMarshal.withCause(err).withStack().withLog()
	}

	_, err = r.Access.Writer.Write(j)
	if err != nil {
		_ = errorResponseWrite.withCause(err).withStack().withLog()
	}
}

// Timing is used to time the processing of resources.
func (r *Resource) Timing(start time.Time) {
	r.TimeElapsed = int64(time.Since(start) / time.Millisecond)
}
