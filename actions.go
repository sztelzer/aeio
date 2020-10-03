package aeio

import (
	"cloud.google.com/go/datastore"
	"google.golang.org/api/iterator"
	"time"
)

// Create is responsible for creating the resource in the datastore. Thus, it registers the new Key.
// After returning the new resource Key, it empties the sub-object and reload the full data from the datastore.
// This assures that any 'after load method' is processed.
// It also calls the object.AfterLoad() method.
func (r *Resource) Create(skipCheckAncestors bool) {
	// start := time.Now()
	// defer r.Timing(start)
	// defer runtime.GC()
	r.EnterAction("create")
	if r.HasErrors() {
		return
	}

	if err := ValidatePaternityChain(r.Key); err != nil {
		r.Error("invalid_path", err)
		return
	}

	if !skipCheckAncestors {
		r.CheckAncestryExistence()
		if r.HasErrors() {
			return
		}
	}

	// if it already have an object, don't bind
	if r.Data == nil {
		r.ObjectFromRequest()
		if r.HasErrors() {
			return
		}
	}

	r.Update()
	if r.HasErrors() {
		return
	}

	r.Data.BeforeLoad(r)
	if r.HasErrors() {
		return
	}

	r.Data.AfterLoad(r)
	r.ExitAction("create")
}

// Update saves, but can also create the new item..
func (r *Resource) Update() {
	var err error
	// start := time.Now()
	// defer r.Timing(start)
	// defer runtime.GC()
	r.EnterAction("update")

	if r.HasErrors() {
		return
	}

	if r.Data == nil {
		r.Error("nothing_to_save", err)
		return
	}

	r.Data.BeforeSave(r)
	if r.HasErrors() {
		return
	}

	r.Key, err = DatastoreClient.Put(r.Access.Request.Context(), r.Key, r)
	if err != nil {
		r.Error("putting_object", err)
		return
	}

	//r.SetMem()

	r.Data.AfterSave(r)

	r.ExitAction("update")
}

// HardSave is an action that just saves the object back. It doesn't process any before/after object methods.
func (r *Resource) HardSave() {
	var err error
	// start := time.Now()
	// defer r.Timing(start)
	// defer runtime.GC()
	r.EnterAction("hardsave")

	if r.HasErrors() {
		return
	}

	if r.Data == nil {
		r.Error("no_object_to_save", err)
		return
	}

	_, err = DatastoreClient.Put(r.Access.Request.Context(), r.Key, r)
	if err != nil {
		r.Error("putting_object", err)
		return
	}

	//r.SetMem()

	r.ExitAction("hardsave")
}

// Read is an action that reads a resource from datastore. It always replace the object present with a new one of the right kind.
func (r *Resource) Read() {
	var err error
	start := time.Now()
	defer r.Timing(start)
	// defer runtime.GC()

	r.EnterAction("read")

	if r.HasErrors() {
		return
	}

	if r.Key == nil {
		r.Error("invalid_path", nil)
		return
	}

	r.Data, err = NewObject(r.Key.Kind)
	if err != nil {
		r.Error("initializing_object", err)
		return
	}

	r.Data.BeforeLoad(r)

	//err = r.GetMem()
	//if err != nil {
	err = DatastoreClient.Get(r.Access.Request.Context(), r.Key, r)
	if err != nil {
		r.Error("loading_object: "+Path(r.Key), err)
		return
	}
	//	r.SetMem()
	//}

	r.Data.AfterLoad(r)
	if r.HasErrors() {
		return
	}
	r.ExitAction("read")
}

// Patch is an action for a special case: it always loads the original object and adjusts only the fields that come from the request.
func (r *Resource) Patch() {
	// var Error Error
	// start := time.Now()
	// defer r.Timing(start)
	// defer runtime.GC()
	r.EnterAction("patch")

	if r.HasErrors() {
		return
	}

	r.Read()

	if r.HasErrors() {
		return
	}

	r.ObjectFromRequest()
	if r.HasErrors() {
		r.Data = nil
		return
	}

	r.Update()

	r.ExitAction("patch")
}

