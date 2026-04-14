package httpd

import (
	"context"
	"net/http"
	"strconv"
)

func UidSetStr(r *http.Request, uid string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), "uid", uid)) // nolint
}
func UidGetStr(r *http.Request) string {
	value := r.Context().Value("uid")
	if value == nil {
		return ""
	}
	switch s := value.(type) {
	case string:
		return s
	case int64:
		return strconv.FormatInt(s, 10)
	case int32:
		return strconv.FormatInt(int64(s), 10)
	case int8:
		return strconv.FormatInt(int64(s), 10)
	case int16:
		return strconv.FormatInt(int64(s), 10)
	default:
		return ""
	}
}
