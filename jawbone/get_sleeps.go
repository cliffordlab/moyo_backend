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
	PARTICIPANT_SLEEP_ENDPOINT = "https://jawbone.com/nudge/api/v.1.1/users/@me/sleeps"
	TICK_BASE_ENDPOINT         = "https://jawbone.com/nudge/api/v.1.1/sleeps/"
)

type SleepResults struct {
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

type SleepItems struct {
	XID           string `json:"xid"`
	Title         string `json:"title"`
	SubType       int    `json:"sub_type"`
	TimeCreated   int    `json:"time_created"`
	TimeCompleted int    `json:"time_completed"`
	Date          int    `json:"date"`
	PlaceLat      string `json:"place_lat"`
	PlaceLon      string `json:"place_lon"`
	PlaceAcc      string `json:"place_acc"`
	PlaceName     string `json:"place_name"`
	Details       struct {
		SmartAlarmFire int    `json:"smart_fire_alarm"`
		AwakeTime      int    `json:"awake_time"`
		AsleepTime     int    `json:"asleep_time"`
		Awakenings     int    `json:"awakenings"`
		Rem            int    `json:"rem"`
		Light          int    `json:"light"`
		Deep           int    `json:"deep"`
		Awake          int    `json:"awake"`
		Duration       int    `json:"duration"`
		TZ             string `json:"tx"`
	} `json:"details"`
}

func getSleeps(client *http.Client, accessToken string, sleepsURL string) ([]byte, bool) {
	req, err := http.NewRequest("GET", sleepsURL, nil)
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

func getSleepsXidsURL(date string) string {
	req, err := http.NewRequest("GET", PARTICIPANT_SLEEP_ENDPOINT, nil)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	q := req.URL.Query()
	q.Add("date", date)
	req.URL.RawQuery = q.Encode()
	return req.URL.String()
}

func getSleepPhases(client *http.Client, accessToken string, xid string) ([]byte, bool) {
	sleepPhasesURL := fmt.Sprintf("https://jawbone.com/nudge/api/v.1.1/sleeps/%s/ticks", xid)

	req, err := http.NewRequest("GET", sleepPhasesURL, nil)
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
