package conv

import (
	"strconv"
)

func StringToInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func StrPtr(s string) *string {
	return &s
}
