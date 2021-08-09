package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	s "strings"

	"github.com/aws/aws-sdk-go/service/s3"
)

type EmailHandler struct {
	Name string
	Svc  *s3.S3
	Type string
}

type EmailRequest struct {
	Email string `json:"email"`
}

//const errorResJSON = `{"error":"json parsing error","error description":"key or value of json is formatted incorrectly"}`

func (eh EmailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	var er EmailRequest
	if err := dec.Decode(&er); err != nil {
		log.Println(err)
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Write([]byte(errorResJSON))
		return
	}

	text := er.Email
	if !s.Contains(text, "@") {
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Write([]byte("{\"failure\": \"your request was not completed please an email\"}"))
		return
	}

	bucket := "emmha"
	key := eh.Type + "/" + text

	uploadResult, err := eh.Svc.PutObject(&s3.PutObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		log.Printf("Failed to upload data to %s/%s, %s\n", bucket, key, err.Error())
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Write([]byte("{\"failure\": \"your request was not completed\"}"))
		return
	}

	log.Printf("This is the result of the upload: %s\n{Key: %s, Success: full}\n", uploadResult.GoString(), key)
	w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	w.Write([]byte("{\"success\": \"you have completed post to moyo\"}"))
}
