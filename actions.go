package aeio

import (
	"errors"
	"fmt"
	"log"
	
	"cloud.google.com/go/datastore"
	"google.golang.org/api/iterator"
)

// Create creates a new resource
func (r *Resource) Create() error {
	var err error
	r.EnterAction(ActionCreate)
	defer r.ExitAction(ActionCreate)

	if err = ValidateKey(r.Key); err != nil {
		return errorInvalidPath.withCause(err).withStack(10).withLog()
	}

	if !r.Key.Incomplete() {
		return errorInvalidPath.withCause(errors.New("path key must be incomplete for creation")).withStack(10).withLog()
	}

	if r.Data == nil {
		err = r.BindRequestData()
		if err != nil {
			return errorUnknown.withCause(err).withStack(10).withLog()
		}
	}

	if data, ok := r.Data.(DataBeforeSave); ok {
		err := data.BeforeSave(r)
		if err != nil {
			return errorUnknown.withCause(err).withStack(10).withLog()
		}
	}

	r.Key, err = DatastoreClient.Put(r.Access.Request.Context(), r.Key, r)
	if err != nil {
		return errorDatastorePut.withCause(err).withStack(10).withLog()
	}

	if data, ok := r.Data.(DataAfterSave); ok {
		err := data.AfterSave(r)
		if err != nil {
			return errorUnknown.withCause(err).withStack(10).withLog()
		}
	}

	return nil
}

func (r *Resource) Update() error {
	var err error
	r.EnterAction(ActionUpdate)
	defer r.ExitAction(ActionUpdate)

	err = ValidateKey(r.Key)
	if err != nil {
		return errorInvalidPath.withCause(err).withStack(10).withLog()
	}

	if r.Key.Incomplete() {
		return errorInvalidPath.withCause(errors.New("path key must be complete for update")).withStack(10).withLog()
	}

	if r.Data == nil {
		err = r.Get()
		if err != nil {
			return err
		}

		err = r.BindRequestData()
		if err != nil {
			return errorUnknown.withCause(err).withStack(10).withLog()
		}
	}

	if data, ok := r.Data.(DataBeforeSave); ok {
		err := data.BeforeSave(r)
		if err != nil {
			return errorUnknown.withCause(err).withStack(10).withLog()
		}
	}

	r.Key, err = DatastoreClient.Put(r.Access.Request.Context(), r.Key, r)
	if err != nil {
		return errorDatastorePut.withCause(err).withStack(10).withLog()
	}

	if data, ok := r.Data.(DataAfterSave); ok {
		err := data.AfterSave(r)
		if err != nil {
			return errorUnknown.withCause(err).withStack(10).withLog()
		}
	}

	return nil
}


// Get is an action that reads a resource from datastore. It always replace the object present with a new one of the right kind.
// The resource only need to have a complete key.
func (r *Resource) Get() error {
	var err error
	r.EnterAction(ActionRead)
	defer r.ExitAction(ActionRead)

	if r.Key.Incomplete() {
		return errorInvalidPath.withHint(fmt.Sprintf("%s", "The key passed to read is incomplete")).withStack(10).withLog()
	}

	err = ValidateKey(r.Key)
	if err != nil {
		return errorUnknown.withCause(err).withStack(10).withLog()
	}

	r.Data, err = NewObject(r.Key.Kind)
	if err != nil {
		return errorUnknown.withCause(err).withStack(10).withLog()
	}

	if data, ok := r.Data.(DataBeforeLoad); ok {
		err = data.BeforeLoad(r)
		if err != nil {
			return errorUnknown.withCause(err).withStack(10).withLog()
		}
	}

	err = DatastoreClient.Get(r.Access.Request.Context(), r.Key, r)
	if err != nil {
		return errorDatastoreRead.withCause(err).withStack(10).withLog()
	}

	if data, ok := r.Data.(DataAfterLoad); ok {
		err = data.AfterLoad(r)
		if err != nil {
			return errorUnknown.withCause(err).withStack(10).withLog()
		}
	}

	return nil
}


func (r *Resource) GetMany() error {
	var err error
	r.EnterAction(ActionReadMany)
	defer r.ExitAction(ActionReadMany)

	// key must be incomplete
	if !r.Key.Incomplete() {
		return errorInvalidPath.withHint("Lists only works under models, not ids: remove the id from the end of path").withStack(10).withLog()
	}

	err = ValidateKey(r.Key)
	if err != nil {
		return errorUnknown.withCause(err).withStack(10).withLog()
	}

	q := datastore.NewQuery(r.Key.Kind)
	if r.Key.Parent != nil {
		q = q.Filter("Parent =", r.Key.Parent)
	}

	err = r.RunListQuery(q)
	if err != nil {
		return errorUnknown.withCause(err).withStack(10).withLog()
	}

	return nil
}

