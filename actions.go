package aeio

import (
	"aeio/helpers/conversions"
	"google.golang.org/appengine/datastore"
	// "time"
)

// Create is responsible for creating the resource in the datastore. Thus, it registers the new Key.
// After returning the new resource Key, it empties the sub-object and reload the full data from the datastore.
// This assures that any 'after load method' is processed.
// It also calls the object.AfterLoad() method.
func (r *Resource) Create(checkAncestors bool) {
	var err error
	// start := time.Now()
	// defer r.Timing(start)
	r.Action = "create"
	// defer runtime.GC()
	//first: check parents Paths
	//get parent() recursively.
	if checkAncestors == true {
		err = r.CheckAncestry()
		if err != nil {
			r.E("ancestor_not_found", err)
		}
	}

	if r.Object == nil {
		r.Object, err = NewObjectKind(r.Key.Kind())
		if err != nil {
			r.E("initializing_object", err)
			return
		}

		err := r.BindJson()
		if err != nil {
			r.E("json_binding", err)
			return
		}
	}
	r.Object.BeforeSave(r)

	if len(r.Errors) == 0 {
		r.Key, err = datastore.Put(*r.Access.Context, r.Key, r)
		if err != nil {
			r.E("putting_object", err)
			return
		}

		r.Paths()
		r.Object.AfterSave(r)
		if len(r.Errors) != 0 {
			return
		}

		// lets assume that if we can skip parent, then this is child and will be reloaded by the parent on his afterload
		if checkAncestors == true {
			//clear
			r.Object, err = NewObjectKind(r.Key.Kind()) //reset object, discard what came on request
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
			r.Paths()
			r.Object.AfterLoad(r)
		}
	}
}

func (r *Resource) Read() {
	// defer runtime.GC()
	var err error
	// start := time.Now()
	// defer r.Timing(start)
	r.Action = "read"

	r.Object, err = NewObjectKind(r.Key.Kind())
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
	r.Paths()
	r.Object.AfterLoad(r)
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

// func (r *Resource) ReadAny() {
// 	start := time.Now()
// 	defer r.Timing(start)
// 	// defer runtime.GC()
// 	r.Action = "readany"
//
// 	var q *datastore.Query
// 	if r.Key.Parent() != nil {
// 		q = datastore.NewQuery(r.Key.Kind()).Ancestor(r.Key.Parent())
// 	} else {
// 		q = datastore.NewQuery(r.Key.Kind())
// 	}
// 	r.RunListQuery(q)
// }

func (r *Resource) RunListQuery(q *datastore.Query) {
	var err error

	//default to 20 results, maximum of 100
	length := conversions.ParseInt(r.Access.Request.FormValue("l"))
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

		// init the object. it's cheap, just a new(kind) under, and easyer than a copy
		nr.Object, err = NewObjectKind(r.Key.Kind())
		if err != nil {
			nr.E("invalid_kind", err)
			continue
		}

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
		nr.Paths()
		nr.Object.AfterLoad(nr)

		//if depth is nil, 0 or false, reset the object after processing.
		depth := r.Access.Request.FormValue("d")
		if conversions.ParseInt(depth) == 0 {
			nr.Object = nil
		}

		r.Resources = append(r.Resources, nr)
	}

	r.Count = i
	r.Paths()
}
