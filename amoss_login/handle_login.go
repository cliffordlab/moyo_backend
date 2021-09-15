package amoss_login

import (
	"encoding/json"
	"log"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/cliffordlab/amoss_services/mathb"
	"github.com/cliffordlab/amoss_services/participant"
	"golang.org/x/crypto/bcrypt"
)

const errorResJSON = `{"error":"json parsing error","error description":"key or value of json is formatted incorrectly"}`
const errorInvalidIDOrPassword = `{"error":"invalid participant ID or password"}`

//LoginHandler struct used to handle login requests
type LoginHandler struct {
	Name string
}

type AmossLoginRequest struct {
	ParticipantID int64  `json:"participantID"`
	Password      string `json:"password"`
	Study         string `json:"study"`
}

func (lh LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("User login request...")
	dec := json.NewDecoder(r.Body)
	var amr AmossLoginRequest
	if err := dec.Decode(&amr); err != nil {
		log.Println(err)
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Write([]byte(errorResJSON))
		return
	}

	var currentParticipant participant.Participant
	// checkID used to login
	log.Println("User logging in with participant ID...")
	currentParticipant.ID = amr.ParticipantID

	log.Println("Participant ID acquired...")
	log.Println("this is the participant ID: " + string(currentParticipant.ID))
	ptidLen := int(math.Log10(float64(currentParticipant.ID)) + 1)
	digitsToPlaceAtEnd := 10 - ptidLen

	for i := 0; i < digitsToPlaceAtEnd; i++ {
		currentParticipant.ID = currentParticipant.ID*10 + 0
	}

	log.Println("this is the participant ID after digitsPlaceAtEnd: " + string(currentParticipant.ID))

	participant.Salt(&currentParticipant, currentParticipant.ID, w)

	if currentParticipant.Salt == "" {
		return
	}
	password := currentParticipant.Salt + amr.Password

	participant.Password(currentParticipant.ID, &currentParticipant)
	if err := bcrypt.CompareHashAndPassword([]byte(currentParticipant.PasswordHash), []byte(password)); err != nil {
		log.Println(err)
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Write([]byte(errorInvalidIDOrPassword))
		return
	}
	// save new salt and password hash
	var src = rand.NewSource(time.Now().UnixNano())
	newSalt := mathb.RandString(58, src)

	newPassword := newSalt + amr.Password
	// Generate new "hash" to store from user password
	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Println("bcrypt hash failed")
	}

	currentParticipant.Salt = newSalt
	currentParticipant.PasswordHash = string(newPasswordHash)

	participant.AlterSaltAndPasswordHash(&currentParticipant)
	participant.LoginParticipant(&currentParticipant, w)
}
