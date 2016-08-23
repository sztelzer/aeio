package aeio

import (
	"encoding/json"
	"errors"
	"google.golang.org/appengine/datastore"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func Forbid(r *Resource) {
	// var m runtime.MemStats
	// runtime.ReadMemStats(&m)
	// r.L("memory", errors.New(fmt.Sprintf("Memory usage: %d bytes (%d system).", m.Alloc, m.Sys)))

	r.Access.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")

	if len(r.Errors) > 0 && r.Errors[0].Reference == "not_authorized" {
		r.Access.Writer.WriteHeader(http.StatusUnauthorized)
	} else {
		r.Access.Writer.WriteHeader(http.StatusForbidden)
	}

	j, _ := json.Marshal(r)
	r.Access.Writer.Write(j)
}

func Allow(r *Resource) {
	// var m runtime.MemStats
	// runtime.ReadMemStats(&m)
	// r.L("memory", errors.New(fmt.Sprintf("Memory usage: %d bytes (%d system).", m.Alloc, m.Sys)))

	r.Access.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	r.Access.Writer.WriteHeader(http.StatusOK)
	j, _ := json.Marshal(r)
	r.Access.Writer.Write(j)
}

func Neglect(r *Resource) {
	// var m runtime.MemStats
	// runtime.ReadMemStats(&m)
	// r.L("memory", errors.New(fmt.Sprintf("Memory usage: %d bytes (%d system).", m.Alloc, m.Sys)))

	r.Access.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")

	if len(r.Errors) > 0 && r.Errors[0].Reference == "not_authorized" {
		r.Access.Writer.WriteHeader(http.StatusNotFound)
	} else {
		r.Access.Writer.WriteHeader(http.StatusNotFound)
	}

	j, _ := json.Marshal(r)
	r.Access.Writer.Write(j)
}

//Key return a key from a path using Access context to build it.
func Key(a *Access, path string) (k *datastore.Key, err error) {
	var kd string
	var id int64
	var p []string

	if path == "" {
		return nil, errors.New("Invalid Path: " + path)
	}

	if ValidPath.MatchString(path) != true {
		return nil, errors.New("Invalid Path: " + path)
	}

	p = strings.SplitN(path, "/", 2)
	p = strings.Split(p[1], "/")
	for i := 0; i < len(p); i = i + 2 {
		kd = p[i]
		if i < len(p)-1 {
			id, _ = strconv.ParseInt(p[i+1], 10, 64)
			k = datastore.NewKey(*a.Context, kd, "", id, k)
		} else {
			k = datastore.NewIncompleteKey(*a.Context, kd, k)
		}
	}
	return k, nil
}

//Path returns the simple path from a Key in the format resource/id/resource/id...
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

// AncestorKey returns the key of the nearest specified kind ancestor for any given key.
// If there is no ancestors, it checks if itself is the kind and return itself.
func AncestorKey(k *datastore.Key, kind string) (key *datastore.Key) {
	for {
		if k.Parent() != nil {
			k = k.Parent()
			if k.Kind() == kind {
				return k
			}
			continue
		}
		if k.Kind() == kind {
			return k
		}
	}
	return nil
}

func NoZeroTime(t time.Time) time.Time {
	if t.IsZero() {
		t = time.Now().Local()
	}
	return t
}

var ValidPath = regexp.MustCompile(`^(?:/[a-z]+/[0-9]+)*$|^(?:/[a-z]+/[0-9]+)*(?:/[a-z]+){1}$`)



// CheckAncestry goes to the datastore to verify the existence of all ancestry parts of the resource.
// It has a small cost as it does various queries, but uses counts.
// TODO:  Maybe could use BatchQuerie.
func (r *Resource) CheckAncestry() (err error) {
	// defer runtime.GC()
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
