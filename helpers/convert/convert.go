package convert

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

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
