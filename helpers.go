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

// Respond writes to the resource writer with the selected status and headers. After calling Respond, the connection
// will be closed, and so it's context. Any process on the request stack may continue after it, but any process using
// the request.Context will be finished.
//
// If the passed status is not http valid, it will be responded http.StatusInternalServerError (500) and the resource will be
// receive the error reference "invalid_status".
//
// If the resource errors contains the reference "not_authorized", the status will be http.StatusForbidden (403) independently
// of the status passed to Respond.
func (r *Resource) Respond() error {
	var err error

	var status int = http.StatusOK

	if r.Error != nil {
		status = r.Error.(Error).HttpStatus
		if http.StatusText(status) == "" {
			return NewError("invalid_status", nil, http.StatusInternalServerError)
		}
	}


	r.Access.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")

	r.Access.Writer.WriteHeader(status)
	log.Printf("%d %s\t%s", status, r.Access.Request.Method, r.Access.Request.URL.Path)

	j, err := json.Marshal(r)
	if err != nil {
		return NewError("marshaling_response", err, http.StatusInternalServerError)
	}

	_, err = r.Access.Writer.Write(j)
	if err != nil {
		return NewError("writing_response", err, http.StatusInternalServerError)
	}
	return nil
}

// AncestorKindKey returns the key of the nearest specified kind ancestor for any given key.
// Only if there is no ancestors, it checks if itself is the kind and return itself.
// If there is no element of the kind, returns nil.
func AncestorKindKey(k *datastore.Key, kind string) *datastore.Key {
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
		} else {
			return nil
		}
	}
}

// RootKey returns the deepest ancestor of the passed key.
func RootKey(k *datastore.Key) (key *datastore.Key) {
	for {
		if k.Parent == nil {
			return k
		}
		k = k.Parent
	}
}

// NoZeroTime checks if time is zero, and if so, returns the actual time.
func NoZeroTime(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now().UTC()
	}
	return t
}

var validPath = regexp.MustCompile(`^(?:/[a-z]+/[0-9]+)*(/[a-z]+)?$`)

// Timing is used to time the processing of resources.
func (r *Resource) Timing(start time.Time) {
	r.TimeElapsed = int64(time.Since(start) / time.Millisecond)
}

// String TODO: revision of all helpers.go functions, maybe should return errors also
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
		s = strconv.FormatUint(v, 10)
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
