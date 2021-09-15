package amoss_streams

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/cliffordlab/amoss_services/capacity"
	check "github.com/cliffordlab/amoss_services/checkHTTP"
	"github.com/cliffordlab/amoss_services/database"
	"github.com/cliffordlab/amoss_services/participant"
	"github.com/dgrijalva/jwt-go"
)

const (
	defaultMaxMemory = 32 << 20
	partialSucess    = `{"partial success":"able to upload some data to awsS3Bucket files",
    "description":"all files were not able to be upload may be due to empty files"}`
	invalidAccessToken = `{"logout user":"Invalid access token."}`
	insertVitalsData   = `INSERT INTO bp_readings 
(created_at, participant_id, systolic_bp, diastolic_bp, pulse, jpg_s3_key, csv_s3_key) 
VALUES($1, $2, $3, $4, $5, $6, $7)`
)

//UploadHandler acts as a proxy between the mobile application and s3
type UploadHandler struct {
	Name string
	Svc  *s3.S3
}

type UploadMoyoHandler struct {
	Name string
	Svc  *s3.S3
}

type UploadHFHandler struct {
	Name string
	Svc  *s3.S3
}

//UploadUTSWHandler acts as a proxy between the mobile application and s3 for utsw project
type UploadUTSWHandler struct {
	Name string
	Svc  *s3.S3
}

func rangeIn(low, hi int) int {
	return low + rand.Intn(hi-low)
}

func setKey(currentParticipant participant.Participant, startOfWeekMillis string, filename string) string {
	var key string
	switch database.ADB.Environment {
	case "dev":
		key = "dev/" + currentParticipant.Study + "/" + strconv.FormatInt(currentParticipant.ID, 10) + "/" + startOfWeekMillis + "/" + filename
	case "local":
		key = "test/" + currentParticipant.Study + "/" + strconv.FormatInt(currentParticipant.ID, 10) + "/" + startOfWeekMillis + "/" + filename
	default:
		key = currentParticipant.Study + "/" + strconv.FormatInt(currentParticipant.ID, 10) + "/" + startOfWeekMillis + "/" + filename
	}
	return key
}

