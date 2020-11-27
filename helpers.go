package aeio

import (
	"cloud.google.com/go/datastore"
	"log"
	"math"
	"net/http"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

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
// Very useful for checking Users path for example.
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

// // String TODO: revision of all helpers.go functions, maybe should return errors also
// func String(v interface{}) (s string) {
// 	switch v := v.(type) {
// 	case float32:
// 		s = strconv.FormatFloat(float64(v), 'f', -1, 64)
// 	case float64:
// 		s = strconv.FormatFloat(v, 'f', -1, 64)
// 	case int:
// 		s = strconv.FormatInt(int64(v), 10)
// 	case int8:
// 		s = strconv.FormatInt(int64(v), 10)
// 	case int16:
// 		s = strconv.FormatInt(int64(v), 10)
// 	case int32:
// 		s = strconv.FormatInt(int64(v), 10)
// 	case int64:
// 		s = strconv.FormatInt(v, 10)
// 	case uint:
// 		s = strconv.FormatUint(uint64(v), 10)
// 	case uint8:
// 		s = strconv.FormatUint(uint64(v), 10)
// 	case uint16:
// 		s = strconv.FormatUint(uint64(v), 10)
// 	case uint32:
// 		s = strconv.FormatUint(uint64(v), 10)
// 	case uint64:
// 		s = strconv.FormatUint(v, 10)
// 	default:
// 		s = ""
// 	}
// 	return
// }

// func StringLatin(v interface{}) (s string) {
// 	switch v := v.(type) {
// 	case float32:
// 		s = strconv.FormatFloat(float64(v), 'f', 2, 64)
// 	case float64:
// 		s = strconv.FormatFloat(v, 'f', 2, 64)
// 	default:
// 		s = ""
// 	}
// 	return strings.Replace(s, ".", ",", 1)
// }

// func ParseFloat(v string) (f float64) {
// 	rex := regexp.MustCompile("[^0-9.]+")
// 	clean := rex.ReplaceAllString(v, "")
// 	f, err := strconv.ParseFloat(clean, 64)
// 	if err != nil {
// 		f = 0
// 	}
// 	return
// }

func parseInt(v string) (i int) {
	rex := regexp.MustCompile("[^0-9.]+")
	clean := rex.ReplaceAllString(v, "")
	f, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		f = 0
	}
	return int(f + math.Copysign(0, f))
}

// func Round(num float64) int {
// 	return int(num + math.Copysign(0.5, num))
// }

// func Fixed(num float64, precision int) float64 {
// 	output := math.Pow(10, float64(precision))
// 	return float64(Round(num*output)) / output
// }

// func Cents(num float64) int {
// 	n := Fixed(num, 2)
// 	return Round(n * 100)
// }

// func GenerateRandomBytes(n int) ([]byte, error) {
// 	b := make([]byte, n)
// 	_, err := rand.Get(b)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return b, nil
// }

// func GenerateRandomString(n int) (string, error) {
// 	b, err := GenerateRandomBytes(n)
// 	return base64.URLEncoding.EncodeToString(b), err
// }

// const letterBytes = "0123456789"
// const letterIdxBits = 6                    // 6 bits to represent a letter index
// const letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
// const letterIdxMax = 63 / letterIdxBits    // # of letter indices fitting in 63 bits

// func GenerateRandomNumber(n int) string {
// 	if n <= 0 {
// 		return ""
// 	}
//
// 	var src = mathRand.NewSource(time.Now().UnixNano())
//
// 	b := make([]byte, n)
// 	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
// 	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
// 		if remain == 0 {
// 			cache, remain = src.Int63(), letterIdxMax
// 		}
// 		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
// 			b[i] = letterBytes[idx]
// 			i--
// 		}
// 		cache >>= letterIdxBits
// 		remain--
// 	}
//
// 	return string(b)
// }

// func HashString(password string, salt string) string {
// 	b := pbkdf2.Key([]byte(password), []byte(salt), 4096, 512, sha512.New)
// 	return base64.URLEncoding.EncodeToString(b)
// }

// func RemoveUnsafe(s string) string {
// 	reg, _ := regexp.Compile("[^A-Za-z]+")
// 	return reg.ReplaceAllString(s, "")
// }


// NewResourceFromRequest initializes a base resource with information from the request.
func NewResourceFromRequest(writer *http.ResponseWriter, request *http.Request) (*Resource, error) {
	r := &Resource{}
	r.Access = newAccess(writer, request)
	r.Key = Key(r.Access.Request.URL.Path)
	if r.Key == nil {
		return r, errorInvalidPath.withStack()
	}
	return r, nil
}

// InitResource uses a Key to initialize a specific resource beyond the root resource. The resource returned is ready to be actioned.
// Examples are returning sub resources or even something outside the scope of root.
// You can pass an incomplete Key to complete when putting the object on the datastore.
func InitResource(access *Access, key *datastore.Key) (r *Resource) {
	return &Resource{Key: key, Access: access}
}

// NewResource is used to create empty children resources. Parent path may be the "" string (root). It returns a resource with an incompleteKey.
// It initializes an object of type kind.
// TODO: swap access and parentKey for parentResource
func NewResource(access *Access, parentKey *datastore.Key, kind string) (*Resource, error) {
	r := &Resource{Access: access}

	if parentKey == nil {
		parentKey = datastore.IncompleteKey("", nil)
	}

	err := ValidatePaternity(parentKey.Kind, kind)
	if err != nil {
		return nil, err
	}

	if parentKey.Kind == "" {
		r.Key = Key("/" + kind)
	} else {
		r.Key = Key(Path(parentKey) + "/" + kind)
	}

	if r.Key == nil {
		return nil, errorInvalidPath.withStack()
	}
	r.Data, err = NewObject(kind)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// Key transforms a path in a datastore key using an access.
func Key(path string) (k *datastore.Key) {
	var kd string
	var id int64
	var p []string

	if path == "" {
		log.Print("path to key from empty path")
		return nil
	}

	if validPath.MatchString(path) != true {
		_, file, line, _ := runtime.Caller(2)
		log.Println("invalid_regex_path", path, file, line)
		return nil
	}

	p = strings.SplitN(path, "/", 2)
	p = strings.Split(p[1], "/")
	for i := 0; i < len(p); i = i + 2 {
		kd = p[i]
		if i < len(p)-1 {
			id, _ = strconv.ParseInt(p[i+1], 10, 64)
			k = datastore.IDKey(kd, id, k)
		} else {
			k = datastore.IncompleteKey(kd, k)
		}
	}
	return k
}

// Path transforms a datastore Key into a Path
func Path(k *datastore.Key) (p string) {
	if k.Incomplete() == false {
		p = "/" + strconv.FormatInt(k.ID, 10)
	}
	p = "/" + k.Kind + p
	for {
		k = k.Parent
		if k == nil {
			break
		}
		p = "/" + k.Kind + "/" + strconv.FormatInt(k.ID, 10) + p
	}
	return p
}

