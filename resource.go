package aeio

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/memcache"
	"google.golang.org/appengine/search"
)

type Resource struct {
	Key       *datastore.Key   `datastore:"-" json:"-"`
	Errors    []*E             `datastore:"-" json:"errors"`
	Object    Object           `datastore:"-" json:"object,omitempty"`
	Count     int              `datastore:"-" json:"count,omitempty"`
	Next      string           `datastore:"-" json:"next,omitempty"`
	Resources []*Resource      `datastore:"-" json:"resources,omitempty"`
	Time      int64            `datastore:"-" json:"time,omitempty"`
	CreatedAt time.Time        `datastore:"-" json:"created_at,omitempty"`
	Access    *Access          `datastore:"-" json:"-"`
	Actions   []string         `datastore:"-" json:"-"`
	Doc       *Doc             `datastore:"-" json:"docs"`
	Previous  []*datastore.Key `datastore:"-" json:"-"`
}

type Object interface {
	BeforeSave(*Resource)
	AfterSave(*Resource)
	BeforeLoad(*Resource)
	AfterLoad(*Resource)
	BeforeDelete(*Resource)
	AddDocument(*Resource)
}

// BaseResource builds also the access object and key.
func RootResource(writer *http.ResponseWriter, request *http.Request) (r *Resource) {
	r = new(Resource)
	r.Access = NewAccess(writer, request)

	key := Key(r.Access, r.Access.Request.URL.Path)
	if Key == nil {
		r.E("invalid_path", nil)
	}

	r.Key = key
	return
}

// InitResource uses a Complete Key to init a specific resource. The resource returned is ready to be actioned.
func InitResource(access *Access, key *datastore.Key) (r *Resource) {
	r = new(Resource)
	r.Access = access
	r.Key = key
	return
}

// NewResource is used to create empty children resources. Parent path may be the "" string (root). It returns a resource with an incompleteKey.
// It initializes an object of type kind.
// TODO: swap access and parentKey for parentResource
func NewResource(access *Access, parentKey *datastore.Key, kind string) (r *Resource) {
	var err error
	r = new(Resource)
	r.Access = access

	err = TestPaternity(parentKey.Kind(), kind)
	if err != nil {
		r.E("invalid_kind", err)
	}

	r.Key = Key(r.Access, Path(parentKey)+"/"+kind)
	if r.Key == nil {
		r.E("invalid_path", nil)
		return
	}
	r.Object, err = NewObject(kind)
	if err != nil {
		r.E("invalid_kind", nil)
		return
	}
	return
}

// NewList return a new resource with the list type added to the key.
func NewListResource(parentResource *Resource, listKind string) (r *Resource) {
	return NewResource(parentResource.Access, parentResource.Key, listKind)
}

//Save puts the object into datastore, inlining CreatedAt and Parent in the object.
func (r *Resource) Save() (ps []datastore.Property, err error) {
	r.CreatedAt = NoZeroTime(r.CreatedAt)
	ps, err = datastore.SaveStruct(r.Object)
	ps = append(ps, datastore.Property{Name: "CreatedAt", Value: r.CreatedAt})
	ps = append(ps, datastore.Property{Name: "Parent", Value: r.Key.Parent()})
	return
}

//Load extracts the datastore data in an object, taking CreatedAt and Parent off the object.
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
	err = datastore.LoadStruct(r.Object, ps2)
	return
}

//Key transforms a path in a datastore key using an access.
func Key(access *Access, path string) (k *datastore.Key) {
	var kd string
	var id int64
	var p []string

	if path == "" {
		return nil
	}

	if ValidPath.MatchString(path) != true {
		return nil
	}

	p = strings.SplitN(path, "/", 2)
	p = strings.Split(p[1], "/")
	for i := 0; i < len(p); i = i + 2 {
		kd = p[i]
		if i < len(p)-1 {
			id, _ = strconv.ParseInt(p[i+1], 10, 64)
			k = datastore.NewKey(*access.Context, kd, "", id, k)
		} else {
			k = datastore.NewIncompleteKey(*access.Context, kd, k)
		}
	}
	return k
}

