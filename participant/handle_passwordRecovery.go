/******************************************************************************
Password Recovery

  "participantID": id,
  "password": string

******************************************************************************/

package participant

import (
	"encoding/json"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cliffordlab/amoss_services/database"
	"github.com/cliffordlab/amoss_services/mathb"
	"golang.org/x/crypto/bcrypt"
)

const (
	errorResJSON = `{"error":"json parsing error","error description":"key or value of json is formatted incorrectly"}`
)

type PasswordRecoveryHTTPRequest struct {
	ParticipantID int64  `json:"participantID"`
	Password      string `json:"password"`
}

type PasswordRecoveryHandler struct {
	Name string
}

func (pwRch PasswordRecoveryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("##### Start Password Recovery process ... #####")
	var pwRcHTTPRequest PasswordRecoveryHTTPRequest
	var participantObject Participant

	//validate header is formatted properly
	log.Println("Check start check header")
	splitMarsToken := checkHeader(w, r)
	log.Println("Mars Token: %+v\n", splitMarsToken)
	// Decode a stream of distinct JSON values from the http.ResponseWriter
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&pwRcHTTPRequest); err != nil {
		log.Println(err)
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Write([]byte(errorResJSON))
		return
	}
	log.Println("After decoding: %+v", pwRcHTTPRequest)

	// Check if participant ID exist
	if checkParticipantID(pwRcHTTPRequest, w) {
		// Encrypt Password and return object
		participantObject = encryptPassword(pwRcHTTPRequest, participantObject)
		// Fill until 10 digit with 0s
		ptidLen := int(math.Log10(float64(participantObject.ID)) + 1)
		digitsToPlaceAtEnd := 10 - ptidLen

		for i := 0; i < digitsToPlaceAtEnd; i++ {
			participantObject.ID = participantObject.ID*10 + 0
		}

		// Update new password into database
		log.Println("Start Uploading to db")
		updatePasswordDB(participantObject, w)
	} else {
		w.Write([]byte("{\"Fail\": \"Participant '" + strconv.Itoa(int(participantObject.ID)) + "' does not exist in the database\"}"))
	}

	log.Println("##### End Password Recovery process #####")
}

func checkParticipantID(pwRcHTTPRequest PasswordRecoveryHTTPRequest, w http.ResponseWriter) bool {
	var participantExist bool
	// Prepared statement for later queries or executions
	stmt, err := database.ADB.Db.Prepare(`SELECT EXISTS(SELECT 1 FROM participants WHERE participant_id=($1))`)
	if err != nil {
		log.Println("Failed to prepare statement for later queries or executions")
		log.Fatalln(err)
	}
	defer stmt.Close()
	rows, err := stmt.Query(&pwRcHTTPRequest.ParticipantID)
	if err != nil {
		log.Println("failed to execute query statement")
		log.Println(err)
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.WriteHeader(http.StatusOK)
	}

	for rows.Next() {
		if err := rows.Scan(&participantExist); err != nil {
			log.Println("failed to scan row for ID")
			log.Println(participantExist)
		}
		log.Println(participantExist)
	}
	rows.Close()
	return participantExist
}

func checkHeader(w http.ResponseWriter, r *http.Request) []string {
	marsToken := r.Header.Get("Authorization")
	splitMarsToken := strings.Split(marsToken, " ")
	if splitMarsToken[0] != "Bearer" {
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid header"}`))
		return splitMarsToken
	}
	return splitMarsToken
}

func encryptPassword(pwRcHTTPRequest PasswordRecoveryHTTPRequest, participantObject Participant) Participant {
	// create a random salt
	log.Println("Ready to alter salt and pwHash")
	var src = rand.NewSource(time.Now().UnixNano())
	salt := mathb.RandString(58, src)

	participantObject.Salt = salt

	//merge salt and password to prepare for hashing
	password := participantObject.Salt + pwRcHTTPRequest.Password

	// Generate "hash" to store from user password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Println("brcypt hash failed")
	}

	participantObject.PasswordHash = string(passwordHash)
	participantObject.ID = int64(pwRcHTTPRequest.ParticipantID)

	return participantObject
}

func updatePasswordDB(participantObject Participant, w http.ResponseWriter) {
	log.Println("Start password update process...")
	stmt, err := database.ADB.Db.Prepare(`UPDATE participants SET (password_hash, password_salt) = ($1, $2) WHERE participant_id = $3;`)
	if err != nil {
		log.Println("failed to prepare alter participant statement")
		log.Fatalln(err)
	}
	defer stmt.Close()

	// Execute query statement
	rows, err := stmt.Query(participantObject.PasswordHash, participantObject.Salt, participantObject.ID)
	if err != nil {
		log.Println("failed to execute query statement")
		log.Println(err)
		return
	}
	w.Write([]byte("{\"success\": \"Participant '" + strconv.Itoa(int(participantObject.ID)) + "' password has been recovered\"}"))
	rows.Close()
}
