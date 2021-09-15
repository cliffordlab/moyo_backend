package support

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/cliffordlab/amoss_services/capacity"
	"github.com/cliffordlab/amoss_services/cryptography"
	"github.com/cliffordlab/amoss_services/database"
)

//RegistrationHandler struct used to handle registration requests
type MoyoDecryptPIDHandler struct {
	Name string
}
type MoyoDecryptPIDRequest struct {
	ParticipantID int64  `json:"participantID"`
	Email         string `json:"email"`
	Phone         int    `json:"phone"`
	Password      string `json:"password"`
	Study         string `json:"study"`
}

func (h MoyoDecryptPIDHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	log.Println("Decrypting Moyo participant...")

	//decode json into struct
	dec := json.NewDecoder(r.Body)
	var mdr MoyoDecryptRequest
	if err := dec.Decode(&mdr); err != nil {
		log.Println("Error decoding registration request!")
		log.Println(err)
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Write([]byte(errorResJSON))
		return
	}

	pid := mdr.ParticipantID
	log.Println("this is the participant ID: ")
	fmt.Println(pid)

	//query db with participant id to get encrypted email and iv
	stmt, err := database.ADB.Db.Prepare(getEmail)
	if err != nil {
		log.Println("failed to prepare select email statement")
		log.Fatalln(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(pid)
	if err != nil {
		log.Println("failed to execute query statement")
		log.Println(err)
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Write([]byte(noSuchUserErr))
		return
	}

	var email []byte
	for rows.Next() {
		err = rows.Scan(&email)
		if err != nil {
			log.Println("failed to scan row for ecrypted email")
			log.Println(err)
		}
	}
	rows.Close()

	fmt.Println(email)
	fmt.Println(string(email))
	fmt.Printf("%x", email)
	log.Println("email prints complete")

	ivStatement, err := database.ADB.Db.Prepare(getIV)
	if err != nil {
		log.Println("failed to prepare select iv statement")
		log.Fatalln(err)
	}
	defer ivStatement.Close()

	ivRows, err := ivStatement.Query(pid)
	if err != nil {
		log.Println("failed to execute query statement")
		log.Println(err)
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(noSuchUserErr))
		return
	}
	var iv []byte
	for ivRows.Next() {
		err = ivRows.Scan(&iv)
		if err != nil {
			log.Println("failed to scan row for ecrypted iv")
			log.Println(err)
		}
	}
	ivRows.Close()

	key := []byte(capacity.CryptoKey)

	log.Println("")
	log.Println("this is the IV: ")
	fmt.Println(iv)
	fmt.Println(string(iv))
	fmt.Printf("%x", iv)
	log.Println("iv prints complete")

	//decrypt email
	decryptedEmail, err := cryptography.Decrypt(key, email, iv)
	if err != nil {
		log.Println("Decryption failed")
		log.Fatalln(err)
	}

	//respond with decrypted email
	//log.Println("this is the email decrypted: " + string(decryptedEmail))
	response := fmt.Sprintf(`{"email":"%s"}`, string(decryptedEmail))
	w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	w.Write([]byte(response))
}
