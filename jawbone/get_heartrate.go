package jawbone

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

const (
	participantHeartRateEndpoint = "https://jawbone.com/nudge/api/v.1.1/users/@me/heartrates"
)

type HeartRateResults struct {
	Meta struct {
		UserXid string `json:"user_xid"`
		Message string `json:"message"`
		Code    int    `json:"code"`
		Time    int    `json:"time"`
	} `json:"meta"`
	Data struct {
		Size int `json:"size"`
	}
}

func getHeartRate(client *http.Client, accessToken string, heartRateURL string) ([]byte, bool) {
	req, err := http.NewRequest("GET", heartRateURL, nil)
	if err != nil {
		log.Println(err)
	}
	var buffer bytes.Buffer
	buffer.WriteString("Bearer ")
	buffer.WriteString(accessToken)
	req.Header.Add("Authorization", buffer.String())
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}
	statusOk := false
	if resp.StatusCode == 200 {
		statusOk = true
		log.Println(string(body))
	}
	return body, statusOk
}

func getHeartRateURL(date string) string {
	req, err := http.NewRequest("GET", participantHeartRateEndpoint, nil)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	q := req.URL.Query()
	q.Add("date", date)
	req.URL.RawQuery = q.Encode()
	return req.URL.String()
}
