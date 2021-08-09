package amoss_login

import (
	"encoding/json"
	"fmt"
	"github.com/cliffordlab/amoss_services/database"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/cliffordlab/amoss_services/capacity"
	"github.com/cliffordlab/amoss_services/mathb"
	"github.com/cliffordlab/amoss_services/participant"
	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
)

const (
	admin   = "admin"
	coord   = "coordinator"
	patient = "patient"
)

var (
	studyNames = [20]string{"hf", "chf", "depression monitoring", "moyo", "test", "super", "pCRF", "sleepBank", "utsw", "sleep technology", "ptsd-vns", "ptsd_twin", "ptsd_grc", "otsuka", "Anytime Fitness Study", "PRO-C study", "vismet", "cfd-sleep-study-test", "cfd-sleep_study", "cfd-classroom-audio"}
)

//RegistrationHandler struct used to handle registration requests
type RegistrationHandler struct {
	Name string
}

func (rh RegistrationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("Registering user... ")
	//validate header is formatted properly
	marsToken := r.Header.Get("Authorization")
	splitMarsToken := strings.Split(marsToken, " ")
	if splitMarsToken[0] != "Bearer" {
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid header"}`))
		return
	}

	//decode json into struct
	dec := json.NewDecoder(r.Body)
	var amr AmossLoginRequest
	if err := dec.Decode(&amr); err != nil {
		log.Println(err)
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Write([]byte(errorResJSON))
		return
	}
	//create a random salt
	var src = rand.NewSource(time.Now().UnixNano())
	salt := mathb.RandString(58, src)

	var newParticipant participant.Participant
	newParticipant.Salt = salt

	//merge salt and password to prepare for hashing
	password := newParticipant.Salt + amr.Password

	// Generate "hash" to store from user password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Println("brcypt hash failed")
	}

	newParticipant.PasswordHash = string(passwordHash)
	newParticipant.ID = int64(amr.ParticipantID)

	ptidLen := int(math.Log10(float64(newParticipant.ID)) + 1)
	digitsToPlaceAtEnd := 10 - ptidLen

	for i := 0; i < digitsToPlaceAtEnd; i++ {
		newParticipant.ID = newParticipant.ID*10 + 0
	}

	var currentParticipant participant.Participant
	var namespace = "/" + database.ADB.Environment
	if database.ADB.Environment == "local" {
		namespace = "/dev"
	}
	switch r.URL.Path {
	case "/api/createAdmin":
		newParticipant.Capacity = "admin"
		participant.CreateAdmin(newParticipant, w)
	case "/api/createCoordinator":
		log.Println("Creating coordinator...")
		setupNewParticipant(coord, splitMarsToken[1], w, &currentParticipant, &newParticipant, &amr)
		if newParticipant.Study != "" {
			participant.CreateNonAdmin(newParticipant, w)
		} else {
			http.Error(w, `{"error":"study type invalid or not included"}`, http.StatusOK)
			return
		}
	case "/api/createPatient":
		log.Println("Creating participant...")
		setupNewParticipant(patient, splitMarsToken[1], w, &currentParticipant, &newParticipant, &amr)
		if newParticipant.Study != "" {
			participant.CreateNonAdmin(newParticipant, w)
		} else {
			http.Error(w, `{"error":"study type invalid or not included"}`, http.StatusOK)
			return
		}
	case namespace + "/api/createAdmin":
		newParticipant.Capacity = "admin"
		participant.CreateAdmin(newParticipant, w)
	case namespace + "/api/createCoordinator":
		log.Println("Creating coordinator...")
		setupNewParticipant(coord, splitMarsToken[1], w, &currentParticipant, &newParticipant, &amr)
		if newParticipant.Study != "" {
			participant.CreateNonAdmin(newParticipant, w)
		} else {
			http.Error(w, `{"error":"study type invalid or not included"}`, http.StatusOK)
			return
		}
	case namespace + "/api/createPatient":
		log.Println("Creating participant...")
		setupNewParticipant(patient, splitMarsToken[1], w, &currentParticipant, &newParticipant, &amr)
		if newParticipant.Study != "" {
			participant.CreateNonAdmin(newParticipant, w)
		} else {
			http.Error(w, `{"error":"study type invalid or not included"}`, http.StatusOK)
			return
		}
	}
}

func setupNewParticipant(cap string, marsToken string, w http.ResponseWriter, cp *participant.Participant, np *participant.Participant, amr *AmossLoginRequest) {
	log.Println("Setting up new participant...")
	token, err := jwt.ParseWithClaims(marsToken, &capacity.NonAdminClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Make sure token's signature wasn't changed
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method")
		}
		return []byte(capacity.JwtSecret), nil
	})

	if err != nil {
		http.Error(w, "Invalid token type", http.StatusUnauthorized)
		return
	}

	if claims, ok := token.Claims.(*capacity.NonAdminClaims); ok && token.Valid {
		cp.Capacity = claims.Capacity
		cp.Study = claims.Study
		cp.ID = claims.ID
	} else {
		http.Error(w, "Invalid token type", http.StatusUnauthorized)
		return
	}

	//patient capacity cannot create users
	if cp.Capacity == patient {
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Write([]byte(`{"error":"policy incapable of creating participants"}`))
		return
	}

	np.Capacity = cap
	//TODO check in 1 month if we should delete Feb25 by tomy
	//makes sure participant policies cannot create user in different study
	log.Println(cap)
	log.Println(cp.Capacity)
	log.Println(cp.ID)
	log.Println(cp.Study)
	if cp.Capacity == cap || cap == patient {
		np.Study = cp.Study
	} else {
		for _, value := range studyNames {
			if value == amr.Study {
				np.Study = amr.Study
			}
		}
	}
}
