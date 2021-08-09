package jawbone

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/cliffordlab/amoss_services/database"
	"github.com/jinzhu/now"
)

// PutJawboneFileInS3 sends jawbone files to s3
func PutJawboneFileInS3(client *http.Client) {
	//start a new aws session
	svc := s3.New(session.New(&aws.Config{Region: aws.String("us-east-1")}))
	layout := "2006-01-02T15:04:05Z"
	//loop through jawbone users in database
	jbUsers := database.FindJawboneUsers()
	for _, user := range jbUsers {
		if user.JawboneDate == "" {
			continue
		}
		currentTime := time.Now().Unix()
		jbDate, err := time.Parse(layout, user.JawboneDate)
		if err != nil {
			fmt.Println(err)
		}
		for currentTime > jbDate.Unix() {
			var jawboneFileBuffer bytes.Buffer
			if user.AccessToken != "" {
				formattedDate := jbDate.Format("20060102")

				var xids []string
				sleeps, statusOk := getSleeps(client, user.AccessToken, getSleepsXidsURL(formattedDate))
				if user.Study == "sleepBank" {
					log.Println("This study is sleep bank")
					log.Printf("Status OK: %t\n", statusOk)
					log.Print("This is the data: ")
					log.Println(string(sleeps))
				}
				if statusOk {
					sleepRes := &SleepResults{}
					err := json.Unmarshal([]byte(sleeps), sleepRes)
					if err != nil {
						log.Fatal(err)
					}
					if sleepRes.Meta.Code == 200 {
						if sleepRes.Data.Size > 0 {
							jawboneFileBuffer.Write(sleeps)
						}
					}

					for i := 0; i < sleepRes.Data.Size; i++ {
						xids = append(xids, sleepRes.Data.Items[i].XID)
					}
				}

				for i := 0; i < len(xids); i++ {
					sleepPhases, statusOk := getSleepPhases(client, user.AccessToken, xids[i])
					if user.Study == "sleepBank" {
						log.Println("This study is sleep bank")
						log.Printf("This is the status code: %t\n", statusOk)
						log.Print("This is the data with xids: ")
						log.Println(sleepPhases)
					}
					if statusOk {
						sleepRes := &SleepResults{}
						err := json.Unmarshal([]byte(sleepPhases), sleepRes)
						if err != nil {
							log.Fatal(err)
						}
						if sleepRes.Meta.Code == 200 {
							if sleepRes.Data.Size > 0 {
								jawboneFileBuffer.Write(sleepPhases)
							}
						}
					}
				}

				var movesXids []string
				moves, statusOk := getMoves(client, user.AccessToken, getMovesXidsURL(formattedDate))
				if statusOk {
					movesRes := &MoveResults{}
					err := json.Unmarshal([]byte(moves), movesRes)
					if err != nil {
						log.Fatal(err)
					}
					if movesRes.Meta.Code == 200 {
						if movesRes.Data.Size > 0 {
							jawboneFileBuffer.Write(moves)
						}
					}

					for i := 0; i < movesRes.Data.Size; i++ {
						movesXids = append(movesXids, movesRes.Data.Items[i].XID)
					}
				}

				for i := 0; i < len(xids); i++ {
					moveTicks, statusOk := getMoveTicks(client, user.AccessToken, xids[i])
					if statusOk {
						movesRes := &MoveResults{}
						err := json.Unmarshal([]byte(moveTicks), movesRes)
						if err != nil {
							log.Fatal(err)
						}
						if movesRes.Meta.Code == 200 {
							if movesRes.Data.Size > 0 {
								jawboneFileBuffer.Write(moveTicks)
							}
						}
					}
				}

				heartRate, statusOk := getHeartRate(client, user.AccessToken, getHeartRateURL(formattedDate))
				if statusOk {
					heartRateRes := &HeartRateResults{}
					err := json.Unmarshal([]byte(heartRate), heartRateRes)
					if err != nil {
						log.Fatal(err)
					}
					if heartRateRes.Meta.Code == 200 {
						if heartRateRes.Data.Size > 0 {
							jawboneFileBuffer.Write(heartRate)
						}
					}
				}

				if user.Study == "sleepBank" {
					log.Println("This is jawbone file buffer for sleep bank: ")
					log.Println(bytes.NewReader(jawboneFileBuffer.Bytes()))
				}

				if len(jawboneFileBuffer.Bytes()) > 0 {
					//send jawbone file to s3
					bucket := "amoss-mhealth"
					filename := formattedDate + ".jb"

					now.FirstDayMonday = true // Set Monday as first day, default is Sunday
					startWeekMillis := now.New(jbDate).BeginningOfWeek().UnixNano() / int64(time.Millisecond)
					monString := strconv.Itoa(int(startWeekMillis))
					trimmedMonString := strings.Replace(monString, "1", "", 1)

					ptidString := strconv.Itoa(user.ParticipantID)

					key := user.Study + "/" + ptidString + "/" + trimmedMonString + "/" + filename

					reader := bytes.NewReader(jawboneFileBuffer.Bytes())
					uploadResult, err := svc.PutObject(&s3.PutObjectInput{
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
				//increment date by 1 day
				jbDate = jbDate.AddDate(0, 0, 1)

				//update date in postgres
				stmt, err := database.ADB.Db.Prepare(`UPDATE wearables SET jawbone_date = $1 WHERE wearable_id = $2;`)
				if err != nil {
					log.Println("failed to prepare date update statement")
					log.Fatalln(err)
				}
				defer stmt.Close()
				postgresDate := jbDate.Format("2006-01-02")
				rows, err := stmt.Query(postgresDate, user.WearableID)
				if err != nil {
					log.Println("failed to execute date update query statement")
					log.Println(err)
					return
				}
				rows.Close()
			}
		}
	}
}
