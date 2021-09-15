package support

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/cliffordlab/amoss_services/capacity"
	"log"
	"net/http"
)

const (
	errorResJSON     = `{"error":"json parsing error","error description":"key or value of json is formatted incorrectly"}`
	defaultMaxMemory = 32 << 20
)

type ContactSupportHandler struct {
	Name string
}

//Handler sends json body, subject and env to email lambda
func (ContactSupportHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	switch r.Method {

	case "POST":
		type AlertBody struct {
			Email   string `json:"email"`
			Body    string `json:"body"`
			Subject string `json:"subject"`
		}
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
		body := m.Get("body")
		subject := m.Get("subject")
		ab := &AlertBody{Email: email, Body: body, Subject: subject}

		log.Printf("email: " + email + ", body: " + body + ", subject: " + subject)
		log.Printf("send ending email info to lambda\n")
		//body := &AlertBody{EmailEncoded: email, Body: body, Subject: subject}
		jsonAlert, err := json.Marshal(ab)
		if err != nil {
			log.Println("Problem with ctask post body marshalling")
		}

		URL := "https://w5k4kp7yt1.execute-api.us-east-1.amazonaws.com/default/sendMoyoBetaEmail"
		req, err := http.NewRequest("POST", URL, bytes.NewBuffer(jsonAlert))
		if err != nil {
			log.Println("failure to create POST request")
			w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		//to point to correct login url
		apiKey := capacity.AWSSESLambdaAPIKey
		log.Println("This is the api key: ")
		req.Header.Add("Content-Type", "text/html")
		req.Header.Add("x-api-key", apiKey)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			//TODO need to add better error handling including
			//sending back internal server error
			log.Println("request to API Gateway failed")
			w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if resp.StatusCode == 200 {
			log.Print("the support email has been sent")
			log.Println(resp.StatusCode)
			w.Header().Add("Content-Type", "application/json; charset=UTF-8")
			w.Write([]byte("{\"response\": \"Support email has been sent\"}"))
		} else if resp.StatusCode == 202 {
			log.Print("the support email had no body")
			log.Println(resp.StatusCode)
			w.Header().Add("Content-Type", "application/json; charset=UTF-8")
			w.Write([]byte("{\"response\": \"Support email has not been sent. There was no data.\"}"))
		}

	default:
		http.Error(w, "HTTP Method needs to be POST", http.StatusHTTPVersionNotSupported)
	}
}

// AlertSupport contacts support team
func EmailMoyoBetaParticipant(participantID string, password string, email string, w http.ResponseWriter) {
	type AlertBody struct {
		Email   string `json:"email"`
		Body    string `json:"body"`
		Subject string `json:"subject"`
	}

	slicedParticipantID := participantID[0:4]

	log.Print("Sending login credentials to consented participant.. ")
	//put new message

	message := fmt.Sprintf("Thank you very much for being part of an ambitious study to end Heart Disease!\n\n"+
		"Here are your login credentials: \n"+
		"Login: %s \n"+
		"Password: %s \n\n"+
		"Download the app from the app store (Apple users): \n"+
		"https://itunes.apple.com/us/app/moyohealth/id1442116056?ls=1&mt=8 \n"+
		"Download the app from our website (Android users): \n"+
		"https://amoss.emory.edu/moyo/download \n"+
		"After you download, please use these credentials to log into the app. Thanks again for participating! \n\n"+
		"The MOYO Team \n\n"+
		"Website: http://moyohealth.net\n"+

		"Email: info@moyohealth.net", slicedParticipantID, password)

	subject := "Moyo login credentials"

	body := &AlertBody{Email: email, Body: message, Subject: subject}

	jsonAlert, err := json.Marshal(body)
	if err != nil {
		log.Println("Problem with ctask post body marshalling")
	}

	URL := "https://w5k4kp7yt1.execute-api.us-east-1.amazonaws.com/default/sendMoyoBetaEmail"
	req, err := http.NewRequest("POST", URL, bytes.NewBuffer(jsonAlert))
	if err != nil {
		log.Println("failure to create POST request")
		return
	}
	//to point to correct login url
	apiKey := capacity.AWSSESLambdaAPIKey
	//req.Header.Add("Content-Type", "application/json;charset=UTF-8")
	req.Header.Add("Content-Type", "text/html")
	req.Header.Add("x-api-key", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		//TODO need to add better error handling including
		//sending back internal server error
		log.Println("request to API Gateway failed")
		return
	}
	if resp.StatusCode == 200 {
		log.Print("the support email has been sent")
		log.Println(resp.StatusCode)
	} else if resp.StatusCode == 202 {
		log.Print("the support email had no body")
		log.Println(resp.StatusCode)
	}
	if resp.StatusCode != 200 {
		log.Print("request failed")
		log.Println(resp.StatusCode)
	}
	w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	w.Write([]byte("{\"success\":\"Participant enrolled successfully. EmailEncoded sent.\"}"))
}

// AlertSupport contacts support team
func EmailMoyoParticipant(email string, moyoID int64, password string, w http.ResponseWriter) (status string) {
	type AlertBody struct {
		Email   string `json:"email"`
		Body    string `json:"body"`
		Subject string `json:"subject"`
	}

	log.Print("Sending registration confirmation to participant.. ")

	message := fmt.Sprintf("Thank you very much for being part of an ambitious study to end Heart Disease!\n\n"+
		"You are now registered and can login using your moyo ID and this auto-generated password: \n\n"+

		"Email: %s \n"+
		"OR \n"+
		"Moyo ID: %v \n\n"+

		"Password: %s \n\n"+

		"The MOYO Team \n\n"+
		"Website: http://moyohealth.net\n"+

		"Email: info@moyohealth.net", email, moyoID, password)

	subject := "Moyo Registration Successful!"

	body := &AlertBody{Email: email, Body: message, Subject: subject}

	jsonAlert, err := json.Marshal(body)
	if err != nil {
		log.Println("Problem with ctask post body marshalling")
	}

	URL := "https://w5k4kp7yt1.execute-api.us-east-1.amazonaws.com/default/sendMoyoBetaEmail"
	req, err := http.NewRequest("POST", URL, bytes.NewBuffer(jsonAlert))
	if err != nil {
		log.Println("failure to create POST request")
		return
	}
	//to point to correct login url
	apiKey := capacity.AWSSESLambdaAPIKey
	//req.Header.Add("Content-Type", "application/json;charset=UTF-8")
	req.Header.Add("Content-Type", "text/html")
	req.Header.Add("x-api-key", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		//TODO need to add better error handling including
		//sending back internal server error
		log.Println("request to API Gateway failed")
		return
	}
	if resp.StatusCode == 200 {
		log.Print("the support email has been sent")
		log.Println(resp.StatusCode)
		return "success"
	} else if resp.StatusCode == 202 {
		log.Print("the support email had no body")
		log.Println(resp.StatusCode)
		return "partial"
	} else if resp.StatusCode != 200 {
		log.Print("request failed")
		log.Println(resp.StatusCode)
		return "failed"
	} else {
		return "error"
	}
	//w.Write([]byte("{\"success\":\"Participant registered successfully. Email sent.\"}"))
}