func (uh UploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//validate header is formatted properly
	headerValue := r.Header.Get("Authorization")
	splitHeaderValue := strings.Split(headerValue, " ")
	if splitHeaderValue[0] != "Mars" {
		http.Error(w, "Invalid token type", http.StatusUnauthorized)
		return
	}
	bearerToken := splitHeaderValue[1]
	token, err := jwt.ParseWithClaims(bearerToken, &capacity.NonAdminClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Make sure token's signature wasn't changed
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected siging method")
		}
		return []byte(capacity.JwtSecret), nil
	})

	if err != nil {
		log.Println("token not valid")
		http.Error(w, "Invalid token type", http.StatusUnauthorized)
		return
	}

	var currentParticipant participant.Participant
	if claims, ok := token.Claims.(*capacity.NonAdminClaims); ok && token.Valid {
		currentParticipant.Study = claims.Study
		currentParticipant.ID = claims.ID
	} else {
		log.Println("token not valid")
		http.Error(w, "Invalid token type", http.StatusUnauthorized)
		return
	}

	var bucket string
	err = r.ParseMultipartForm(defaultMaxMemory)

	var startOfWeekMillis string
	log.Println("This is the study: " + currentParticipant.Study)
	log.Println("This is the Participant ID: " + strconv.FormatInt(currentParticipant.ID, 10))
	startOfWeekMillis = r.Header.Get("weekMillis")
	if len(startOfWeekMillis) != 12 {
		log.Println("token length is wrong")
		w.Header().Add("Content-Type", "application/json; charset=UTF-8")
		w.Write([]byte("{\"error\": \"invalid header\"}"))
		return
	}
	if err != nil {
		partialKey := SetPartialKey(currentParticipant, startOfWeekMillis)
		log.Printf("{Error: %s, Key: %s}\n", err.Error(), partialKey)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	bucket = "awsS3Bucket"
	fullUpload := true

	log.Println("This is the bucket: " + bucket)
	//get a ref to the parsed multipart form
	m := r.MultipartForm
	path := r.Form.Get("path")
	var key string

	//get the *fileheaders
	files := m.File["upload"]
	for i, f := range files {
		//for each fileheader, get a handle to the actual file
		filename := files[i].Filename
		if path != "" {
			key = path + "/" + filename
		} else {

			key = setKey(currentParticipant, startOfWeekMillis, filename)
		}
		file, err := f.Open()
		defer file.Close()
		if err != nil {
			log.Printf("{Error: %s, Key: %s}\n", err.Error(), key)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		uploadResult, err := uh.Svc.PutObject(&s3.PutObjectInput{
			Bucket: &bucket,
			Key:    &key,
			Body:   file,
		})
		if err != nil {
			fullUpload = false
			log.Printf("Failed to upload data to %s/%s, %s\n", bucket, key, err.Error())
		}

		if !fullUpload {
			log.Printf("This is the result of the upload: %s\n{Key: %s, Success: partial}\n", uploadResult.GoString(), key)
			w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			w.Write([]byte(partialSucess))
		} else {
			log.Printf("This is the result of the upload: %s\n{Key: %s, Success: full}\n", uploadResult.GoString(), key)
			w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			w.Write([]byte("{\"success\": \"you have completed upload to awsS3Bucket\"}"))
		}
	}
}

func (uh UploadMoyoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startOfWeekMillis := r.Header.Get("weekMillis")
	if len(startOfWeekMillis) != 12 {
		log.Println("token length is wrong")
		w.Header().Add("Content-Type", "application/json; charset=UTF-8")
		w.Write([]byte("{\"error\": \"invalid header\"}"))
		return
	}
	//validate header is formatted properly
	headerValue := r.Header.Get("Authorization")
	splitHeaderValue := strings.Split(headerValue, " ")
	if splitHeaderValue[0] != "Mars" {
		http.Error(w, "Invalid token type", http.StatusUnauthorized)
		return
	}
	bearerToken := splitHeaderValue[1]
	token, err := jwt.ParseWithClaims(bearerToken, &capacity.NonAdminClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Make sure token's signature wasn't changed
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected siging method")
		}
		return []byte(capacity.JwtSecret), nil
	})

	if err != nil {
		log.Println("token not valid")
		http.Error(w, "Invalid token type", http.StatusUnauthorized)
		return
	}

	var currentParticipant participant.Participant
	if claims, ok := token.Claims.(*capacity.NonAdminClaims); ok && token.Valid {
		currentParticipant.Study = claims.Study
		currentParticipant.ID = claims.ID
	} else {
		log.Println("token not valid")
		http.Error(w, "Invalid token type", http.StatusUnauthorized)
		return
	}

	log.Println("Querying db for access token")
	query := "SELECT access_token FROM participants WHERE participant_id = $1 AND access_token = $2"
	rows, err := database.ADB.Db.Query(query, currentParticipant.ID, bearerToken)
	if err != nil {
		log.Println("failed to execute get access_token query")
		log.Println(err)
		http.Error(w, "Invalid access token. Please log user out", http.StatusForbidden)
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Write([]byte(invalidAccessToken))
		return
	}
	var accessTokenDB string
	for rows.Next() {
		err = rows.Scan(&accessTokenDB)
		if err != nil {
			log.Println("failed to scan row for access token")
			log.Println(err)
		}
	}
	rows.Close()

	if accessTokenDB == bearerToken {
		bucket := "awsS3Bucket"
		fullUpload := true

		err = r.ParseMultipartForm(defaultMaxMemory)
		if err != nil {
			partialKey := SetPartialKey(currentParticipant, startOfWeekMillis)

			log.Printf("{Error: %s, Key: %s}\n", err.Error(), partialKey)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		//get a ref to the parsed multipart form
		m := r.MultipartForm

		//get the *fileheaders
		files := m.File["upload"]
		for i, f := range files {
			//for each fileheader, get a handle to the actual file
			filename := files[i].Filename
			key := setKey(currentParticipant, startOfWeekMillis, filename)

			file, err := f.Open()
			defer file.Close()
			if err != nil {
				log.Printf("{Error: %s, Key: %s}\n", err.Error(), key)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			uploadResult, err := uh.Svc.PutObject(&s3.PutObjectInput{
				Bucket: &bucket,
				Key:    &key,
				Body:   file,
			})
			if err != nil {
				fullUpload = false
				log.Printf("Failed to upload data to %s/%s, %s\n", bucket, key, err.Error())
			}

			if !fullUpload {
				log.Printf("This is the result of the upload: %s\n{Key: %s, Success: partial}\n", uploadResult.GoString(), key)
			} else {
				log.Printf("This is the result of the upload: %s\n{Key: %s, Success: full}\n", uploadResult.GoString(), key)
			}
		}
		if !fullUpload {
			w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			w.Write([]byte(partialSucess))
		} else {
			w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			w.Write([]byte("{\"success\": \"you have completed upload to awsS3Bucket/moyo\"}"))
		}
	} else {
		log.Println("Access token does not match that of the database")
		log.Println("Participant_ID: ", currentParticipant.ID)
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Write([]byte(invalidAccessToken))
		return
	}
}

func (uh UploadUTSWHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startOfWeekMillis := r.Header.Get("weekMillis")
	if len(startOfWeekMillis) != 12 {
		log.Println("token length is wrong")
		w.Header().Add("Content-Type", "application/json; charset=UTF-8")
		w.Write([]byte("{\"error\": \"invalid header\"}"))
		return
	}

	bucket := "awsS3Bucket"
	fullUpload := true
	var currentParticipant participant.Participant

	err := r.ParseMultipartForm(defaultMaxMemory)
	if err != nil {
		partialKey := SetPartialKey(currentParticipant, startOfWeekMillis)
		log.Printf("{Error: %s, Key: %s}\n", err.Error(), partialKey)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//get a ref to the parsed multipart form
	m := r.MultipartForm

	//get the *fileheaders
	files := m.File["upload"]
	for i, f := range files {
		//for each fileheader, get a handle to the actual file
		filename := files[i].Filename
		key := setKey(currentParticipant, startOfWeekMillis, filename)
		file, err := f.Open()
		defer file.Close()
		if err != nil {
			log.Printf("{Error: %s, Key: %s}\n", err.Error(), key)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		uploadResult, err := uh.Svc.PutObject(&s3.PutObjectInput{
			Bucket: &bucket,
			Key:    &key,
			Body:   file,
		})
		if err != nil {
			fullUpload = false
			log.Printf("Failed to upload data to %s/%s, %s\n", bucket, key, err.Error())
		}

		if !fullUpload {
			log.Printf("This is the result of the upload: %s\n{Key: %s, Success: partial}\n", uploadResult.GoString(), key)
			w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			w.Write([]byte(partialSucess))
		} else {
			log.Printf("This is the result of the upload: %s\n{Key: %s, Success: full}\n", uploadResult.GoString(), key)
			w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			w.Write([]byte("{\"success\": \"you have completed upload to awsS3Bucket/utsw\"}"))
		}
	}
}

func (uh UploadHFHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startOfWeekMillis := r.Header.Get("weekMillis")
	if len(startOfWeekMillis) != 12 {
		log.Println("token length is wrong")
		w.Header().Add("Content-Type", "application/json; charset=UTF-8")
		w.Write([]byte("{\"error\": \"invalid header\"}"))
		return
	}
	//validate header is formatted properly
	headerValue := r.Header.Get("Authorization")
	splitHeaderValue := strings.Split(headerValue, " ")
	if splitHeaderValue[0] != "Mars" {
		http.Error(w, "Invalid token type", http.StatusUnauthorized)
		return
	}
	bearerToken := splitHeaderValue[1]
	token, err := jwt.ParseWithClaims(bearerToken, &capacity.NonAdminClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Make sure token's signature wasn't changed
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected siging method")
		}
		return []byte(capacity.JwtSecret), nil
	})

	if err != nil {
		log.Println("unable to parse with claims")
		log.Println("issuing new token")
		altID := rangeIn(100000000, 999999999)
		insertParticipant := `INSERT INTO participants (participant_id, password_hash, password_salt, capacity_id, study_id)
	VALUES ($1, $2, $3, (SELECT capacity_id FROM participant_capacity WHERE capacity_id=$4),
	(SELECT study_id FROM studies WHERE study_id=$5))`
		//TODO create user
		stmt, err := database.ADB.Db.Prepare(insertParticipant)
		if err != nil {
			log.Println("failed to prepare create coordinator statement")
			log.Fatalln(err)
		}
		defer stmt.Close()

		rows, err := stmt.Query(int64(altID), "", "", "patient", "hf")
		if err != nil {
			log.Println("failed to execute query statement")
			log.Println(err)
			w.WriteHeader(http.StatusOK)
			duplicateUserErr := `{"error":"cannot create a duplicate participant"}`
			w.Write([]byte(duplicateUserErr))
			return
		}
		rows.Close()

		token := capacity.CreateAccessToken("patient", "hf", int64(altID))
		tokenError := fmt.Sprintf(`{"token error":"unable to parse", "new token":"%s", "alt ID":"%d"}`, token, altID)
		w.Write([]byte(tokenError))
		return
	}

	var currentParticipant participant.Participant
	if claims, ok := token.Claims.(*capacity.NonAdminClaims); ok && token.Valid {
		currentParticipant.Study = claims.Study
		currentParticipant.ID = claims.ID
	} else {
		log.Println("token not valid")
		http.Error(w, "Invalid token type", http.StatusUnauthorized)
		return
	}

	bucket := "awsS3Bucket"
	fullUpload := true

	err = r.ParseMultipartForm(defaultMaxMemory)
	if err != nil {
		partialKey := SetPartialKey(currentParticipant, startOfWeekMillis)
		log.Printf("{Error: %s, Key: %s}\n", err.Error(), partialKey)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//get a ref to the parsed multipart form
	m := r.MultipartForm

	//get the *fileheaders
	files := m.File["upload"]
	for i, f := range files {
		//for each fileheader, get a handle to the actual file
		filename := files[i].Filename
		key := setKey(currentParticipant, startOfWeekMillis, filename)
		file, err := f.Open()
		defer file.Close()
		if err != nil {
			log.Printf("{Error: %s, Key: %s}\n", err.Error(), key)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		uploadResult, err := uh.Svc.PutObject(&s3.PutObjectInput{
			Bucket: &bucket,
			Key:    &key,
			Body:   file,
		})
		if err != nil {
			fullUpload = false
			log.Printf("Failed to upload data to %s/%s, %s\n", bucket, key, err.Error())
		}

		if !fullUpload {
			log.Printf("This is the result of the upload: %s\n{Key: %s, Success: partial}\n", uploadResult.GoString(), key)
			w.Write([]byte(partialSucess))
		} else {
			log.Printf("This is the result of the upload: %s\n{Key: %s, Success: full}\n", uploadResult.GoString(), key)
			w.Write([]byte("{\"success\": \"you have completed upload to awsS3Bucket/hf\"}"))
		}
	}
}

func HandleDataTransferToS3Bucket(w http.ResponseWriter, req *http.Request, newParticipant participant.Participant) (upload string) {
	if check.IsMultipart(req) {
		//get the *fileheaders
		req.ParseMultipartForm(0)
		defer req.MultipartForm.RemoveAll()
		file, info, err := req.FormFile("upload")
		if err != nil {
			log.Fatalf("Error while parsing multipart form for file: %s\n", err.Error())
		}
		fmt.Println("parsing multipart was ok")
		defer file.Close()
		fmt.Printf("Recieved the file: %v\n", info.Filename)

		bucket := "awsS3Bucket"
		fullUpload := true
		//for each fileheader, get a handle to the actual file
		filename := info.Filename

		key := setKey(newParticipant, "Consent & Demographic Questionnaire", filename)
		if err != nil {
			log.Printf("{Error: %s, Key: %s}\n", err.Error(), key)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		svc := s3.New(session.New(&aws.Config{Region: aws.String("us-east-1")}))

		uploadResult, err := svc.PutObject(&s3.PutObjectInput{
			Bucket: &bucket,
			Key:    &key,
			Body:   file,
		})
		if err != nil {
			fullUpload = false
			log.Printf("Failed to upload data to %s/%s, %s\n", bucket, key, err.Error())
		}

		if !fullUpload {
			log.Printf("This is the result of the upload: %s\n{Key: %s, Success: partial}\n", uploadResult.GoString(), key)
			return "partial"
		} else {
			log.Printf("This is the result of the upload: %s\n{Key: %s, Success: full}\n", uploadResult.GoString(), key)
			return "success"
		}
	} else {
		fmt.Println("Failed Request of file sending")
		w.Write([]byte("Please send files as multipart/form-data"))
		return "failed"
	}
}

const errorResJSON = `{"error":"json parsing error","error description":"key or value of json is formatted incorrectly"}`

func SetPartialKey(currentParticipant participant.Participant, startOfWeekMillis string) string {
	var partialKey string
	switch database.ADB.Environment {
	case "dev":
		partialKey = "dev/" + currentParticipant.Study + "/" + strconv.FormatInt(currentParticipant.ID, 10) + "/" + startOfWeekMillis
	case "local":
		partialKey = "test/" + currentParticipant.Study + "/" + strconv.FormatInt(currentParticipant.ID, 10) + "/" + startOfWeekMillis
	default:
		partialKey = currentParticipant.Study + "/" + strconv.FormatInt(currentParticipant.ID, 10) + "/" + startOfWeekMillis
	}

	return partialKey
}
