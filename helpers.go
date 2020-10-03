package aeio

import (
	"cloud.google.com/go/datastore"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"golang.org/x/crypto/pbkdf2"
	"log"
	"math"
	mathRand "math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func Forbid(r *Resource) {
	// var m runtime.MemStats
	// runtime.ReadMemStats(&m)
	// r.Log("memory", errors.New(fmt.Sprintf("Memory usage: %d bytes (%d system).", m.Alloc, m.Sys)))


	r.Access.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")

	if len(r.Errors) > 0 && r.Errors[0].Reference == "not_authorized" {
		r.Access.Writer.WriteHeader(http.StatusUnauthorized)

		log.Printf("%d %s\t%s", http.StatusUnauthorized, r.Access.Request.Method, r.Access.Request.URL.Path)

	} else {
		r.Access.Writer.WriteHeader(http.StatusForbidden)
		log.Printf("%d %s\t%s", http.StatusForbidden, r.Access.Request.Method, r.Access.Request.URL.Path)
	}

	j, _ := json.Marshal(r)
	r.Access.Writer.Write(j)
}

func Allow(r *Resource) {
	// var m runtime.MemStats
	// runtime.ReadMemStats(&m)
	// r.Log("memory", errors.New(fmt.Sprintf("Memory usage: %d bytes (%d system).", m.Alloc, m.Sys)))

	r.Access.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	r.Access.Writer.WriteHeader(http.StatusOK)
	log.Printf("%d %s\t%s", http.StatusOK, r.Access.Request.Method, r.Access.Request.URL.Path)
	j, _ := json.Marshal(r)
	r.Access.Writer.Write(j)
}

func Neglect(r *Resource) {
	// var m runtime.MemStats
	// runtime.ReadMemStats(&m)
	// r.Log("memory", errors.New(fmt.Sprintf("Memory usage: %d bytes (%d system).", m.Alloc, m.Sys)))

	r.Access.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")

	if len(r.Errors) > 0 && r.Errors[0].Reference == "not_authorized" {
		r.Access.Writer.WriteHeader(http.StatusNotFound)
		log.Printf("%d %s\t%s", http.StatusNotFound, r.Access.Request.Method, r.Access.Request.URL.Path)

	} else {
		r.Access.Writer.WriteHeader(http.StatusNotFound)
		log.Printf("%d %s\t%s", http.StatusNotFound, r.Access.Request.Method, r.Access.Request.URL.Path)
	}

	j, _ := json.Marshal(r)
	r.Access.Writer.Write(j)
}

// AncestorKey returns the key of the nearest specified kind ancestor for any given key.
// If there is no ancestors, it checks if itself is the kind and return itself.
func AncestorKey(k *datastore.Key, kind string) (key *datastore.Key) {
	for {
		if k.Parent != nil {
			k = k.Parent
			if k.Kind == kind {
				return k
			}
			continue
		}
		if k.Kind == kind {
			return k
		}
	}
}

func OwnerKey(k *datastore.Key) (key *datastore.Key) {
	for {
		if k.Parent == nil {
			return k
		}
		k = k.Parent
	}
}

func NoZeroTime(t time.Time) time.Time {
	if t.IsZero() {
		t = time.Now().Local()
	}
	return t
}

var ValidPath = regexp.MustCompile(`^(?:/[a-z]+/[0-9]+)*(/[a-z]+)?$`)

// Timing is used to time the processing of resources.
func (r *Resource) Timing(s time.Time) {
	r.Time = int64(time.Since(s) / time.Millisecond)
}

//TODO: revision of all helpers.go functions, maybe should return errors also
func String(v interface{}) (s string) {
	switch v := v.(type) {
	case float32:
		s = strconv.FormatFloat(float64(v), 'f', -1, 64)
	case float64:
		s = strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		s = strconv.FormatInt(int64(v), 10)
	case int8:
		s = strconv.FormatInt(int64(v), 10)
	case int16:
		s = strconv.FormatInt(int64(v), 10)
	case int32:
		s = strconv.FormatInt(int64(v), 10)
	case int64:
		s = strconv.FormatInt(v, 10)
	case uint:
		s = strconv.FormatUint(uint64(v), 10)
	case uint8:
		s = strconv.FormatUint(uint64(v), 10)
	case uint16:
		s = strconv.FormatUint(uint64(v), 10)
	case uint32:
		s = strconv.FormatUint(uint64(v), 10)
	case uint64:
		s = strconv.FormatUint(uint64(v), 10)
	default:
		s = ""
	}
	return
}

func StringLatin(v interface{}) (s string) {
	switch v := v.(type) {
	case float32:
		s = strconv.FormatFloat(float64(v), 'f', 2, 64)
	case float64:
		s = strconv.FormatFloat(v, 'f', 2, 64)
	default:
		s = ""
	}
	return strings.Replace(s, ".", ",", 1)
}

func ParseFloat(v string) (f float64) {
	rex := regexp.MustCompile("[^0-9.]+")
	clean := rex.ReplaceAllString(v, "")
	f, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		f = 0
	}
	return
}

func ParseInt(v string) (i int) {
	rex := regexp.MustCompile("[^0-9.]+")
	clean := rex.ReplaceAllString(v, "")
	f, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		f = 0
	}
	return int(f + math.Copysign(0, f))
}

func Round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func Fixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(Round(num*output)) / output
}

func Cents(num float64) int {
	n := Fixed(num, 2)
	return Round(n * 100)
}

func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func GenerateRandomString(n int) (string, error) {
	b, err := GenerateRandomBytes(n)
	return base64.URLEncoding.EncodeToString(b), err
}

const letterBytes = "0123456789"
const letterIdxBits = 6                    // 6 bits to represent a letter index
const letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
const letterIdxMax = 63 / letterIdxBits    // # of letter indices fitting in 63 bits

func GenerateRandomNumber(n int) string {
	if n <= 0 {
		return ""
	}

	var src = mathRand.NewSource(time.Now().UnixNano())

	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

func HashString(password string, salt string) string {
	b := pbkdf2.Key([]byte(password), []byte(salt), 4096, 512, sha512.New)
	return base64.URLEncoding.EncodeToString(b)
}

func RemoveUnsafe(s string) string {
	reg, _ := regexp.Compile("[^A-Za-z]+")
	return reg.ReplaceAllString(s, "")
}
