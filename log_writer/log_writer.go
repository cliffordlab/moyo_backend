package log_writer

import (
	"log"
	"net/http"
)

type LogWriter struct {
	http.ResponseWriter
}

func (w LogWriter) Write(p []byte) (n int, err error) {
	n, err = w.ResponseWriter.Write(p)
	if err != nil {
		log.Printf("Write failed: %v", err)
	}
	return
}