//Path transforms a datastore Key into a Path
func Path(k *datastore.Key) (p string) {
	if k.Incomplete() == false {
		p = "/" + strconv.FormatInt(k.IntID(), 10)
	}
	p = "/" + k.Kind() + p
	for {
		k = k.Parent()
		if k == nil {
			break
		}
		p = "/" + k.Kind() + "/" + strconv.FormatInt(k.IntID(), 10) + p
	}
	return p
}

func (r *Resource) BindRequestObject() {
	var err error
	if r.Object == nil {
		r.Object, err = NewObject(r.Key.Kind())
		if err != nil {
			r.E("initializing_object", err)
			return
		}
	}

	if r.Access.Request.ContentLength < 2 {
		// r.E("body_is_empty", nil)
		return
	}

	bodyContent, err := ioutil.ReadAll(r.Access.Request.Body)
	if err != nil {
		r.E("body_is_weird", nil)
		return
	}

	err = json.Unmarshal(bodyContent, &r.Object)
	if err != nil {
		r.E("json_binding", err)
		return
	}
}

func (r *Resource) MarshalJSON() ([]byte, error) {
	type Alias Resource
	return json.Marshal(&struct {
		Path string `json:"path"`
		*Alias
	}{
		Path:  Path(r.Key),
		Alias: (*Alias)(r),
	})
}

// CheckAncestry verifies the path for full validity. First it checks for chain paternity issues. Second it check if ancestors really exist in the datastore.
// This means that change to paternity rules or deleted ancestors will block the creation of a new child.
// TODO: Maybe could use BatchQuerie.
func (r *Resource) CheckAncestry() {
	var k = r.Key
	//test paternity
	err := TestKeyChainPaternity(k)
	if err != nil {
		r.E("broken_ancestor_chain", err)
		return
	}
	//test existence
	for {
		if k.Parent() != nil {
			k = k.Parent()
			q := datastore.NewQuery(k.Kind()).Filter("__key__ =", k)
			c, err := q.Count(*r.Access.Context)
			if err != nil {
				r.E("ancestor_not_found", err)
				break
			}
			if c == 0 {
				r.E("ancestor_not_found", Path(k))
				break
			}
			continue
		}
		break
	}
}

func (r *Resource) Cross(key **datastore.Key, path *string) (ok bool) {
	if *path != "" && *key == nil {
		*key = Key(r.Access, *path)
		if *key != nil {
			return true
		}
		r.E("invalid_path", *path)
		return false
	}

	if *key != nil {
		*path = Path(*key)
		if *path != "" {
			return true
		}
		k := *key
		r.E("invalid_key", k.String())
		return false
	}

	r.E("invalid_path", *path)
	return false
}

type rgob struct {
	Object    Object
	CreatedAt time.Time
}

func (r *Resource) SetMem() {
	memitem := &memcache.Item{
		Key:        r.Key.Encode(),
		Expiration: time.Duration(24*7) * time.Hour,
		Object: &rgob{
			Object:    r.Object,
			CreatedAt: r.CreatedAt,
		},
	}

	if r.PreviousAction("create") {
		err := memcache.Gob.Add(*r.Access.Context, memitem)
		if err != nil {
			r.L("memcache_set", err)
		}
		return
	}

	err := memcache.Gob.Set(*r.Access.Context, memitem)
	if err != nil {
		r.L("memcache_set", err)
	}
}

func (r *Resource) GetMem() (err error) {
	data := new(rgob)
	_, err = memcache.Gob.Get(*r.Access.Context, r.Key.Encode(), data)
	if err != nil {
		r.L("memcache_get", err)
		return
	}
	r.Object = data.Object
	r.CreatedAt = data.CreatedAt
	return
}

func (r *Resource) DelMem() {
	err := memcache.Delete(*r.Access.Context, r.Key.Encode())
	if err != nil {
		r.L("memcache_del", err)
	}
}

