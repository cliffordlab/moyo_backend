package amoss_login

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/cliffordlab/amoss_services/capacity"
	"github.com/cliffordlab/amoss_services/database"
	"github.com/cliffordlab/amoss_services/mathb"
	"github.com/cliffordlab/amoss_services/support"
	"golang.org/x/crypto/bcrypt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

const (
	defaultMaxMemory = 32 << 20
	errorResJSONMoyo = `{"error":"json parsing error test","error description":"key or value of json is formatted incorrectly"}`
	noSuchUserErr    = `{"error":"invalid email address."}`
	alterParticipant = `UPDATE participants SET (password_hash, password_salt) = ($1, $2) WHERE participant_id = $3;`
)

//RegistrationHandler struct used to handle registration requests
type MoyoBetaRegistrationHandler struct {
	Name string
}

type MoyoBetaRegistrationRequest struct {
	Email string `json:"email"`
}

func (MoyoBetaRegistrationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("Attempting to register Moyo Beta Participant..")

	err := r.ParseMultipartForm(defaultMaxMemory)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println("This is the error: ")
		log.Println(err)
		return
	}

	//get a ref to the parsed multipart form
	m := r.Form
	email := m.Get("email")

	// hash and pepper email
	emailHash := getEmailHash(email)

	//lookup email in db
	query := "SELECT participant_id FROM participants WHERE email_hash = $1"
	rows, err := database.ADB.Db.Query(query, emailHash)
	if err != nil {
		log.Println("failed to execute get participant ID query")
		log.Println("This email is not on whitelist..")
		log.Println(err)
		http.Error(w, "Invalid email address", http.StatusUnauthorized)
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Write([]byte(noSuchUserErr))
		return
	}

	//SEND THIS PARTICIPANT ID IN EMAIL
	var participantID string
	for rows.Next() {
		err = rows.Scan(&participantID)
		if err != nil {
			log.Println("failed to scan row for participant ID")
			log.Println("This email is not on whitelist..")
			log.Println(err)
			w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			w.Write([]byte(noSuchUserErr))
		}
	}
	rows.Close()

	// generate random password and insert into DB (password hash and password_salt)
	salt, passwordHash, password := getPasswordHash(err)

	// Insert new password into DB
	log.Println("Ready to alter salt and pwHash")
	stmt, err := database.ADB.Db.Prepare(alterParticipant)
	if err != nil {
		log.Println("failed to prepare alter participant statement")
		log.Fatalln(err)
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Write([]byte(noSuchUserErr))
	}
	defer stmt.Close()

	rows2, err := stmt.Query(passwordHash, salt, participantID)
	if err != nil {
		log.Println("failed to execute query statement. particpant id error.")
		log.Println(err)
		http.Error(w, "Invalid email address.", http.StatusUnauthorized)
		//w.Write([]byte(noSuchUserErr))
		return
	}
	rows2.Close()

	// send email with participant ID and password if everything is successful
	support.EmailMoyoBetaParticipant(participantID, password, email, w)
}

func getPasswordHash(err error) (string, []byte, string) {
	//create a random salt
	var src = rand.NewSource(time.Now().UnixNano())
	salt := mathb.RandString(58, src)
	// SEND THIS PASSWORD IN EMAIL
	password := RandomString(7)
	//merge salt and password to prepare for hashing
	//INSERT THIS INTO DB password_salt
	passwordSalt := salt + password
	// Generate "hash" to store from user password
	//INSERT THIS INTO DB password_hash
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(passwordSalt), bcrypt.DefaultCost)

	if err != nil {
		log.Println("brcypt hash failed")
	}
	return salt, passwordHash, password
}

func getEmailHash(email string) string {
	var pepper = capacity.JwtSecret
	//var email = email
	pepperedEmail := fmt.Sprintf("%s%s", email, pepper)
	// Generate "hash" to store from user password
	hasherSha256 := sha256.New()
	hasherSha256.Write([]byte(pepperedEmail))
	emailHash := hex.EncodeToString(hasherSha256.Sum(nil))
	return emailHash
}

func RandomString(len int) string {
	bytes := make([]byte, len)
	for i := 0; i < len; i++ {
		bytes[i] = byte(65 + rand.Intn(25)) //A=65 and Z = 65+25
	}
	return strings.ToLower(string(bytes))
}
