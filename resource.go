package aeio

import (
	"encoding/json"
	"google.golang.org/appengine/datastore"
	"io/ioutil"
	"net/http"
	"time"
)

type Resource struct {
	Key       *datastore.Key `datastore:"-" json:"-"`
	Errors    []*E           `datastore:"-" json:"errors,omitempty"`
	Object    Object				 `datastore:"-" json:"object,omitempty"`
	Count     int            `datastore:"-" json:"count,omitempty"`
	Next      string         `datastore:"-" json:"next,omitempty"`
	Resources []*Resource    `datastore:"-" json:"resources,omitempty"`
	Access    *Access        `datastore:"-" json:"-"`
	Action    string         `datastore:"-" json:"-"`
	Time      int64          `datastore:"-" json:"time,omitempty"`
	CreatedAt time.Time      `datastore:"-" json:"created_at,omitempty"`
}

type Object interface {
	BeforeSave(*Resource)
	AfterSave(*Resource)
	BeforeLoad(*Resource)
	AfterLoad(*Resource)
	BeforeDelete(*Resource)
}

// BaseResource builds also the access object and key.
func RootResource(writer *http.ResponseWriter, request *http.Request) (r *Resource, err error) {
	r = new(Resource)
	r.Access = NewAccess(writer, request)
	r.Key, err = Key(r.Access, r.Access.Request.URL.Path)
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
func NewResource(access *Access, parentKey *datastore.Key, kind string) (r *Resource, err error) {
	r = new(Resource)
	r.Access = access
	r.Key, err = Key(r.Access, Path(parentKey) + "/" + kind)
	if err != nil {
		return nil, err
	}
	r.Object, err = NewObject(kind)
	if err != nil {
		return nil, err
	}
	return
}

// NewList return a new resource with the list type added to the key.
func NewListResource(parentResource *Resource, listKind string) (r *Resource, err error){
	return NewResource(parentResource.Access, parentResource.Key, listKind)
}




//Save puts the object into datastore, inlining CreatedAt and Parent in the object.
func (r *Resource) Save() (ps []datastore.Property, err error) {
	ps, err = datastore.SaveStruct(r.Object)
	ps = append(ps, datastore.Property{Name: "CreatedAt", Value: NoZeroTime(r.CreatedAt)})
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


// BindJson is a method that fills up the object with the Request Body
func (r *Resource) BindJson() (err error) {
	if r.Access.Request.ContentLength == 0 {
		return nil //don't try to decode anything.
	}
	content, err := ioutil.ReadAll(r.Access.Request.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(content, &r.Object)
	if err != nil {
		return err
	}
	return
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
