package jawbone

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cliffordlab/amoss_services/capacity"
	"github.com/cliffordlab/amoss_services/database"
	jwt "github.com/dgrijalva/jwt-go"
)

// JawboneClientSecret is retrieved from vault is main function to be used in this handler
var JawboneClientSecret string

const errorResJSON = `{"error":"json parsing error","error description":"key or value of json is formatted incorrectly"}`

const (
	insertJawbone = `INSERT INTO wearables (wearable_id, access_token, refresh_token, brand, jawbone_date)
	VALUES ($1, $2, $3, (SELECT brand FROM wearable_brand WHERE brand=$4), $5);`
	updateParticipantWithJawbone = `UPDATE participants SET wearable_id = $1 WHERE participant_id = $2;`
	updateWearable               = `UPDATE wearables SET wearable_id = $1, access_token = $2, refresh_token = $3 WHERE wearable_id = $4`
	duplicateUserErr             = `{"error":"Participant already has jawbone added or is not a participant"}`
)

// JBTokenHandler to use to get
type JBTokenHandler struct {
	Name string
}

type jbTokens struct {
	refreshToken string
	accessToken  string
}

type jawboneParticipant struct {
	Code        string `json:"code"`
	Participant int64  `json:"participantID"`
}

func (jt JBTokenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//validate header is formatted properly
	headerValue := r.Header.Get("Authorization")
	splitHeaderValue := strings.Split(headerValue, " ")
	if splitHeaderValue[0] != "Mars" {
		http.Error(w, "Invalid token type", http.StatusUnauthorized)
		return
	}
	marsToken := splitHeaderValue[1]
	token, err := jwt.ParseWithClaims(marsToken, &capacity.NonAdminClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Make sure token's signature wasn't changed
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected siging method")
		}
		return []byte(capacity.JwtSecret), nil
	})
	if err != nil {
		log.Println("unable to parse with claims")
		http.Error(w, "Invalid token type", http.StatusUnauthorized)
		return
	}

	//variable keeps jawbone refresh and access tokens
	var tokens jbTokens
	var jbp jawboneParticipant
	var currentStudy = ""

	if claims, ok := token.Claims.(*capacity.NonAdminClaims); ok && token.Valid {
		if claims.Capacity == "coordinator" {
			currentStudy = claims.Study
			dec := json.NewDecoder(r.Body)
			if err := dec.Decode(&jbp); err != nil {
				log.Println(err)
				w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
				w.Write([]byte(errorResJSON))
				return
			}
			getParticipantAccessToken(jbp.Code, w, &tokens)
		} else {
			log.Println("token not valid")
			http.Error(w, `{"error": "Invalid token type"}`, http.StatusOK)
			return
		}
	} else {
		log.Println("token not valid")
		http.Error(w, `{"error": "Invalid token type"}`, http.StatusOK)
		return
	}

	ptidLen := int(math.Log10(float64(jbp.Participant)) + 1)
	digitsToPlaceAtEnd := 10 - ptidLen

	for i := 0; i < digitsToPlaceAtEnd; i++ {
		jbp.Participant = jbp.Participant*10 + 0
	}

	//make request to retrieve xid
	if tokens.accessToken != "" {
		client := &http.Client{}
		req, err := http.NewRequest("GET", "https://jawbone.com/nudge/api/v.1.1/users/@me", nil)
		if err != nil {
			log.Println(err)
			log.Println("failed to create request")
			http.Error(w, `{"error": "Unable to add Jawbone"}`, http.StatusOK)
			return
		}
		req.Header.Add("Authorization", "Bearer "+tokens.accessToken)
		resp, err := client.Do(req)
		if err != nil {
			log.Println(err)
			log.Println("request to jawbone servers failed")
			http.Error(w, `{"error": "Unable to add Jawbone"}`, http.StatusOK)
			return
		}
		defer resp.Body.Close()
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		bodyString := string(bodyBytes)

		var m map[string]map[string]interface{}
		if err := json.Unmarshal([]byte(bodyString), &m); err != nil {
			log.Println(err)
			log.Println("request to jawbone servers failed")
			http.Error(w, `{"error": "Unable to add Jawbone"}`, http.StatusOK)
			return
		}
		xid := m["data"]["xid"].(string)

		// insert xid, access token, and refresh token into db
		stmt, err := database.ADB.Db.Prepare(insertJawbone)
		if err != nil {
			log.Println("preparation insert wearable id query statement failed")
			log.Println(err)
			http.Error(w, `{"error": "Unable to add Jawbone"}`, http.StatusOK)
			return
		}
		defer stmt.Close()
		t := time.Now()
		formattedT := t.Format("2006-01-02")
		rows, err := stmt.Query(xid, tokens.accessToken, tokens.refreshToken, "jawbone", formattedT)
		if err != nil {
			log.Println(err.Error())
			if currentStudy == "sleepBank" {
				if strings.Contains(err.Error(), "duplicate key") && strings.Contains(err.Error(), "wearables_pkey") {
					// put this into a function to avoid duplicate code
					stmt, err = database.ADB.Db.Prepare(updateParticipantWithJawbone)
					if err != nil {
						log.Println("preparation of update participant with jawbone query statement failed")
						log.Println(err)
						http.Error(w, `{"error": "Unable to add Jawbone"}`, http.StatusOK)
						return
					}
					defer stmt.Close()

					rows, err = stmt.Query(xid, jbp.Participant)
					if err != nil {
						log.Println("failed to execute update participant with jawbone query statement")
						log.Println(err)
						w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(duplicateUserErr))
						return
					}
					rows.Close()

					response := fmt.Sprint("{\"success\":\"jawbone added to participant\"}")
					w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
					w.WriteHeader(http.StatusCreated)
					w.Write([]byte(response))
					return
				}
			} else {
				log.Println("failed to execute add jawbone query statement")
				w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(duplicateUserErr))
				return
			}
		}
		rows.Close()

		stmt, err = database.ADB.Db.Prepare(updateParticipantWithJawbone)
		if err != nil {
			log.Println("preparation of update participant with jawbone query statement failed")
			log.Println(err)
			http.Error(w, `{"error": "Unable to add Jawbone"}`, http.StatusOK)
			return
		}
		defer stmt.Close()

		rows, err = stmt.Query(xid, jbp.Participant)
		if err != nil {
			log.Println("failed to execute update participant with jawbone query statement")
			log.Println(err)
			w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(duplicateUserErr))
			return
		}
		rows.Close()

		response := fmt.Sprint("{\"success\":\"jawbone added to participant\"}")
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(response))
	}
}

