package aeio

import (
	"encoding/json"
	"google.golang.org/appengine/datastore"
	"net/http"
	"regexp"
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

// Timing is used to time the processing of resources.
func (r *Resource) Timing(s time.Time) {
	r.Time = int64(time.Since(s) / time.Millisecond)
}
