package garminauth

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"github.com/cliffordlab/amoss_services/capacity"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/cliffordlab/amoss_services/database"
	"github.com/garyburd/go-oauth/oauth"
	"github.com/jinzhu/now"
)

type GarminPingHandler struct {
	Name string
	Svc  *s3.S3
}

type PingNotification struct {
	Epochs []struct {
		UserAccessToken          string `json:"userAccessToken"`
		UploadStartTimeInSeconds int    `json:"uploadStartTimeInSeconds"`
		UploadEndTimeInSeconds   int    `json:"uploadEndTimeInSeconds"`
		CallbackURL              string `json:"callbackURL"`
	} `json:"epochs"`
	Dailies []struct {
		UserAccessToken          string `json:"userAccessToken"`
		UploadStartTimeInSeconds int    `json:"uploadStartTimeInSeconds"`
		UploadEndTimeInSeconds   int    `json:"uploadEndTimeInSeconds"`
		CallbackURL              string `json:"callbackURL"`
	} `json:"dailies"`
	Activities []struct {
		UserAccessToken          string `json:"userAccessToken"`
		UploadStartTimeInSeconds int    `json:"uploadStartTimeInSeconds"`
		UploadEndTimeInSeconds   int    `json:"uploadEndTimeInSeconds"`
		CallbackURL              string `json:"callbackURL"`
	} `json:"activities"`
	Sleeps []struct {
		UserAccessToken          string `json:"userAccessToken"`
		UploadStartTimeInSeconds int    `json:"uploadStartTimeInSeconds"`
		UploadEndTimeInSeconds   int    `json:"uploadEndTimeInSeconds"`
		CallbackURL              string `json:"callbackURL"`
	} `json:"sleeps"`
	BodyComps []struct {
		UserAccessToken          string `json:"userAccessToken"`
		UploadStartTimeInSeconds int    `json:"uploadStartTimeInSeconds"`
		UploadEndTimeInSeconds   int    `json:"uploadEndTimeInSeconds"`
		CallbackURL              string `json:"callbackURL"`
	} `json:"bodyComps"`
	StressDetails []struct {
		UserAccessToken          string `json:"userAccessToken"`
		UploadStartTimeInSeconds int    `json:"uploadStartTimeInSeconds"`
		UploadEndTimeInSeconds   int    `json:"uploadEndTimeInSeconds"`
		CallbackURL              string `json:"callbackURL"`
	} `json:"stressDetails"`
	UserMetrics []struct {
		UserAccessToken          string `json:"userAccessToken"`
		UploadStartTimeInSeconds int    `json:"uploadStartTimeInSeconds"`
		UploadEndTimeInSeconds   int    `json:"uploadEndTimeInSeconds"`
		CallbackURL              string `json:"callbackURL"`
	} `json:"userMetrics"`
	MoveIQ []struct {
		UserAccessToken          string `json:"userAccessToken"`
		UploadStartTimeInSeconds int    `json:"uploadStartTimeInSeconds"`
		UploadEndTimeInSeconds   int    `json:"uploadEndTimeInSeconds"`
		CallbackURL              string `json:"callbackURL"`
	} `json:"moveIQ"`
	PulseOx []struct {
		UserAccessToken          string `json:"userAccessToken"`
		UploadStartTimeInSeconds int    `json:"uploadStartTimeInSeconds"`
		UploadEndTimeInSeconds   int    `json:"uploadEndTimeInSeconds"`
		CallbackURL              string `json:"callbackURL"`
	} `json:"pulseOx"`
}