func getParticipantAccessToken(code string, w http.ResponseWriter, tokens *jbTokens) {
	form := url.Values{}
	form.Add("grant_type", "authorization_code")
	form.Add("client_id", "ccBFTw7TxbY")
	form.Add("client_secret", JawboneClientSecret)
	form.Add("code", code)

	baseURL := "https://jawbone.com/auth/oauth2/token"

	resp, err := http.PostForm(baseURL, form)
	if err != nil {
		log.Println(err)
		log.Println("error while making post request to jawbone")
		http.Error(w, `{"error": "failed to add jawbone"}`, http.StatusOK)
		return
	}
	recievedJSON, err := ioutil.ReadAll(resp.Body) //turning response body in bytes
	if err != nil {
		log.Println(err)
		log.Println("error while converting the response body into bytes")
		http.Error(w, `{"error": "failed to add jawbone"}`, http.StatusOK)
		return
	}
	//turns bytes in recievedJson to values in arbitraryJson map
	var arbitraryJSON map[string]interface{}
	json.Unmarshal([]byte(recievedJSON), &arbitraryJSON)

	//loop through maps to get needed values
	var accessToken string
	var refreshToken string
	for key, value := range arbitraryJSON {
		switch key {
		case "access_token":
			accessToken = value.(string)
		case "refresh_token":
			refreshToken = value.(string)
		default:
		}
	}

	if accessToken != "" && refreshToken != "" {
		log.Printf("This is access token testing %s\n", accessToken)
		tokens.accessToken = accessToken
		tokens.refreshToken = refreshToken
		return
	}
	log.Printf("Unsuccessful request to jawbone for tokens\n")
	http.Error(w, `{"error": "failed to add jawbone"}`, http.StatusOK)
}
