package jawbone

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

const (
	PARTICIPANT_MOVES_ENDPOINT = "https://jawbone.com/nudge/api/v.1.1/users/@me/moves"
)

type MoveResults struct {
	Meta struct {
		UserXid string `json:"user_xid"`
		Message string `json:"message"`
		Code    int    `json:"code"`
		Time    int    `json:"time"`
	} `json:"meta"`
	Data struct {
		Items []SleepItems `json:"items"`
		Size  int          `json:"size"`
	}
}

type MoveItems struct {
	XID           string `json:"xid"`
	Title         string `json:"title"`
	Type          string `json:"type"`
	TimeCreated   int    `json:"time_created"`
	TimeUpdated   int    `json:"time_updated"`
	TimeCompleted int    `json:"time_completed"`
	Date          int    `json:"date"`
	Details       struct {
		Distance      int `json:"distance"`
		KM            int `json:"km"`
		Steps         int `json:"steps"`
		ActiveTime    int `json:"active_time"`
		LongestActive int `json:"longest_active"`
	} `json:"details"`
}

func getMoves(client *http.Client, accessToken string, movesURL string) ([]byte, bool) {
	req, err := http.NewRequest("GET", movesURL, nil)
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

func getMovesXidsURL(date string) string {
	req, err := http.NewRequest("GET", PARTICIPANT_MOVES_ENDPOINT, nil)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	q := req.URL.Query()
	q.Add("date", date)
	req.URL.RawQuery = q.Encode()
	return req.URL.String()
}

func getMoveTicks(client *http.Client, accessToken string, xid string) ([]byte, bool) {
	moveTicksURL := fmt.Sprintf("https://jawbone.com/nudge/api/v.1.1/moves/%s/ticks", xid)

	req, err := http.NewRequest("GET", moveTicksURL, nil)
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
	log.Println(string(body))
	return body, statusOk
}
