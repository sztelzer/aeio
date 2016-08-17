package aeio

import (
	"encoding/json"
	"errors"
	"google.golang.org/appengine/datastore"
	"io/ioutil"
	"net/http"
	"runtime"
	"time"
	// "bytes"
	// "encoding/binary"
)

// type Resourcer interface {
// 	Create(*Access)
// 	Patch(*Access)
// 	Read(*Access)
// 	ReadChildren(*Access)
// 	ReadAny(*Access)
// 	E(error)
// }

type Resource struct {
	Path      string `datastore:"-" json:"path"`
	Errors    []*E   `datastore:"-" json:"errors,omitempty"`
	Object    `datastore:"-" json:"object,omitempty"`
	Count     int            `datastore:"-" json:"count,omitempty"`
	Next      string         `datastore:"-" json:"next,omitempty"`
	Resources []*Resource    `datastore:"-" json:"resources,omitempty"`
	Access    *Access        `datastore:"-" json:"-"`
	Key       *datastore.Key `datastore:"-" json:"-"`
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

type Handler func(*Resource)

// BaseResource builds also the access object and key.
func RootResource(writer *http.ResponseWriter, request *http.Request) (r *Resource, err error) {
	r = new(Resource)
	r.Access = NewAccess(writer, request)
	r.Key, err = Key(r.Access, r.Access.Request.URL.Path)
	if err == nil {
		r.Path = Path(r.Key)
	}
	return
}

// InitResource uses a Complete Key to init a specific resource. The resource returned is ready to be actioned.
func InitResource(access *Access, key *datastore.Key) (r *Resource) {
	r = new(Resource)
	r.Access = access
	r.Key = key
	r.Path = Path(r.Key)
	return
}

// NewResource is used to create empty children resources. Parent path may be the "" string (root). It returns a resource with an incompleteKey.
// It initializes an object of type kind.
func NewResource(access *Access, parentKey *datastore.Key, kind string) (r *Resource, err error) {
	r = new(Resource)
	r.Access = access
	r.Path = Path(parentKey) + "/" + kind
	r.Key, err = Key(r.Access, r.Path)
	if err != nil {
		return nil, err
	}
	r.Object, err = NewObject(kind)
	if err != nil {
		return nil, err
	}
	return
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

// Path is a simple method that takes the path from the resource key.
func (r *Resource) Paths() {
	r.Path = Path(r.Key)
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

// CheckAncestry goes to the datastore to verify the existence of all ancestry parts of the resource.
// It has a small cost as it does various queries, but uses counts.
// TODO:  Maybe could use BatchQuerie.
func (r *Resource) CheckAncestry() (err error) {
	defer runtime.GC()
	var k = r.Key
	for {
		if k.Parent() != nil {
			k = k.Parent()
			q := datastore.NewQuery(k.Kind()).Filter("__key__ =", k)
			c, err := q.Count(*r.Access.Context)
			if err != nil {
				return err
			}
			if c == 0 {
				return errors.New("not found: " + Path(k))
			}
			continue
		}
		break
	}
	return nil
}

// Timing is used to time the processing of resources.
func (r *Resource) Timing(s time.Time) {
	r.Time = int64(time.Since(s) / time.Millisecond)
}
