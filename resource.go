package aeio

import (
	"encoding/json"
	"google.golang.org/appengine/datastore"
	"io/ioutil"
	"net/http"
	"time"
	// "golang.org/x/net/context"
	"errors"
	"strconv"
	"strings"
)

type Resource struct {
	Key       *datastore.Key `datastore:"-" json:"-"`
	Errors    []*E           `datastore:"-" json:"errors"`
	Object    Object         `datastore:"-" json:"object,omitempty"`
	Count     int            `datastore:"-" json:"count,omitempty"`
	Next      string         `datastore:"-" json:"next,omitempty"`
	Resources []*Resource    `datastore:"-" json:"resources,omitempty"`
	Time      int64          `datastore:"-" json:"time,omitempty"`
	CreatedAt time.Time      `datastore:"-" json:"created_at,omitempty"`
	Access    *Access        `datastore:"-" json:"-"`
	Action    string         `datastore:"-" json:"-"`
}

type Object interface {
	BeforeSave(*Resource)
	AfterSave(*Resource)
	BeforeLoad(*Resource)
	AfterLoad(*Resource)
	BeforeDelete(*Resource)
}

// BaseResource builds also the access object and key.
func RootResource(writer *http.ResponseWriter, request *http.Request) (r *Resource) {
	r = new(Resource)
	r.Access = NewAccess(writer, request)
	r.Key = Key(r.Access, r.Access.Request.URL.Path)
	if r.Key == nil {
		r.E("invalid_path", nil)
	}
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

// SO, we use bindjson instead of Unmarshal, so we can choose the object type beforehand.
// But we can use raw message in the object so to identify the type and initialize the right object.
// This will take a lot off the code.
// BindJson is a method that fills up the object with the Request Body
// func (r *Resource) BindJson() (err error) {
// 	if r.Access.Request.ContentLength == 0 {
// 		return nil //don't try to decode anything.
// 	}
//
// 	content, err := ioutil.ReadAll(r.Access.Request.Body)
// 	if err != nil {
// 		return err
// 	}
//
// 	err = json.Unmarshal(content, &r.Object)
// 	if err != nil {
// 		return err
// 	}
//
// 	return
// }

func (r *Resource) BindRequestObject() (err error) {
	if r.Access.Request.ContentLength == 0 {
		return nil
	}

	bodyContent, err := ioutil.ReadAll(r.Access.Request.Body)
	if err != nil {
		return err
	}

	r.L("body_content", errors.New(string(bodyContent)))
	// return errors.New("halt!")

	if r.Object == nil {
		r.Object, err = NewObject(r.Key.Kind())
		if err != nil {
			return err
		}
	}

	err = json.Unmarshal(bodyContent, &r.Object)
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

// CheckAncestry goes to the datastore to verify the existence of all ancestry parts of the resource.
// It has a small cost as it does various queries, but uses counts.
// TODO:  Maybe could use BatchQuerie.

func (r *Resource) CheckAncestry() {
	var k = r.Key
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

// func (r *Resource) CheckAncestry() (err error) {
// 	// defer runtime.GC()
// 	var k = r.Key
// 	for {
// 		if k.Parent() != nil {
// 			k = k.Parent()
// 			q := datastore.NewQuery(k.Kind()).Filter("__key__ =", k)
// 			c, err := q.Count(*r.Access.Context)
// 			if err != nil {
// 				return err
// 			}
// 			if c == 0 {
// 				return errors.New("not found: " + Path(k))
// 			}
// 			continue
// 		}
// 		break
// 	}
// 	return nil
// }

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
