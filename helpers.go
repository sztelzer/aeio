package aeio

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"net/http"
	"time"
	"google.golang.org/appengine/datastore"
	"regexp"
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

// Path returns the simple path from a Key in the format resource/id/resource/id...
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
func AncestorKey(k *datastore.Key, kind string) (key *datastore.Key) {
	for {
		if k.Parent() != nil {
			k = k.Parent()
			if k.Kind() == kind {
				return k
			}
			continue
		}
		break
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
