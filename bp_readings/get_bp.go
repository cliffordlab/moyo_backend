package bp_readings

import (
	"encoding/json"
	"fmt"
	"github.com/cliffordlab/amoss_services/capacity"
	"github.com/cliffordlab/amoss_services/database"
	"github.com/cliffordlab/amoss_services/participant"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	insertBP                 = ``
	selectBP                 = ``
	errorResJSON             = `{"error":"json parsing error","error description":"key or value of json is formatted incorrectly"}`
	errorInvalidIDOrPassword = `{"error":"invalid participant ID or password"}`
	noSuchUserErr            = `{"error":"invalid participant id or password"}`
)

type QueryHandler struct {
	Name string
}

type Body struct {
	ParticipantID int64  `json:"participantID"`
	StartingDate  string `json:"startingFromDate"`
	EndingDate    string `json:"endingDate"`
	Study         string `json:"study"`
}

type Series struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type SeriesChart struct {
	Name   string   `json:"name"`
	Series []Series `json:"series"`
}

type VitalsAndSymptomsSeries struct {
	Series []SeriesChart `json:"data"`
}

func (QueryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("Querying database for blood pressure data..")
	params := mux.Vars(r)
	pidString := params["participant_id"]
	pidInt64, _ := strconv.ParseInt(pidString, 10, 64)

	if s, err := strconv.ParseFloat(pidString, 64); err == nil {
		fmt.Println(s) // 3.14159265
		ptidLen := int(math.Log10(float64(s)) + 1)
		digitsToPlaceAtEnd := 10 - ptidLen
		for i := 0; i < digitsToPlaceAtEnd; i++ {
			pidInt64 = pidInt64*10 + 0
		}
	}

	log.Println("Checking Auth Token....")
	//validate header is formatted properly
	headerValue := r.Header.Get("Authorization")
	splitHeaderValue := strings.Split(headerValue, " ")
	if splitHeaderValue[0] != "Bearer" {
		http.Error(w, "Invalid token type", http.StatusUnauthorized)
		return
	}
	bearerToken := splitHeaderValue[1]
	token, err := jwt.ParseWithClaims(bearerToken, &capacity.NonAdminClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Make sure token's signature wasn't changed
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected siging method")
		}
		return []byte(capacity.JwtSecret), nil
	})

	if err != nil {
		log.Println("token not valid")
		http.Error(w, "Invalid token type", http.StatusUnauthorized)
		return
	}

	var currentParticipant participant.Participant
	if claims, ok := token.Claims.(*capacity.NonAdminClaims); ok && token.Valid {
		currentParticipant.Study = claims.Study
		currentParticipant.ID = claims.ID
	} else {
		log.Println("token not valid")
		http.Error(w, "Invalid token type", http.StatusUnauthorized)
		return
	}

	log.Println("Querying database...")

	vitalsData, done := getVitalsData(w, err, pidInt64)
	if done {
		return
	}
	symptomsData, done := getSymptomsData(w, err, pidInt64)
	if done {
		return
	}

	var symptomsSeriesChart []VitalsAndSymptomsSeries
	symptomsSeriesChart = append(symptomsSeriesChart, VitalsAndSymptomsSeries{vitalsData})
	symptomsSeriesChart = append(symptomsSeriesChart, VitalsAndSymptomsSeries{symptomsData})

	jsonObject, _ := json.MarshalIndent(symptomsSeriesChart, "", "    ")
	log.Println(string(jsonObject))
	w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	w.Write(jsonObject)
}