func (r *Resource) ReadAll() {
	// start := time.Now()
	// defer r.Timing(start)
	// defer runtime.GC()
	// r.Action = "readall"
	r.EnterAction("readall")

	var q *datastore.Query
	if r.Key.Parent != nil {
		q = datastore.NewQuery(r.Key.Kind).Filter("Parent =", r.Key.Parent)
	} else {
		q = datastore.NewQuery(r.Key.Kind)
	}
	r.RunListQuery(q)

	r.ExitAction("readall")
}

func (r *Resource) ReadAny() {
	// start := time.Now()
	// defer r.Timing(start)
	// defer runtime.GC()
	r.EnterAction("readany")

	var q *datastore.Query
	if r.Key.Parent != nil {
		q = datastore.NewQuery(r.Key.Kind).Ancestor(r.Key.Parent)
	} else {
		//this handles getting everything, including roots.
		q = datastore.NewQuery(r.Key.Kind)
	}
	r.RunListQuery(q)

	r.ExitAction("readany")
}

func (r *Resource) RunListQuery(q *datastore.Query) {
	var err error

	//default to 20 results, maximum of 100
	//length := ParseInt(r.Access.Request.FormValue("l"))
	//if length <= 0 {
	//	length = ListSizeDefault
	//}
	//if length > ListSizeMax {
	//	length = ListSizeMax
	//}
	q = q.Limit(ListSizeDefault)

	// if there is a Next cursor use it
	var cursor datastore.Cursor
	next := r.Access.Request.FormValue("n")
	if next != "" {
		cursor, err = datastore.DecodeCursor(next)
		if err != nil {
			r.Error("invalid_cursor", err)
			return
		}
		q = q.Start(cursor)
	}

	orderAscend := r.Access.Request.FormValue("a")
	if orderAscend != "" {
		q = q.Order(orderAscend)
	}

	orderDescend := r.Access.Request.FormValue("z")
	if orderDescend != "" {
		q = q.Order("-" + orderDescend)
	}

	//init a counter
	i := 0

	// finally, run!
	t := DatastoreClient.Run(r.Access.Request.Context(), q)
	for {
		nr := new(Resource)
		nr.Access = r.Access
		// nr.Action = "read"

		// init the object. it's cheap, just a new(kind) under, and easier than a copy
		nr.Data, err = NewObject(r.Key.Kind)
		if err != nil {
			nr.Error("invalid_kind", err)
			continue
		}

		// r.Log("types", errors.New(fmt.Sprintf("%+v", ObjectsRegistry)))
		// r.Log("resource", errors.New(fmt.Sprintf("%+v", nr)))
		//
		// ooo, Error := NewObject(r.Key.Kind())
		//
		// r.Log("kind", errors.New(fmt.Sprintf("%+v", r.Key.Kind())))
		// r.Log("obj", errors.New(fmt.Sprintf("%+v", ooo)))

		nr.Data.BeforeLoad(nr)

		nr.Key, err = t.Next(nr)
		if err == iterator.Done {
			cc, cerr := t.Cursor()
			if cerr == nil {
				r.Next = cc.String()
			}
			break
		}
		i++

		//if this instance has an Error, save the Error and put it as well, but do not process it more.
		if err != nil {
			nr.Error("listed_object_key", err)
			r.Resources = append(r.Resources, nr)
			continue
		}

		// nr.Count = 1
		nr.Data.AfterLoad(nr)

		//if depth is nil, 0 or false, reset the object after processing.
		//depth := r.Access.Request.FormValue("d")
		//if ParseInt(depth) == 0 {
		//	nr.Data = nil
		//}

		r.Resources = append(r.Resources, nr)
	}

	r.Count = i
}

func (r *Resource) Delete() {
	var err error
	// defer runtime.GC()
	r.EnterAction("delete")

	r.Read()
	if r.Errors != nil {
		return
	}

	// r.Action = "delete"
	r.Data.BeforeDelete(r)
	if r.Errors != nil {
		return
	}

	err = DatastoreClient.Delete(r.Access.Request.Context(), r.Key)
	if err != nil {
		r.Error("delete", err)
	}

	//r.DelMem()

	r.ExitAction("delete")

}
