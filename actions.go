package aeio

import (
	"aeio/helpers/convert"
	"google.golang.org/appengine/datastore"
)

// Create is responsible for creating the resource in the datastore. Thus, it registers the new Key.
// After returning the new resource Key, it empties the sub-object and reload the full data from the datastore.
// This assures that any 'after load method' is processed.
// It also calls the object.AfterLoad() method.
func (r *Resource) Create(dontCheckAncestors bool) {
	var err error
	// start := time.Now()
	// defer r.Timing(start)
	r.Action = "create"

	// defer runtime.GC()

	//first: check parents Paths
	//get parent() recursively.
	if dontCheckAncestors == false {
		r.CheckAncestry()
		if r.HasErrors() {
			return
		}
	}

	err = r.BindRequestObject()
	if err != nil {
		r.E("json_binding", err)
		return
	}

	r.L("stockResourceBinded", r.Object)

	// }
	r.Object.BeforeSave(r)

	if len(r.Errors) == 0 {

		r.Key, err = datastore.Put(*r.Access.Context, r.Key, r)
		if err != nil {
			r.E("putting_object", err)
			return
		}

		r.Object.AfterSave(r)
		if len(r.Errors) != 0 {
			return
		}

		// lets assume that if we can skip parent, then this is child and will be reloaded by the parent on his afterload
		if dontCheckAncestors == false {
			//clear
			r.Object, err = NewObject(r.Key.Kind()) //reset object, discard what came on request
			if err != nil {
				r.E("initializing_object", err)
				return
			}
			r.Object.BeforeLoad(r)
			err = datastore.Get(*r.Access.Context, r.Key, r)
			if err != nil {
				r.E("reloading_object", err)
				return
			}
			r.Count = 1
			r.Object.AfterLoad(r)
		}
	}
}

// this is for use inside the code, it just saves the object back. Must have an object loaded.
// hard doesn't process any before/after object methods
func (r *Resource) HardSave() {
	// defer runtime.GC()
	var err error
	if r.Object != nil {
		r.Key, err = datastore.Put(*r.Access.Context, r.Key, r)
		if err != nil {
			r.E("putting_object", err)
			return
		}
		return
	}
	r.E("no_object_to_save", err)
	return
}

func (r *Resource) Read() {
	// defer runtime.GC()
	var err error
	// start := time.Now()
	// defer r.Timing(start)
	r.Action = "read"

	r.Object, err = NewObject(r.Key.Kind())
	if err != nil {
		r.E("initializing_object", err)
		return
	}

	r.Object.BeforeLoad(r)
	err = datastore.Get(*r.Access.Context, r.Key, r)
	if err != nil {
		r.E("loading_object: "+Path(r.Key), err)
		return
	}

	r.Count = 1
	r.Object.AfterLoad(r)
}

func (r *Resource) Patch() {
	var err error
	// start := time.Now()
	// defer r.Timing(start)
	// defer runtime.GC()

	r.Object, err = NewObject(r.Key.Kind())
	if err != nil {
		r.E("initializing_object", err)
		return
	}

	r.Object.BeforeLoad(r)
	err = datastore.Get(*r.Access.Context, r.Key, r)
	if err != nil {
		r.E("preloading_object", err)
		return
	}
	r.Object.AfterLoad(r)
	if len(r.Errors) > 0 {
		r.E("after_preloading_object", err)
		r.Object = nil
		return
	}

	err = r.BindRequestObject()
	if err != nil {
		r.E("json_binding", err)
		r.Object = nil
		return
	}
	r.Action = "patch"
	r.Object.BeforeSave(r)

	if len(r.Errors) == 0 {
		r.Key, err = datastore.Put(*r.Access.Context, r.Key, r) //Save should run BeforeSave() on object
		if err != nil {
			r.E("putting_object", err)
			return
		}
		r.Object.AfterSave(r)

		r.Object, err = NewObject(r.Key.Kind()) //reset object, discard what came on request
		if err != nil {
			r.E("initializing_object", err)
			return
		}

		r.Object.BeforeLoad(r)
		err = datastore.Get(*r.Access.Context, r.Key, r)
		if err != nil {
			r.E("reloading_object", err)
			return
		}

		r.Count = 1
		r.Object.AfterLoad(r)

	} else {
		r.Object = nil
		return
	}
}

