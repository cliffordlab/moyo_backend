package checkHTTP

import (
	"fmt"
	"net/http"
	"strings"
)

//IsMultipart returns true if the given request is multipart forrm
func IsMultipart(r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	fmt.Printf("This is the content type value %s\n", contentType)
	return strings.Contains(contentType, "multipart/form-data")
}
