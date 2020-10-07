package aeio

import (
	"cloud.google.com/go/datastore"
	"google.golang.org/api/iterator"
	"net/http"
)

// Create is responsible for creating the resource in the datastore. Thus, it registers the new Key.
// After returning the new resource Key, it empties the sub-object and reload the full data from the datastore.
// This assures that any 'after load method' is processed.
// It also calls the object.AfterLoad() method.
func (r *Resource) Create() error {
	var err error
	r.EnterAction("create")
	defer r.ExitAction("create")

	err = r.CheckAncestors()
	if err != nil {
		return err
	}

	// if it already have an object, don't bind
	if r.Data == nil {
		err = r.BindData()
		if err != nil {
			return err
		}
	}

	err = r.Update()
	if err != nil {
		return err
	}

	err = r.Data.BeforeLoad(r)
	if err != nil {
		return err
	}

	err = r.Data.AfterLoad(r)
	return err
}

// Update saves, but can also create the new item..
func (r *Resource) Update() error {
	var err error
	r.EnterAction("update")
	defer r.ExitAction("update")

	err = r.Data.BeforeSave(r)
	if err != nil {
		return err
	}

	r.Key, err = DatastoreClient.Put(r.Access.Request.Context(), r.Key, r)
	if err != nil {
		return NewError("datastore", err, http.StatusInternalServerError)
	}

	err = r.Data.AfterSave(r)
	return err
}

// HardSave is an action that just saves the object back. It doesn't process any before/after object methods.
func (r *Resource) HardSave() error {
	var err error
	r.EnterAction("hardsave")
	defer r.ExitAction("hardsave")

	if r.Data == nil {
		return NewError("no_data_to_save", nil, http.StatusBadRequest)
	}

	_, err = DatastoreClient.Put(r.Access.Request.Context(), r.Key, r)
	if err != nil {
		return NewError("datastore", err, http.StatusInternalServerError)
	}
	return nil
}

// Read is an action that reads a resource from datastore. It always replace the object present with a new one of the right kind.
func (r *Resource) Read() error {
	var err error
	r.EnterAction("read")
	defer r.ExitAction("read")


	if r.Key == nil {
		return NewError("invalid_key", nil, http.StatusBadRequest)
	}

	r.Data, err = NewObject(r.Key.Kind)
	if err != nil {
		return err
	}

	err = r.Data.BeforeLoad(r)
	if err != nil {
		return err
	}

	err = DatastoreClient.Get(r.Access.Request.Context(), r.Key, r)
	if err != nil {
		return NewError("loading_data", err, http.StatusNotFound)
	}

	err = r.Data.AfterLoad(r)
	return err
}

// Patch is an action for a special case: it always loads the original object and adjusts only the fields that come from the request.
func (r *Resource) Patch() error {
	var err error
	r.EnterAction("patch")
	defer r.ExitAction("patch")

	err = r.Read()
	if err != nil {
		return err
	}

	err = r.BindData()
	if err != nil {
		r.Data = nil
		return err
	}

	err = r.Update()
	return err
}

func (r *Resource) ReadAll() error {
	var err error
	r.EnterAction("readall")
	defer r.ExitAction("readall")

	var q *datastore.Query
	if r.Key.Parent != nil {
		q = datastore.NewQuery(r.Key.Kind).Filter("Parent =", r.Key.Parent)
	} else {
		q = datastore.NewQuery(r.Key.Kind)
	}
	err = r.RunListQuery(q)
	return err
}

func (r *Resource) ReadAny() error {
	r.EnterAction("readany")
	defer r.ExitAction("readany")

	var q *datastore.Query
	if r.Key.Parent != nil {
		q = datastore.NewQuery(r.Key.Kind).Ancestor(r.Key.Parent)
	} else {
		// this handles getting everything, including roots.
		q = datastore.NewQuery(r.Key.Kind)
	}
	err := r.RunListQuery(q)
	return err
}

func (r *Resource) RunListQuery(q *datastore.Query) error {
	var err error

	var lengths []int
	var length int = 0
	lengths = append(lengths, ParseInt(r.Access.Request.FormValue("length")))
	lengths = append(lengths, ParseInt(r.Access.Request.FormValue("len")))
	lengths = append(lengths, ParseInt(r.Access.Request.FormValue("l")))
	for _, v := range lengths {
		if length > v {
			length = v
		}
	}

	if length <= 0 {
		length = ListSizeDefault
	}
	if length > ListSizeMax {
		length = ListSizeMax
	}
	q = q.Limit(length)

	// if there is a Next cursor use it
	var cursor datastore.Cursor
	next := r.Access.Request.FormValue("n")
	if next != "" {
		cursor, err = datastore.DecodeCursor(next)
		if err != nil {
			return NewError("invalid_cursor", err, http.StatusBadRequest)
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

	// init a counter
	i := 0

	// finally, run!
	t := DatastoreClient.Run(r.Access.Request.Context(), q)
	for {
		nr := new(Resource)
		nr.Access = r.Access

		nr.Data, err = NewObject(r.Key.Kind)
		if err != nil {
			nr.error = err
			continue
		}

		err = nr.Data.BeforeLoad(nr)
		if err != nil {
			nr.error = err
			r.Resources = append(r.Resources, nr)
			continue
		}

		nr.Key, err = t.Next(nr)
		if err == iterator.Done {
			cursor, err = t.Cursor()
			if err == nil {
				r.Next = cursor.String()
			}
			break
		}
		i++

		err = nr.Data.AfterLoad(nr)
		if err != nil {
			nr.error = err
		}

		// if depth is nil, 0 or false, reset the object after processing.
		// depth := r.Access.Request.FormValue("d")
		// if ParseInt(depth) == 0 {
		//	nr.Data = nil
		// }

		r.Resources = append(r.Resources, nr)
	}

	r.ResourcesCount = i
	return nil
}

func (r *Resource) Delete() error {
	var err error
	r.EnterAction("delete")
	defer r.ExitAction("delete")

	err = r.Read()
	if err != nil {
		return err
	}

	// r.AssertAction = "delete"
	err = r.Data.BeforeDelete(r)
	if err != nil {
		return err
	}

	err = DatastoreClient.Delete(r.Access.Request.Context(), r.Key)
	if err != nil {
		return NewError("delete", err, http.StatusInternalServerError)
	}

	return nil
}