func (gph GarminPingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	w.WriteHeader(http.StatusOK)

	dec := json.NewDecoder(r.Body)
	var pings PingNotification
	if err := dec.Decode(&pings); err != nil {
		log.Println(err)
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Write([]byte(errorResJSON))
		return
	}

	for _, element := range pings.Epochs {
		var token = element.UserAccessToken
		var startTime = element.UploadStartTimeInSeconds
		var endTime = element.UploadEndTimeInSeconds
		var callBackURL = element.CallbackURL
		var summaryName = "_epochs.garmin"
		uploadData(token, startTime, endTime, callBackURL, gph, summaryName)
	}

	for _, element := range pings.Dailies {
		var token = element.UserAccessToken
		var startTime = element.UploadStartTimeInSeconds
		var endTime = element.UploadEndTimeInSeconds
		var callBackURL = element.CallbackURL
		var summaryName = "_dailies.garmin"
		uploadData(token, startTime, endTime, callBackURL, gph, summaryName)
	}

	for _, element := range pings.Activities {
		var token = element.UserAccessToken
		var startTime = element.UploadStartTimeInSeconds
		var endTime = element.UploadEndTimeInSeconds
		var callBackURL = element.CallbackURL
		var summaryName = "_activities.garmin"
		uploadData(token, startTime, endTime, callBackURL, gph, summaryName)
	}

	for _, element := range pings.Sleeps {
		var token = element.UserAccessToken
		var startTime = element.UploadStartTimeInSeconds
		var endTime = element.UploadEndTimeInSeconds
		var callBackURL = element.CallbackURL
		var summaryName = "_sleeps.garmin"
		uploadData(token, startTime, endTime, callBackURL, gph, summaryName)
	}

	for _, element := range pings.BodyComps {
		var token = element.UserAccessToken
		var startTime = element.UploadStartTimeInSeconds
		var endTime = element.UploadEndTimeInSeconds
		var callBackURL = element.CallbackURL
		var summaryName = "_bodycomps.garmin"
		uploadData(token, startTime, endTime, callBackURL, gph, summaryName)
	}

	for _, element := range pings.StressDetails {
		var token = element.UserAccessToken
		var startTime = element.UploadStartTimeInSeconds
		var endTime = element.UploadEndTimeInSeconds
		var callBackURL = element.CallbackURL
		var summaryName = "_stress.garmin"
		uploadData(token, startTime, endTime, callBackURL, gph, summaryName)
	}

	for _, element := range pings.UserMetrics {
		var token = element.UserAccessToken
		var startTime = element.UploadStartTimeInSeconds
		var endTime = element.UploadEndTimeInSeconds
		var callBackURL = element.CallbackURL
		var summaryName = "_usermetrics.garmin"
		uploadData(token, startTime, endTime, callBackURL, gph, summaryName)
	}

	for _, element := range pings.MoveIQ {
		var token = element.UserAccessToken
		var startTime = element.UploadStartTimeInSeconds
		var endTime = element.UploadEndTimeInSeconds
		var callBackURL = element.CallbackURL
		var summaryName = "_moveiq.garmin"
		uploadData(token, startTime, endTime, callBackURL, gph, summaryName)
	}

	for _, element := range pings.PulseOx {
		var token = element.UserAccessToken
		var startTime = element.UploadStartTimeInSeconds
		var endTime = element.UploadEndTimeInSeconds
		var callBackURL = element.CallbackURL
		var summaryName = "_pulseox.garmin"
		uploadData(token, startTime, endTime, callBackURL, gph, summaryName)
	}
}

func uploadData(token string, startTime int, endTime int, callBackURL string, gph GarminPingHandler, summaryName string) {
	log.Println("Uploading Garmin " + summaryName + " data...")

	rows, err := database.ADB.Db.Query(`SELECT participant_id, study_id, garmin_secret_token FROM participants WHERE garmin_access_token=$1;`, token)
	if err != nil {
		log.Println("Database query for token failed!")
		log.Fatal(err)
	}
	defer rows.Close()
	var ptid int
	var study sql.NullString
	var secretToken sql.NullString

	for rows.Next() {
		err := rows.Scan(&ptid, &study, &secretToken)
		if err != nil {
			log.Println("Line 196. Scan rows failed!")
			log.Println(err)
			return
		}
	}
	defer rows.Close()

	// check if access token was connected with a user
	// if not end function
	if study.String == "" || secretToken.String == "" {
		log.Println("This access token is not in the database")
		return
	}
	req, err := http.NewRequest("GET", callBackURL, nil)
	if err != nil {
		log.Println("Line 211. Get request to " + callBackURL + " failed!")
		log.Fatal(err)
		return
	}
	client := oauth.Client{}
	query := make(url.Values)
	query.Add("userAccessToken", token)
	query.Add("secretToken", secretToken.String)
	if err := client.SetAuthorizationHeader(req.Header, &oauth.Credentials{
		Token:  capacity.GarminToken,
		Secret: capacity.GarminSecret}, "GET", req.URL, nil, "", token, secretToken.String); err != nil {
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("Line 226. Get request to " + callBackURL + " failed: ")
		log.Fatal(err)
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	var garminFileBuffer bytes.Buffer
	garminFileBuffer.Write(bodyBytes)
	var fileExtName = strings.ToLower(summaryName)

	log.Println("Sending garmin file to S3...")

	//send garmin file to s3
	bucket := "amoss-mhealth"
	uploadStartTime := strings.TrimPrefix(strconv.Itoa(startTime), "1")
	filename := uploadStartTime + fileExtName
	//filename := uploadStartTime + ".garmin"

	now.WeekStartDay = time.Monday // Set Monday as first day, default is Sunday
	startWeekMillis := now.New(time.Now()).BeginningOfWeek().UnixNano() / int64(time.Millisecond)
	monString := strconv.Itoa(int(startWeekMillis))
	trimmedMonString := strings.Replace(monString, "1", "", 1)

	ptidString := strconv.Itoa(ptid)

	var key string
	switch database.ADB.Environment {
	case "dev":
		key = "dev/" + study.String + "/" + ptidString + "/" + trimmedMonString + "/" + filename
	case "local":
		key = "test/" + study.String + "/" + ptidString + "/" + trimmedMonString + "/" + filename
	default:
		key = study.String + "/" + ptidString + "/" + trimmedMonString + "/" + filename
	}

	reader := bytes.NewReader(garminFileBuffer.Bytes())
	uploadResult, err := gph.Svc.PutObject(&s3.PutObjectInput{
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
