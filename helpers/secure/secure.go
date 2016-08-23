package secure

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"golang.org/x/crypto/pbkdf2"
	mathrand "math/rand"
	"regexp"
	"time"
)

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

	var src = mathrand.NewSource(time.Now().UnixNano())

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
