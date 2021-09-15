package emory

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/cliffordlab/amoss_services/capacity"
	"github.com/cliffordlab/amoss_services/database"
	"github.com/cliffordlab/amoss_services/participant"
	"github.com/cliffordlab/amoss_services/support/moyo_mom_emory"
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
	insertSymptomsData = `INSERT INTO mme_symptoms 
(created_at, participant_id, blurried_vision, headache, difficulty_breathing, side_pain) 
VALUES($1, $2, $3, $4, $5, $6)`
)

type UploadMMEVitalsHandler struct {
	Name string
	Svc  *s3.S3
}

type UploadMMESymptomsHandler struct {
	Name string
	Svc  *s3.S3
}

type ParticipantSymptomsRequest struct {
	BV        bool
	HA        bool
	DB        bool
	SP        bool
	CreatedAt int64
}

func (u UploadMMESymptomsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling symptom data upload to Moyo Mom - Emory..")
	contentType := r.Header.Get("Content-type")
	log.Println("This is the Content-Type: ")
	log.Println(contentType)

	err := r.ParseMultipartForm(defaultMaxMemory)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println("This is the error: ")
		log.Println(err)
		return
	}
	m := r.Form

	var psr ParticipantSymptomsRequest

	psr.BV, _ = strconv.ParseBool(m.Get("blurried_vision"))
	psr.HA, _ = strconv.ParseBool(m.Get("headache"))
	psr.DB, _ = strconv.ParseBool(m.Get("difficulty_breathing"))
	psr.SP, _ = strconv.ParseBool(m.Get("side_pain"))
	psr.CreatedAt, _ = strconv.ParseInt(m.Get("created_at"), 10, 64)

	log.Println("blurried_vision: " + strconv.FormatBool(psr.BV))
	log.Println("headache: " + strconv.FormatBool(psr.HA))
	log.Println("difficulty_breathing: " + strconv.FormatBool(psr.DB))
	log.Println("side_pain: " + strconv.FormatBool(psr.SP))

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
	checkSymptomsThreshold(psr, currentParticipant)

	//todo	query database to match header token with token in DB and continue
	// else return you are already logged in and log them out return 400 unauthorized or forbidden?
	// Bearer token or token?

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

	log.Println("This is the access token: " + accessTokenDB)
	log.Println("This is the bearer token: " + bearerToken)

	if accessTokenDB == bearerToken {
		log.Println("Access token matches Database token...")
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
			//file, err := files[i].Open()
			file, err := f.Open()
			defer file.Close()
			if err != nil {
				log.Printf("{Error: %s, Key: %s}\n", err.Error(), key)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			uploadResult, err := u.Svc.PutObject(&s3.PutObjectInput{
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
		insertSymptomsIntoDB(currentParticipant, psr)
	} else {
		log.Println("Access token does not match that of the database")
		log.Println("Participant_ID: ", currentParticipant.ID)
		//http.(w, "Invalid access token. Please log user out", http.StatusConflict)
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Write([]byte(invalidAccessToken))
		return
	}
}

func insertSymptomsIntoDB(currentParticipant participant.Participant, psr ParticipantSymptomsRequest) {
	log.Println("Inserting symptoms into DB..")

	stmt, err := database.ADB.Db.Prepare(insertSymptomsData)
	if err != nil {
		log.Println("failed to prepare insert symptom statement")
		log.Fatalln(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(psr.CreatedAt, currentParticipant.ID, psr.BV, psr.HA, psr.DB, psr.SP)
	if err != nil {
		log.Println("failed to execute query statement")
		log.Fatalln(err)
	}
	log.Println("Symptom data inserted successfully into db.")
	rows.Close()
}

func checkSymptomsThreshold(psr ParticipantSymptomsRequest, currentParticipant participant.Participant) {
	log.Println("Checking symptoms thresholds... ")
	//systolic BP >160 mm Hg or diastolic BP>110 mm Hg
	patientID := currentParticipant.ID
	patientIDString := strconv.FormatInt(patientID, 10)

	bvString := strconv.FormatBool(psr.BV)
	haString := strconv.FormatBool(psr.HA)
	dbString := strconv.FormatBool(psr.DB)
	spString := strconv.FormatBool(psr.SP)

	if psr.BV == true || psr.HA == true || psr.DB == true || psr.SP == true {
		log.Println("Symptoms threshold reached. Attempting to send email...")
		moyo_mom_emory.SendSymptomsEmail(aws.String("\n Symptom alert for participant: " + patientIDString + "\n Blurried vision: " + bvString + "\n Head ache: " + haString + "\n Difficulty Breathing: " + dbString + "\n Side Pain: " + spString))
	}
}

type ParticipantVitalsRequest struct {
	SBP       int
	DBP       int
	Pulse     int
	CreatedAt int64
}

func (uh UploadMMEVitalsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling vital data upload to Moyo Mom - Emory..")
	contentType := r.Header.Get("Content-type")
	log.Println("This is the Content-Type: ")
	log.Println(contentType)

	var pvr ParticipantVitalsRequest
	err := r.ParseMultipartForm(defaultMaxMemory)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println("This is the error: ")
		log.Println(err)
		return
	}
	m := r.Form

	pvr.SBP, _ = strconv.Atoi(m.Get("sbp"))
	pvr.DBP, _ = strconv.Atoi(m.Get("dbp"))
	pvr.Pulse, _ = strconv.Atoi(m.Get("pulse"))
	pvr.CreatedAt, _ = strconv.ParseInt(m.Get("created_at"), 10, 64)

	log.Println("SBP: " + strconv.Itoa(pvr.SBP))
	log.Println("DBP" + strconv.Itoa(pvr.DBP))
	log.Println("Pulse" + strconv.Itoa(pvr.Pulse))

	startOfWeekMillis := r.Header.Get("weekMillis")
	if len(startOfWeekMillis) != 12 {
		log.Println("token length is wrong")
		w.Header().Add("Content-Type", "application/json; charset=UTF-8")
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
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

	checkThreshold(pvr, currentParticipant)

	//todo	query database to match header token with token in DB and continue
	// else return you are already logged in and log them out return 400 unauthorized or forbidden?
	// Bearer token or token?

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

	log.Println("This is the access token: " + accessTokenDB)
	log.Println("This is the bearer token: " + bearerToken)

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
		csvFilename := strconv.FormatInt(currentParticipant.ID, 10) + "_" + strconv.FormatInt(pvr.CreatedAt, 10) + "_bp.csv"

		csvS3Key := setKey(currentParticipant, startOfWeekMillis, csvFilename)
		fullUpload = uh.uploadCSV(csvFilename, pvr, bucket, fullUpload, csvS3Key)
		fullUpload, done := uh.uploadJPEG(w, r, currentParticipant, startOfWeekMillis, bucket, fullUpload)
		if done {
			return
		}
		if !fullUpload {
			w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			w.Write([]byte(partialSucess))
		} else {
			w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			w.Write([]byte("{\"success\": \"you have completed upload to awsS3Bucket/moyo-mom-emory/\"}"))
		}
		insertVitalsToDB(currentParticipant, mMEJPEGS3Key, csvS3Key, pvr)

	} else {
		log.Println("Access token does not match that of the database")
		log.Println("Participant_ID: ", currentParticipant.ID)
		//http.(w, "Invalid access token. Please log user out", http.StatusConflict)
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Write([]byte(invalidAccessToken))
		return
	}
}

var mMEJPEGS3Key string

func checkThreshold(pvr ParticipantVitalsRequest, currentParticipant participant.Participant) {
	log.Println("Checking vital thresholds... ")
	//systolic BP >160 mm Hg or diastolic BP>110 mm Hg
	patientID := currentParticipant.ID
	patientIDString := strconv.FormatInt(patientID, 10)

	sbpString := strconv.Itoa(pvr.SBP)
	dbpString := strconv.Itoa(pvr.DBP)
	pulseString := strconv.Itoa(pvr.Pulse)

	if pvr.SBP > 160 || pvr.DBP > 110 {
		log.Println("Vital threshold reached. Attempting to send email...")
		moyo_mom_emory.SendVitalEmail(aws.String("Threshold reached for participant: " + patientIDString + " SDP: " + sbpString + " DBP: " + dbpString + " Pulse: " + pulseString))
	}

}

func (uh UploadMMEVitalsHandler) uploadJPEG(w http.ResponseWriter, r *http.Request, currentParticipant participant.Participant, startOfWeekMillis string, bucket string, fullUpload bool) (bool, bool) {
	log.Println("Uploading JPEG to S3...")
	m := r.MultipartForm

	//get the *fileheaders
	files := m.File["upload"]
	for i, f := range files {
		//for each fileheader, get a handle to the actual file
		filename := files[i].Filename
		mMEJPEGS3Key = setKey(currentParticipant, startOfWeekMillis, filename)
		log.Println("this is the key: ")
		log.Println(mMEJPEGS3Key)

		//file, err := files[i].Open()
		file, err := f.Open()
		defer file.Close()
		if err != nil {
			log.Printf("{Error: %s, Key: %s}\n", err.Error(), mMEJPEGS3Key)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return false, true
		}

		uploadResult, err := uh.Svc.PutObject(&s3.PutObjectInput{
			Bucket: &bucket,
			Key:    &mMEJPEGS3Key,
			Body:   file,
		})
		if err != nil {
			fullUpload = false
			log.Printf("Failed to upload data to %s/%s, %s\n", bucket, mMEJPEGS3Key, err.Error())
		}

		if !fullUpload {
			log.Printf("This is the result of the upload: %s\n{Key: %s, Success: partial}\n", uploadResult.GoString(), mMEJPEGS3Key)
		} else {
			log.Printf("This is the result of the upload: %s\n{Key: %s, Success: full}\n", uploadResult.GoString(), mMEJPEGS3Key)
		}
	}
	return fullUpload, false
}

func (uh UploadMMEVitalsHandler) uploadCSV(csvFilename string, pvr ParticipantVitalsRequest, bucket string, fullUpload bool, s3key string) bool {
	log.Println("Writing new CSV file to upload to ...")
	log.Println("SBP: " + strconv.Itoa(pvr.SBP))
	log.Println("DBP" + strconv.Itoa(pvr.DBP))
	log.Println("Pulse" + strconv.Itoa(pvr.Pulse))
	// init byte buffer var
	var bb bytes.Buffer
	bb.Write([]byte("SBP: " + strconv.Itoa(pvr.SBP) + ", "))
	bb.Write([]byte("DBP: " + strconv.Itoa(pvr.DBP) + ", "))
	bb.Write([]byte("Pulse: " + strconv.Itoa(pvr.Pulse) + ", "))
	reader := bytes.NewReader(bb.Bytes())
	log.Println("Uploading new csv File... ")
	uploadResult, err := uh.Svc.PutObject(&s3.PutObjectInput{
		Body:   reader,
		Bucket: &bucket,
		Key:    &s3key,
	})

	//file, err := os.Create(csvFilename)
	//if err != nil {
	//	log.Fatalf("failed creating file: %s", err)
	//}
	//defer file.Close()
	//
	//writer := csv.NewWriter(file)
	//defer writer.Flush()
	//bpData := [][]string{
	//	{"sbp", "dbp", "pulse"},
	//	{strconv.Itoa(pvr.SBP), strconv.Itoa(pvr.DBP), strconv.Itoa(pvr.Pulse)},
	//}
	//
	//for _, value := range bpData {
	//	err := writer.Write(value)
	//	if err != nil {
	//		log.Fatalf("failed writing to file: %s", err)
	//	}
	//}
	//uploadResult, err := uh.Svc.PutObject(&s3.PutObjectInput{
	//	Bucket: &bucket,
	//	Key:    &s3key,
	//	Body:   file,
	//})
	if err != nil {
		fullUpload = false
		log.Printf("Failed to upload data to %s/%s, %s\n", bucket, s3key, err.Error())
	}

	if !fullUpload {
		log.Printf("This is the result of the upload: %s\n{Key: %s, Success: partial}\n", uploadResult.GoString(), s3key)
	} else {
		// Removing file from the directory
		// Using Remove() function
		//e := os.Remove(csvFilename)
		//if e != nil {
		//	log.Fatal(e)
		//}
		log.Printf("This is the result of the upload: %s\n{Key: %s, Success: full}\n", uploadResult.GoString(), s3key)
	}
	return fullUpload
}

func insertVitalsToDB(currentParticipant participant.Participant, jpgS3Key string, csvS3Key string, pvr ParticipantVitalsRequest) {
	log.Println("Inserting s3Key into DB..")

	log.Println("pvr.SBP: " + strconv.Itoa(pvr.SBP))
	log.Println("pvr.DBP: " + strconv.Itoa(pvr.DBP))
	log.Println("pvr.Pulse: " + strconv.Itoa(pvr.Pulse))

	stmt, err := database.ADB.Db.Prepare(insertVitalsData)
	if err != nil {
		log.Println("failed to prepare insert s3 key statement")
		log.Fatalln(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(pvr.CreatedAt, currentParticipant.ID, pvr.SBP, pvr.DBP, pvr.Pulse, jpgS3Key, csvS3Key)
	if err != nil {
		log.Println("failed to execute query statement")
		log.Fatalln(err)
	}
	log.Println("S3 Key inserted successfully into db.")
	rows.Close()
}

func SetPartialKey(currentParticipant participant.Participant, startOfWeekMillis string) string {
	var partialKey string
	partialKey = "test/" + currentParticipant.Study + "/" + strconv.FormatInt(currentParticipant.ID, 10) + "/" + startOfWeekMillis

	return partialKey
}

func setKey(currentParticipant participant.Participant, startOfWeekMillis string, filename string) string {
	var key string
	key = "test/" + currentParticipant.Study + "/" + strconv.FormatInt(currentParticipant.ID, 10) + "/" + startOfWeekMillis + "/" + filename

	return key
}
