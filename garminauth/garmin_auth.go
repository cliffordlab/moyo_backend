package garminauth

import (
	"encoding/json"
	"fmt"
	"github.com/cliffordlab/amoss_services/capacity"
	"github.com/cliffordlab/amoss_services/log_writer"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"

	"github.com/cliffordlab/amoss_services/database"
	"github.com/garyburd/go-oauth/oauth"
)

const errorResJSON = `{"error":"json parsing error","error description":"key or value of json is formatted incorrectly"}`

type GarminUnauthorizedRequestHandler struct {
	Name string
}

type GarminAccessTokenHandler struct {
	Name string
}

type GarminTokens struct {
	RequestToken       string `json:"oauth_token"`
	RequestTokenSecret string `json:"oauth_token_secret"`
	Verifier           string `json:"oauth_verifier"`
	Participant        int64  `json:"ptid"`
}

func (gh GarminAccessTokenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("Serving Garmin access token...")
	w = log_writer.LogWriter{w}
	dec := json.NewDecoder(r.Body)
	var gt GarminTokens
	if err := dec.Decode(&gt); err != nil {
		log.Println(err)
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Write([]byte(errorResJSON))
		return
	}
	tokens, _ := acquireAccessToken(gt)
	tokenMap, _ := url.ParseQuery(tokens)

	ptidLen := int(math.Log10(float64(gt.Participant)) + 1)
	digitsToPlaceAtEnd := 10 - ptidLen

	for i := 0; i < digitsToPlaceAtEnd; i++ {
		gt.Participant = gt.Participant*10 + 0
	}

	updateTokens := `UPDATE participants SET (garmin_access_token, garmin_secret_token)
	= ($1, $2) WHERE participant_id = $3`

	stmt, err := database.ADB.Db.Prepare(updateTokens)
	if err != nil {
		log.Println("failed to prepare alter participant statement")
		log.Fatalln(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(tokenMap["oauth_token"][0],
		tokenMap["oauth_token_secret"][0], gt.Participant)
	if err != nil {
		log.Println("failed to execute query statement")
		log.Println(err)
		return
	}
	rows.Close()

	w.WriteHeader(http.StatusCreated)
	w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	w.Write([]byte(`{"result":"success"}`))
}

func (gh GarminUnauthorizedRequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling unauthorized Garmin request...")
	w = log_writer.LogWriter{w}
	tokens, _ := acquireUnauthorizedTokenAndSecret()
	tokenMap, _ := url.ParseQuery(tokens)
	response := fmt.Sprintf(`
		{
			"oauth_token":"%s",
			"oauth_token_secret":"%s"
		}
		`, tokenMap["oauth_token"][0], tokenMap["oauth_token_secret"][0])

	w.WriteHeader(http.StatusOK)
	w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	w.Write([]byte(response))
}

func acquireAccessToken(gt GarminTokens) (string, error) {
	log.Println("Acquiring access token...")
	// adding http in client for nice but will pass in as parameter later
	req, err := http.NewRequest("POST",
		"https://connectapi.garmin.com/oauth-service/oauth/access_token", nil)
	if err != nil {
		return "", err
	}
	client := oauth.Client{}

	// Sign the request.
	if err := client.SetAuthorizationHeader(req.Header, &oauth.Credentials{
		Token:  capacity.GarminToken,
		Secret: capacity.GarminSecret}, "POST", req.URL, nil,
		gt.Verifier, gt.RequestToken, gt.RequestTokenSecret); err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	bodyString := string(bodyBytes)
	return bodyString, nil
}

func acquireUnauthorizedTokenAndSecret() (string, error) {
	log.Println("Acquiring unauthorized token and secret...")
	// adding http in client for nice but will pass in as parameter later
	req, err := http.NewRequest("POST", "https://connectapi.garmin.com/oauth-service/oauth/request_token", nil)
	if err != nil {
		return "", err
	}
	client := oauth.Client{}

	// Sign the request.
	if err := client.SetAuthorizationHeader(req.Header, &oauth.Credentials{
		Token:  capacity.GarminToken,
		Secret: capacity.GarminSecret}, "POST", req.URL, nil, "", "", ""); err != nil {
		return "", err
	}
	//delete this because stdout sgarmin key
	//log.Printf("This is the header for request token: \n%s\n\n", req.Header.Get("Authorization"))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	bodyString := string(bodyBytes)
	return bodyString, nil
}