func (r *Resource) EnterAction(action string) {
	if r.Action("error") {
		return
	}

	ValidAction(action)
	if len(r.Actions) > 0 {
		if r.Actions[len(r.Actions)-1] == action {
			panic(fmt.Sprintln("repeated_action"))
		}
	}
	r.Actions = append(r.Actions, action)
}

func (r *Resource) ExitAction(action string) {
	if r.Action("error") && action != "error" {
		return
	}

	ValidAction(action)
	if len(r.Actions) > 0 {
		if r.Actions[len(r.Actions)-1] != action {
			panic(fmt.Sprintln("exiting_wrong_action"))
		}
		r.Actions = r.Actions[:len(r.Actions)-1]
	} else {
		panic(fmt.Sprintln("nothing_to_exit"))
	}
}

func (r *Resource) ErrorAction() {
	r.Actions = nil
	r.EnterAction("error")
}

func (r *Resource) Action(action string) (ok bool) {
	if len(r.Actions) > 0 {
		return r.Actions[0] == action
	}
	return
}

func (r *Resource) PreviousAction(action string) (ok bool) {
	if len(r.Actions) > 1 {
		return r.Actions[len(r.Actions)-2] == action
	}
	return
}

type Doc []search.Field

// Load loads all of the provided fields into d.
// It does not first reset *d to an empty slice.
func (d *Doc) Load(f []search.Field, _ *search.DocumentMetadata) error {
	*d = append(*d, f...)
	return nil
}

// Save returns all of d's fields as a slice of Fields.
func (d *Doc) Save() ([]search.Field, *search.DocumentMetadata, error) {
	return *d, nil, nil
}

func (d *Doc) Add(o interface{}) {
	fl, err := search.SaveStruct(o)
	if err != nil {
		panic(err)
	}

	_ = (search.FieldLoadSaver)(d).Load(fl, nil)
	if err != nil {
		panic(err)
	}
}

func (d *Doc) Append(o *Doc) {
	*d = append(*d, *o...)
}

func (r *Resource) Index(name string) {
	index, err := search.Open(name)
	if err != nil {
		r.L("index_open", err)
		return
	}

	_, err = index.Put(*r.Access.Context, r.Key.Encode(), (search.FieldLoadSaver)(r.Doc))
	if err != nil {
		r.L("index_put", err)
	}

	return
}

type ChainMeta struct {
	Owner string `datastore:"-" json:"-" search:"resource_owner"`
	Kind  string `datastore:"-" json:"-" search:"resource_kind"`
}

func (r *Resource) Document(deep int) {
	if r.Doc == nil {
		r.Doc = &Doc{}
	}

	if r.Object == nil {
		r.Read()
		if r.HasErrors() {
			return
		}
	}

	r.Object.AddDocument(r)

	if deep <= 1 {
		r.Doc.Add(&ChainMeta{
			Kind:  r.Key.Kind(),
			Owner: OwnerKey(r.Key).Encode(),
		})
	}

	if deep > 0 {
		if r.Key.Parent() != nil {
			r.Previous = append(r.Previous, r.Key.Parent())
		}
		for _, pk := range r.Previous {
			pr := InitResource(r.Access, pk)
			pr.Document(deep + 1)
			r.Doc.Append(pr.Doc)
		}
	}
}

// type FieldJson struct {
// 	Value string `json:"value"`
// }

//d Document
//l FieldList
//
func (d *Doc) MarshalJSON() ([]byte, error) {
	jd := map[string]interface{}{}
	for _, f := range ([]search.Field)(*d) {
		jd[f.Name] = f.Value
	}

	return json.Marshal(jd)
}

// func (r *Resource) MarshalJSON() ([]byte, error) {
// 	type Alias Resource
// 	return json.Marshal(&struct {
// 		Path string `json:"path"`
// 		*Alias
// 	}{
// 		Path:  Path(r.Key),
// 		Alias: (*Alias)(r),
// 	})
// }
