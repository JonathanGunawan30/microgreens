package conv

import (
	"strconv"

	"github.com/gosimple/slug"
)

func GenerateSlug(title string) string {
	return slug.Make(title)
}

func StringToInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}