func (r *Resource) GetManyCount() error {
	var err error
	r.EnterAction(ActionReadManyCount)
	defer r.ExitAction(ActionReadManyCount)

	// key must be incomplete
	if !r.Key.Incomplete() {
		return errorInvalidPath.withHint("Lists only works under models, not ids: remove the id from the end of path").withStack(10).withLog()
	}

	err = ValidateKey(r.Key)
	if err != nil {
		return errorUnknown.withCause(err).withStack(10).withLog()
	}

	q := datastore.NewQuery(r.Key.Kind)
	if r.Key.Parent != nil {
		q = q.Filter("Parent =", r.Key.Parent)
	}

	count, err := DatastoreClient.Count(r.Access.Request.Context(), q)
	if err != nil {
		return errorUnknown.withCause(err).withStack(10).withLog()
	}

	r.ResourcesCount = &count

	return nil
}



func (r *Resource) GetAny() error {
	var err error
	r.EnterAction(ActionReadAny)
	defer r.ExitAction(ActionReadAny)

	// key must be incomplete
	if !r.Key.Incomplete() {
		return errorInvalidPath
	}

	// err = ValidateKey(r.Key)
	// if err != nil {
	// 	return err
	// }

	log.Println(r.Key.String(), r.Key.Kind)

	q := datastore.NewQuery(r.Key.Kind)
	if r.Key.Parent != nil {
		q = q.Ancestor(r.Key.Parent)
	}

	err = r.RunListQuery(q)
	if err != nil {
		return errorUnknown.withCause(err).withStack(10).withLog()
	}

	return nil
}

const (
	headerListNext            = "X-GetMany-Next-Cursor"
	headerListLimit           = "X-GetMany-Limit"
	headerListPageSize        = "X-GetMany-Page-Size"
	headerListFieldAscending  = "X-GetMany-Field-Ascending"
	headerListFieldDescending = "X-GetMany-Field-Descending"
)

func (r *Resource) RunListQuery(q *datastore.Query) error {
	var err error

	q = q.Limit(ListSizeDefault)

	// use cursor
	var cursor datastore.Cursor
	next := r.Access.Request.Header.Get(headerListNext)
	if next != "" {
		cursor, err = datastore.DecodeCursor(next)
		if err != nil {
			return errorDatastoreInvalidCursor.withCause(err).withStack(10)
		}
		q = q.Start(cursor)
	}

	if field := r.Access.Request.Header.Get(headerListFieldAscending); field != "" {
		q = q.Order(field)
	}

	if field := r.Access.Request.Header.Get(headerListFieldDescending); field != "" {
		q = q.Order("-" + field)
	}

	// finally, run one page!
	ite := DatastoreClient.Run(r.Access.Request.Context(), q)

	for i := 0; i < ListSizeDefault; i++ {

		// this is the use case of a NewClone() method for r
		nr := new(Resource)
		nr.Access = r.Access
		nr.ActionsStack = r.ActionsStack
		
		nr.Data, err = NewObject(r.Key.Kind)
		if err != nil {
			return err
		}

		nrTemp := new(Resource)
		nrTemp.Access = r.Access
		nrTemp.Data, err = NewObject(r.Key.Kind)
		if err != nil {
			return err
		}

		var iteErr error
		nr.Key, iteErr = ite.Next(nrTemp)
		if iteErr == iterator.Done {
			r.Next = ""
			break
		} else if iteErr != nil {
			return errorUnknown.withCause(iteErr).withStack(10).withLog()
		}

		// TODO: Just for using BeforeLoad we need to copy data two times because of the temp. Check if next and get are equivalent and use only one get.
		if data, ok := nr.Data.(DataBeforeLoad); ok {
			err = data.BeforeLoad(nr)
			if err != nil {
				if err := nrTemp.CopyData(nr); err != nil {
					log.Print(err)
				}
				r.Resources = append(r.Resources, nr)
				return errorUnknown.withCause(err).withStack(10)
			}
		}

		if err := nrTemp.CopyData(nr); err != nil {
			return errorUnknown.withCause(err).withStack(10)
		}

		if data, ok := nr.Data.(DataAfterLoad); ok {
			err = data.AfterLoad(nr)
			if err != nil {
				r.Resources = append(r.Resources, nr)
				return errorUnknown.withCause(err).withStack(10)
			}
		}

		r.Resources = append(r.Resources, nr)

		cursor, err = ite.Cursor()
		if err == nil {
			r.Next = cursor.String()
		}
	}

	length := len(r.Resources)
	r.ResourcesCount = &length
	return nil
}

func (r *Resource) Delete() error {
	var err error
	r.EnterAction(ActionDelete)
	defer r.ExitAction(ActionDelete)

	err = r.Get()
	if err != nil {
		return err
	}

	if data, ok := r.Data.(DataBeforeDelete); ok {
		err = data.BeforeDelete(r)
		if err != nil {
			return errorUnknown.withCause(err).withStack(10).withLog()
		}
	}

	err = DatastoreClient.Delete(r.Access.Request.Context(), r.Key)
	if err != nil {
		return errorDatastoreDelete.withCause(err).withStack(10).withLog()
	}

	if data, ok := r.Data.(DataAfterDelete); ok {
		err = data.AfterDelete(r)
		if err != nil {
			return errorUnknown.withCause(err).withStack(10).withLog()
		}
	}

	return nil
}