func getSymptomsData(w http.ResponseWriter, err error, pidInt64 int64) ([]SeriesChart, bool) {
	querySymptoms := "SELECT created_at, blurried_vision, headache, difficulty_breathing, side_pain FROM mme_symptoms WHERE participant_id = $1"
	rows, err := database.ADB.Db.Query(querySymptoms, pidInt64)
	if err != nil {
		log.Println("failed to retrieve blood pressure data")
		log.Fatalln(err)
		return nil, false
	}

	// Vital Data
	var id []byte
	var bv []byte
	var ha []byte
	var db []byte
	var sp []byte
	var createdAt []byte
	var seriesArray []Series
	var symptomsSeriesChart []SeriesChart
	bvTotalTrue := 0
	bvTotalFalse := 0
	hATotalTrue := 0
	hATotalFalse := 0
	dBTotalTrue := 0
	dBTotalFalse := 0
	sPTotalTrue := 0
	sPTotalFalse := 0

	for rows.Next() {
		err = rows.Scan(&id, &bv, &ha, &db, &sp)
		//idInt, _ := strconv.ParseInt(string(id), 10, 64)
		createdAtInt, _ := strconv.ParseInt("1"+string(createdAt), 10, 64)
		log.Println("this is the createdAt: " + string(createdAt))
		log.Println("this is the createdAtInt: " + strconv.FormatInt(createdAtInt, 10))
		tm := time.Unix(0, createdAtInt*int64(time.Millisecond))
		log.Println("this is the tm: " + tm.String())
		dateTimeFormatted := tm.Format("01/02/2006 3:04 PM")
		log.Println("this is the dateTimeFormatted: " + dateTimeFormatted)

		if err != nil {
			log.Println("failed to scan row for bp data")
			log.Println(err)
			w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			w.Write([]byte(noSuchUserErr))
			return nil, true
		}

		bvBool := string(bv)
		haBool := string(ha)
		dbBool := string(db)
		spBool := string(sp)

		switch bvBool {
		case "true":
			bvTotalTrue++
		case "false":
			bvTotalFalse++
		default:
		}

		switch haBool {
		case "true":
			hATotalTrue++
		case "false":
			hATotalFalse++
		default:
		}

		switch dbBool {
		case "true":
			dBTotalTrue++
		case "false":
			dBTotalFalse++
		default:
		}

		switch spBool {
		case "true":
			sPTotalTrue++
		case "false":
			sPTotalFalse++
		default:
		}
	}

	seriesArray = append(seriesArray, Series{
		Name: "Blurried Vision", Value: bvTotalTrue,
	})
	seriesArray = append(seriesArray, Series{
		Name: "Blurried Vision False", Value: bvTotalFalse,
	})
	seriesArray = append(seriesArray, Series{
		Name: "Headache", Value: hATotalTrue,
	})
	seriesArray = append(seriesArray, Series{
		Name: "Headache False", Value: hATotalFalse,
	})
	seriesArray = append(seriesArray, Series{
		Name: "Difficulty Breathing", Value: dBTotalTrue,
	})
	seriesArray = append(seriesArray, Series{
		Name: "Difficulty Breathing False", Value: dBTotalFalse,
	})
	seriesArray = append(seriesArray, Series{
		Name: "Side Pain", Value: sPTotalTrue,
	})
	seriesArray = append(seriesArray, Series{
		Name: "Side Pain False", Value: sPTotalFalse,
	})
	symptomsSeriesChart = append(symptomsSeriesChart, SeriesChart{
		Name:   "Symptoms",
		Series: seriesArray,
	})
	return symptomsSeriesChart, false
}

func getVitalsData(w http.ResponseWriter, err error, pidInt64 int64) ([]SeriesChart, bool) {
	queryVitals := "SELECT created_at, participant_id, systolic_bp, diastolic_bp, pulse, is_verified FROM bp_readings WHERE participant_id = $1 ORDER BY created_at ASC "

	rows, err := database.ADB.Db.Query(queryVitals, pidInt64)
	if err != nil {
		log.Println("failed to retrieve blood pressure data")
		log.Fatalln(err)
		return nil, true
	}

	// Vital Data
	var id []byte
	var sbp []byte
	var dbp []byte
	var pulse []byte
	var createdAt []byte
	var sbpArray []Series
	var dbpArray []Series
	var pulseArray []Series
	var isVerified bool
	var bpData []SeriesChart

	for rows.Next() {
		err = rows.Scan(&createdAt, &id, &sbp, &dbp, &pulse, &isVerified)
		//idInt, _ := strconv.ParseInt(string(id), 10, 64)
		createdAtInt, _ := strconv.ParseInt("1"+string(createdAt), 10, 64)
		log.Println("this is the createdAt: " + string(createdAt))
		log.Println("this is the createdAtInt: " + strconv.FormatInt(createdAtInt, 10))
		tm := time.Unix(0, createdAtInt*int64(time.Millisecond))
		log.Println("this is the tm: " + tm.String())
		dateTimeFormatted := tm.Format("01/02/2006 3:04 PM")
		log.Println("this is the dateTimeFormatted: " + dateTimeFormatted)

		dbpInt, _ := strconv.Atoi(string(dbp))
		sbpInt, _ := strconv.Atoi(string(sbp))
		pulseInt, _ := strconv.Atoi(string(pulse))
		//pid := idInt
		//if bpArray != nil {
		//	bpArray = append(bpArray, BloodPressure{
		//		CreatedAt: createdAtInt, Sbp: sbpInt, Dbp: dbpInt,
		//	})
		//} else {

		//todo convert created_at to date
		sbpArray = append(sbpArray, Series{
			Name: dateTimeFormatted, Value: sbpInt,
		})
		dbpArray = append(dbpArray, Series{
			Name: dateTimeFormatted, Value: dbpInt,
		})
		pulseArray = append(pulseArray, Series{
			Name: dateTimeFormatted, Value: pulseInt,
		})
		//}

		if err != nil {
			log.Println("failed to scan row for bp data")
			log.Println(err)
			w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			w.Write([]byte(noSuchUserErr))
			return nil, true
		}
	}
	//for _, element := range bpArray {
	//	bpData = append(bpData, BPData{
	//		ID:            pid,
	//		BloodPressure: element,
	//	})
	//}
	bpData = append(bpData, SeriesChart{
		Name:   "SBP",
		Series: sbpArray,
	})
	bpData = append(bpData, SeriesChart{
		Name:   "DBP",
		Series: dbpArray,
	})
	bpData = append(bpData, SeriesChart{
		Name:   "Pulse",
		Series: pulseArray,
	})
	return bpData, false
}
