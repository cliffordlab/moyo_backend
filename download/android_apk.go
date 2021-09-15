package download

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

type APKDownloadHandler struct {
	Name string
	Svc  *s3.S3
}

func (U APKDownloadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("Serving AMoSS APK " + U.Name)

	var key string

	switch U.Name {
	case "Download UTSW handler":
		key = "moyo-utsw-v1.0.0.0.apk"
	case "Download MSM handler":
		key = "amoss-moyo-release.apk"
	case "Download HF handler":
		key = "amoss-hf-release-v1.0.0.0.apk"
	}

	log.Println("key: " + key)
	presignedURL := GetS3PreSignedUrl(key, U)
	http.Redirect(w, r, presignedURL, http.StatusFound)
}

func GetS3PreSignedUrl(key string, u APKDownloadHandler) string {
	expiration := time.Duration(10080)
	log.Println("Creating presigned URL...")

	req, _ := u.Svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String("amoss-moyo-apk"),
		Key:    aws.String(key),
	})

	preSignedURL, err := req.Presign(expiration * time.Minute)
	if err != nil {
		fmt.Println("Failed to sign request", err)
	}
	log.Println("here is the presigned URL:  ")
	log.Println(preSignedURL)
	return preSignedURL
}
