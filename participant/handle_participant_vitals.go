package participant

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/cliffordlab/amoss_services/database"
	"github.com/gorilla/mux"
)

const (
	defaultMaxMemory                 = 32 << 20
	selectParticipants               = `SELECT participant_id, Count(*) FROM bp_readings WHERE is_verified=FALSE GROUP BY participant_id;`
	selectParticipantVitals          = `SELECT created_at FROM bp_readings WHERE participant_id=$1 AND is_verified=FALSE;`
	selectParticipantVitalUploadData = `SELECT created_at, participant_id, systolic_bp, diastolic_bp, pulse, csv_s3_key, jpg_s3_key, s3_presigned_url, is_verified FROM bp_readings WHERE participant_id=$1 AND created_at=$2 AND is_verified=false;`
	updates3Key                      = `UPDATE bp_readings SET (s3_key, is_verified) VALUES ($1, true) where participant_id=$2 and created_at=$3;`
	updateIsVerified                 = `UPDATE bp_readings SET is_verified=true WHERE jpg_s3_key=$1 AND participant_id=$2 AND created_at=$3;`
	//updateParticipantVitalFile  = `UPDATE bp_readings SET s3_presigned_url=$3 where participant_id=$1 and created_at=$2 returning *;`
	selectParticipantJPGS3Key  = `SELECT jpg_s3_key FROM bp_readings WHERE participant_id=$1 AND created_at=$2`
	selectParticipantCSVS3Key  = `SELECT csv_s3_key FROM bp_readings WHERE participant_id=$1 AND created_at=$2`
	insertS3PresignedURL       = `UPDATE bp_readings SET s3_presigned_url = $1 WHERE participant_id=$2 AND created_at=$3 AND jpg_s3_key=$4`
	updateParticipantVitalFile = `UPDATE bp_readings SET systolic_bp=$1, diastolic_bp=$2, pulse=$3, is_verified=true WHERE participant_id=$4 AND csv_s3_key=$5;`
)

type ListParticipantsHandler struct {
	Name string
}

type VitalChartHandler struct {
	Name string
}

type ListUnverifiedFilesHandler struct {
	Name string
}

type UnverifiedBPFileHandler struct {
	Name string
	Svc  *s3.S3
}