func (r *Resource) ReadAll() {
	// start := time.Now()
	// defer r.Timing(start)
	// defer runtime.GC()
	r.Action = "readall"

	var q *datastore.Query
	if r.Key.Parent() != nil {
		q = datastore.NewQuery(r.Key.Kind()).Filter("Parent =", r.Key.Parent())
	} else {
		q = datastore.NewQuery(r.Key.Kind())
	}
	r.RunListQuery(q)
}

func (r *Resource) ReadAny() {
	// start := time.Now()
	// defer r.Timing(start)
	// defer runtime.GC()
	r.Action = "readany"

	var q *datastore.Query
	if r.Key.Parent() != nil {
		q = datastore.NewQuery(r.Key.Kind()).Ancestor(r.Key.Parent())
	} else {
		//this handles getting everything, including roots.
		q = datastore.NewQuery(r.Key.Kind())
	}
	r.RunListQuery(q)
}

func (r *Resource) RunListQuery(q *datastore.Query) {
	var err error

	//default to 20 results, maximum of 100
	length := convert.ParseInt(r.Access.Request.FormValue("l"))
	if length <= 0 {
		length = DefaultListSize
	}
	if length > MaxListSize {
		length = MaxListSize
	}
	q = q.Limit(length)

	// if there is a Next cursor use it
	var cursor datastore.Cursor
	next := r.Access.Request.FormValue("n")
	if next != "" {
		cursor, err = datastore.DecodeCursor(next)
		if err != nil {
			r.E("invalid_cursor", err)
			return
		}
		q = q.Start(cursor)
	}

	ascend := r.Access.Request.FormValue("a")
	if ascend != "" {
		q = q.Order(ascend)
	}

	descend := r.Access.Request.FormValue("z")
	if descend != "" {
		q = q.Order("-" + descend)
	}

	//init a counter
	i := 0

	// finally, run!
	t := q.Run(*r.Access.Context)
	for {
		nr := new(Resource)
		nr.Access = r.Access
		nr.Action = "read"

		// init the object. it's cheap, just a new(kind) under, and easier than a copy
		nr.Object, err = NewObject(r.Key.Kind())
		if err != nil {
			nr.E("invalid_kind", err)
			continue
		}

		// r.L("types", errors.New(fmt.Sprintf("%+v", ObjectsRegistry)))
		// r.L("resource", errors.New(fmt.Sprintf("%+v", nr)))
		//
		// ooo, err := NewObject(r.Key.Kind())
		//
		// r.L("kind", errors.New(fmt.Sprintf("%+v", r.Key.Kind())))
		// r.L("obj", errors.New(fmt.Sprintf("%+v", ooo)))

		nr.Object.BeforeLoad(nr)

		nr.Key, err = t.Next(nr)
		if err == datastore.Done {
			cc, cerr := t.Cursor()
			if cerr == nil {
				r.Next = cc.String()
			}
			break
		}
		i++

		//if this instance has an error, save the error and put it as well, but do not process it more.
		if err != nil {
			nr.E("listed_object_key", err)
			r.Resources = append(r.Resources, nr)
			continue
		}

		// nr.Count = 1
		nr.Object.AfterLoad(nr)

		//if depth is nil, 0 or false, reset the object after processing.
		depth := r.Access.Request.FormValue("d")
		if convert.ParseInt(depth) == 0 {
			nr.Object = nil
		}

		r.Resources = append(r.Resources, nr)
	}

	r.Count = i
}

func (r *Resource) Delete() {
	var err error
	// defer runtime.GC()
	r.Read()
	if r.Errors != nil {
		return
	}

	r.Action = "delete"
	r.Object.BeforeDelete(r)
	if r.Errors != nil {
		return
	}

	err = datastore.Delete(*r.Access.Context, r.Key)
	if err != nil {
		r.E("delete", err)
	}
}
