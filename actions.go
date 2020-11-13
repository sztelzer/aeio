package aeio

import (
	"cloud.google.com/go/datastore"
	"fmt"
	"google.golang.org/api/iterator"
	"log"
	"net/http"

	// "google.golang.org/api/iterator"
)

// Update saves: create, overwrite completely, or overwrite parts
func (r *Resource) Put() error {
	var err error
	r.EnterAction(actionPut)
	defer r.ExitAction(actionPut)

	err = ValidateKey(r.Key)
	if err != nil {
		return err
	}

	if r.Access.Request.Method == http.MethodPatch {
		err = r.Get()
		if err != nil {
			return err
		}
	}

	err = r.BindRequestData()
	if err != nil {
		return err
	}

	if data, ok := r.Data.(DataBeforeSave); ok {
		err := data.BeforeSave(r)
		if err != nil {
			return errorUnknown.withCause(err).withStack().withLog()
		}
	}

	r.Key, err = DatastoreClient.Put(r.Access.Request.Context(), r.Key, r)
	if err != nil {
		return errorDatastorePut.withCause(err).withStack()
	}

	if data, ok := r.Data.(DataAfterSave); ok {
		err := data.AfterSave(r)
		if err != nil {
			return errorUnknown.withCause(err).withStack().withLog()
		}
	}
	return err
}


// Read is an action that reads a resource from datastore. It always replace the object present with a new one of the right kind.
func (r *Resource) Get() error {
	var err error
	r.EnterAction(actionGet)
	defer r.ExitAction(actionGet)

	if r.Key.Incomplete() {
		return errorInvalidPath.withHint(fmt.Sprintf("%s", "The key passed to get is incomplete"))
	}

	err = ValidateKey(r.Key)
	if err != nil {
		return err
	}

	r.Data, err = NewObject(r.Key.Kind)
	if err != nil {
		return err
	}

	if data, ok := r.Data.(DataBeforeLoad); ok {
		err = data.BeforeLoad(r)
		if err != nil {
			return err
		}
	}

	err = DatastoreClient.Get(r.Access.Request.Context(), r.Key, r)
	if err != nil {
		return errorDatastoreRead.withCause(err)
	}

	if data, ok := r.Data.(DataAfterLoad); ok {
		err = data.AfterLoad(r)
		if err != nil {
			return err
		}
	}

	return err
}


func (r *Resource) List() error {
	var err error
	r.EnterAction(actionList)
	defer r.ExitAction(actionList)

	// key must be incomplete
	if !r.Key.Incomplete() {
		return errorInvalidPath.withHint("Lists only works under models, not ids: remove the id from the end of path")
	}

	err = ValidateKey(r.Key)
	if err != nil {
		return err
	}

	q := datastore.NewQuery(r.Key.Kind)
	if r.Key.Parent != nil {
		q = q.Filter("Parent =", r.Key.Parent)
	}

	err = r.RunListQuery(q)

	return err
}

func (r *Resource) ListAny() error {
	var err error
	r.EnterAction(actionListAny)
	defer r.ExitAction(actionListAny)

	// key must be incomplete
	if !r.Key.Incomplete() {
		return errorInvalidPath
	}

	err = ValidateKey(r.Key)
	if err != nil {
		return err
	}

	log.Println(r.Key.String(), r.Key.Kind)

	q := datastore.NewQuery(r.Key.Kind)
	if r.Key.Parent != nil {
		q = q.Ancestor(r.Key.Parent)
	}

	err = r.RunListQuery(q)
	return err
}

const (
	headerListNext            = "X-List-Next-Cursor"
	headerListLimit           = "X-List-Limit"
	headerListPageSize        = "X-List-Page-Size"
	headerListFieldAscending  = "X-List-Field-Ascending"
	headerListFieldDescending = "X-List-Field-Descending"
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
			return errorDatastoreInvalidCursor.withCause(err).withStack()
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
			log.Println("HERE!!")
			return errorUnknown.withCause(iteErr).withStack().withLog()
		}

		// TODO: Just for using BeforeLoad we need to copy data two times because of the temp. Check if next and get are equivalent and use only one get.
		if data, ok := nr.Data.(DataBeforeLoad); ok {
			err = data.BeforeLoad(nr)
			if err != nil {
				if err := nrTemp.CopyData(nr); err != nil {
					log.Print(err)
				}
				r.Resources = append(r.Resources, nr)
				return errorUnknown.withCause(err).withStack()
			}
		}

		if err := nrTemp.CopyData(nr); err != nil {
			return errorUnknown.withCause(err).withStack()
		}

		if data, ok := nr.Data.(DataAfterLoad); ok {
			err = data.AfterLoad(nr)
			if err != nil {
				r.Resources = append(r.Resources, nr)
				return errorUnknown.withCause(err).withStack()
			}
		}

		r.Resources = append(r.Resources, nr)

		cursor, err = ite.Cursor()
		if err == nil {
			r.Next = cursor.String()
		}
	}

	r.ResourcesCount = len(r.Resources)
	return nil
}

func (r *Resource) Delete() error {
	var err error
	r.EnterAction("delete")
	defer r.ExitAction("delete")

	err = r.Get()
	if err != nil {
		return err
	}

	if data, ok := r.Data.(DataBeforeDelete); ok {
		err = data.BeforeDelete(r)
		if err != nil {
			return errorUnknown.withCause(err).withStack().withLog()
		}
	}

	err = DatastoreClient.Delete(r.Access.Request.Context(), r.Key)
	if err != nil {
		return errorDatastoreDelete.withCause(err).withStack().withLog()
	}

	if data, ok := r.Data.(DataAfterDelete); ok {
		err = data.AfterDelete(r)
		if err != nil {
			return errorUnknown.withCause(err).withStack().withLog()
		}
	}

	return nil
}