func (l ListParticipantsHandler) ServeHTTP(writer http.ResponseWriter, _ *http.Request) {
	log.Println("Listing all distinct participants")

	stmt, err := database.ADB.Db.Prepare(selectParticipants)
	if err != nil {
		log.Println("failed to prepare create participant statement")
		log.Fatalln(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		log.Println("failed to execute query statement")
		log.Fatalln(err)
	}
	results := database.PgToJSON(rows)
	println("Participant List Json:")
	println(results)
	writer.WriteHeader(http.StatusOK)
	writer.Write(results)
}

func (v VitalChartHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	//id, err := strconv.ParseInt(strings.TrimPrefix(request.URL.Path, "/id: /"), 10, 64)
	//if err == nil {
	//	fmt.Printf("%d of type %T")
	//}
}

func (l ListUnverifiedFilesHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	log.Println("Listing participant's unverified file uploads...")
	params := mux.Vars(request)
	id := params["participant_id"]
	stmt, err := database.ADB.Db.Prepare(selectParticipantVitals)
	if err != nil {
		log.Println("failed to prepare select vitals statement")
		log.Fatalln(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(id)
	if err != nil {
		log.Println("failed to execute query statement")
		log.Fatalln(err)
	}
	result := database.PgToJSON(rows)
	writer.WriteHeader(http.StatusOK)
	writer.Write(result)
}

func (u UnverifiedBPFileHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	method := request.Method
	params := mux.Vars(request)
	id := params["participant_id"]
	creationTime := params["created_at"]
	log.Println("This is id: " + id + " Thiis is the time: " + creationTime)
	switch method {
	case "GET":
		log.Println("GET request:")
		insertS3PresignedURLtoDB(id, creationTime, u)
		getParticipantVitalData(writer, id, creationTime)
	case "PUT":
		log.Println("PUT request:")
		updateVitalIsVerified(id, creationTime, writer)
	case "POST":
		log.Println("POST request:")
		updateParticipantVitals(id, creationTime, writer, request, u)
	}
}

func updateVitalIsVerified(id string, creationTime string, writer http.ResponseWriter) {
	// update db
	s3Key := getS3Key(id, creationTime, "jpg")
	updateDBIsVerified(s3Key, id, creationTime, writer)
}

func updateDBIsVerified(key string, id string, creationTime string, writer http.ResponseWriter) {
	println("Updating database vital verification...")

	stmt, err := database.ADB.Db.Prepare(updateIsVerified)
	if err != nil {
		log.Println("failed to prepare update participant's vital statements")
		log.Fatalln(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(key, id, creationTime)
	if err != nil {
		log.Println("failed to execute query statement")
		log.Fatalln(err)
	}
	rows.Close()

	fmt.Printf("Successfully updated db for: ", key)

	writer.WriteHeader(http.StatusOK)
	writer.Write([]byte("{\"success\": \"you have completed update to awsS3Bucket/moyo-mom-emory/\"}"))
}

func insertS3PresignedURLtoDB(id string, creationTime string, u UnverifiedBPFileHandler) {
	s3Key := getS3Key(id, creationTime, "jpg")
	url := GetS3PreSignedUrl(s3Key, u)
	insertURLintoDB(url, id, creationTime, s3Key)
}

func getS3Key(id string, creationTime string, fileType string) string {
	log.Println("Querying db for participant vital file s3 key..")
	var queryStatement string
	switch fileType {
	case "jpg":
		queryStatement = selectParticipantJPGS3Key
	case "csv":
		queryStatement = selectParticipantCSVS3Key
	}

	stmt, err := database.ADB.Db.Prepare(queryStatement)
	if err != nil {
		log.Println("failed to prepare select participant's vital statements")
		log.Fatalln(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(id, creationTime)
	if err != nil {
		log.Println("failed to execute query statement")
		log.Fatalln(err)
	}
	var s3Key []byte
	for rows.Next() {
		err = rows.Scan(&s3Key)
		if err != nil {
			log.Println("failed to scan row for s3 key")
			log.Println(err)
		}
	}
	rows.Close()
	s3KeyString := string(s3Key)
	println("this is the s3 key: " + s3KeyString)
	return s3KeyString
}

func insertURLintoDB(url string, id string, creationTime string, key string) {
	log.Println("Inserting s3 presigned URL into db..")

	log.Println("Here is the presigned URL:")
	log.Println(url)
	log.Println("Here is the id:")
	log.Println(id)
	log.Println("Here is the creationTime:")
	log.Println(creationTime)
	log.Println("Here is the key:")
	log.Println(key)

	stmt, err := database.ADB.Db.Prepare(insertS3PresignedURL)
	if err != nil {
		log.Println("failed to prepare insert url statement")
		log.Fatalln(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(url, id, creationTime, key)
	if err != nil {
		log.Println("failed to execute query statement")
		log.Fatalln(err)
	}
	log.Println("S3 presigned URL inserted successfully into db.")
	rows.Close()
}

func getParticipantVitalData(writer http.ResponseWriter, id string, creationTime string) {
	log.Println("Querying db for participant's file upload data...")

	stmt, err := database.ADB.Db.Prepare(selectParticipantVitalUploadData)
	if err != nil {
		log.Println("failed to prepare select participant's vital statements")
		log.Fatalln(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(id, creationTime)
	if err != nil {
		log.Println("failed to execute query statement")
		log.Fatalln(err)
	}
	result := database.PgToJSON(rows)
	writer.WriteHeader(http.StatusOK)
	writer.Write(result)
}

func updateParticipantVitals(id string, creationTime string, writer http.ResponseWriter, request *http.Request, u UnverifiedBPFileHandler) {
	log.Println("Updating participant vitals...")
	var vr VitalsRequest
	err := request.ParseMultipartForm(defaultMaxMemory)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		log.Println("This is the error: ")
		log.Println(err)
		return
	}
	m := request.Form

	vr.SBP, _ = strconv.Atoi(m.Get("sbp"))
	vr.DBP, _ = strconv.Atoi(m.Get("dbp"))
	vr.Pulse, _ = strconv.Atoi(m.Get("pulse"))
	log.Println("request.Form.Get(sbp): " + request.Form.Get("sbp"))
	log.Println("dbp: " + request.Form.Get("dbp"))
	log.Println("pulse: " + request.Form.Get("pulse"))

	log.Println("strconv.Itoa(vr.SBP: " + strconv.Itoa(vr.SBP))
	log.Println("strconv.Itoa(vr.DBP: " + strconv.Itoa(vr.DBP))
	log.Println("strconv.Itoa(vr.Pulse: " + strconv.Itoa(vr.Pulse))

	s3Key := getS3Key(id, creationTime, "csv")
	// s3 copy original file and name pid_timestamp.file_old
	renameS3Object(u, s3Key)
	// write new file with approved values
	uploadNewFile(u, s3Key, vr)
	// update db
	updateDB(s3Key, vr, id)
	writer.WriteHeader(http.StatusOK)
	writer.Write([]byte("{\"success\": \"you have completed update to awsS3Bucket/moyo-mom-emory/\"}"))
}

func updateDB(s3Key string, vr VitalsRequest, id string) {
	println("Updating database with verified bp values.")

	stmt, err := database.ADB.Db.Prepare(updateParticipantVitalFile)
	if err != nil {
		log.Println("failed to prepare update participant's vital statements")
		log.Fatalln(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(vr.SBP, vr.DBP, vr.Pulse, id, s3Key)
	if err != nil {
		log.Println("failed to execute query statement")
		log.Fatalln(err)
	}
	rows.Close()

	fmt.Printf("Successfully update db for: ", s3Key)

}

type VitalsRequest struct {
	SBP   int
	DBP   int
	Pulse int
}

func uploadNewFile(u UnverifiedBPFileHandler, s3Key string, vr VitalsRequest) {
	log.Println("Creating new BP CSV File...")
	// init byte buffer var
	var bb bytes.Buffer
	bb.Write([]byte("SBP: " + strconv.Itoa(vr.SBP) + ", "))
	bb.Write([]byte("DBP: " + strconv.Itoa(vr.DBP) + ", "))
	bb.Write([]byte("Pulse: " + strconv.Itoa(vr.Pulse) + ", "))
	bucket := "awsS3Bucket"
	key := s3Key
	reader := bytes.NewReader(bb.Bytes())
	log.Println("Uploading new csv File... ")
	uploadResult, err := u.Svc.PutObject(&s3.PutObjectInput{
		Body:   reader,
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		log.Fatalf("Failed to upload data to %s/%s, %s\n", bucket, key, err.Error())
		return
	}
	log.Printf("This is the result of the upload with key %s: %s\n", key, uploadResult.GoString())
}

func renameS3Object(u UnverifiedBPFileHandler, s3Key string) {
	log.Println("Renaming S3 Object...")

	log.Println("Copying S3 Object...")
	// Copy the item
	_, err := u.Svc.CopyObject(&s3.CopyObjectInput{Bucket: aws.String("awsS3Bucket"), CopySource: aws.String("awsS3Bucket/" + s3Key), Key: aws.String(s3Key + "_old")})
	if err != nil {
		fmt.Printf("Item %q copy unsuccessful: " + s3Key)
		return
	}
	fmt.Printf("Item %q successfully copied: " + s3Key)

	// Wait to see if the item got copied
	err = u.Svc.WaitUntilObjectExists(&s3.HeadObjectInput{Bucket: aws.String("awsS3Bucket"), Key: aws.String(s3Key + "_old")})
	if err != nil {
		fmt.Printf("Item %q copy unsuccessful: " + s3Key)
		return
	}
	fmt.Printf("Item %q successfully copied: " + s3Key)

	log.Println("Deleting Old S3 Object...")

	// delete original file
	_, err = u.Svc.DeleteObject(&s3.DeleteObjectInput{Bucket: aws.String("awsS3Bucket"), Key: aws.String(s3Key)})
	if err != nil {
		fmt.Printf("Item %q delete unsucessful: " + s3Key)
		return
	}
	fmt.Printf("Item %q successfully delete: " + s3Key)

	err = u.Svc.WaitUntilObjectNotExists(&s3.HeadObjectInput{Bucket: aws.String("awsS3Bucket"), Key: aws.String(s3Key)})
	if err != nil {
		fmt.Printf("Item %q delete unsucessful: " + s3Key)
		return
	}
	fmt.Printf("Item %q successfully delete: " + s3Key)

}

//func GetS3PreSignedUrl(bucket string, key string, region string, expiration time.Duration) {
func GetS3PreSignedUrl(key string, u UnverifiedBPFileHandler) string {
	expiration := time.Duration(10080)
	log.Println("Creating presigned URL...")

	req, _ := u.Svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String("awsS3Bucket"),
		//Key:    aws.String("test/7775000000/586799573906/bloodpressure.jpg"),
		Key: aws.String(key),
	})

	preSignedURL, err := req.Presign(expiration * time.Minute)
	if err != nil {
		fmt.Println("Failed to sign request", err)
	}
	log.Println("here is the presigned URL:  ")
	log.Println(preSignedURL)
	return preSignedURL
}
