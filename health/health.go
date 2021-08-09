package health

import (
	"log"
	"net/http"
)

var (
	healthStatus = http.StatusOK
)

// Handler responds to health check requests.
func 	Handler(w http.ResponseWriter, r *http.Request) {
	log.Println("Server is healthy")
	w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	w.WriteHeader(healthStatus)
}
