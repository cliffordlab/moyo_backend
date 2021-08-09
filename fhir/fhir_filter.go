package fhir

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	xj "github.com/basgys/goxml2json"
	"github.com/cliffordlab/amoss_services/database"
	"github.com/cliffordlab/fhir_filter/fhir_categories"
	"io/ioutil"
	"log"
	"net/http"
)

var (
	// ErrNameNotProvided is thrown when a name is not provided
	ErrNameNotProvided = errors.New("no name was provided in the HTTP body")
)

type FhirCreds struct {
	AmossParticipantID string `json:"participant_ID"`
	PatientToken string `json:"patient_token"`
	PatientID    string `json:"patient_ID"`
	Category     string `json:"category"`
}

//RegistrationHandler struct used to handle registration requests
type FhirFilterHandler struct {
	Name string
	Svc  *s3.S3
}

func (gh FhirFilterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startOfWeekMillis := r.Header.Get("weekMillis")

	if len(startOfWeekMillis) != 12 {
		log.Println("token length is wrong")
		w.Header().Add("Content-Type", "application/json; charset=UTF-8")
		w.Write([]byte("{\"error\": \"invalid header\"}"))
		return
	}
	//func FhirFilterHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {'
	requestBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	// stdout and stderr are sent to AWS CloudWatch Logs
	log.Printf("Start fhir request\n")
	fc := FhirCreds{}
	log.Println(r.Body)
	err = json.Unmarshal(requestBody, &fc)
	if err != nil {
		log.Println(err)
		log.Printf("json unmarshall error\n")
		//return events.APIGatewayProxyResponse{}, ErrNameNotProvided
		//http.Error(w, "Invalid email address", http.StatusUnauthorized)
		w.Write([]byte("json unmarshall error"))
	}

	if fc.Category == "" || fc.PatientID == "" {
		patientFailure := "{\"error\":\"missing patient credentials\"}"
		//return events.APIGatewayProxyResponse{
		//	Body:       patientFailure,
		//	StatusCode: 200,
		//}, nil
		log.Println("missing patient credentials or category not available")
		w.Write([]byte(patientFailure))

	}

	endpoints := map[string]string{
		"allergy":   "/AllergyIntolerance?patient=",
		"med_order": "/MedicationOrder?patient=",
		"condition": "/Condition?patient=",
		"observation":  "/Observation?patient=",
		"diagnostic":   "/DiagnosticReport?patient=",
		"immunization": "/Immunization?patient=",
		"procedure":    "/Procedure?patient=",
		"device":       "/Device?patient=",
		"doc_ref":      "/DocumentReference?patient=",
		"care_plan":    "/CarePlan?patient=",
	}

	category := endpoints[fc.Category]
	if category == "" {
		fhirFailure := "{\"error\":\"category not recognized\"}"
		//return events.APIGatewayProxyResponse{
		//	Body:       fhirFailure,
		//	StatusCode: 200,
		//}, nil
		log.Println("category not recognized")
		w.Write([]byte(fhirFailure))

	}
	client := &http.Client{}
	svc := s3.New(session.New(&aws.Config{Region: aws.String("us-east-1")}))

	for key, value := range endpoints {
		//const baseURL = "https://open-ic.epic.com/FHIR/api/FHIR/DSTU2"
		const baseURL = "https://epicintprxyprd.swmed.edu/FHIR/api/FHIR/DSTU2"
		//const baseURL = "https://epicintprxytst.swmed.edu/FHIR/api/FHIR/DSTU2"
		fullURL := fmt.Sprintf("%s%s%s", baseURL, value, fc.PatientID)

		if key == "observation" {
			fullURL = fullURL + "&code=8310-5"
		}

		log.Printf("This is the full url:")
		log.Printf(fullURL)

		req, err := http.NewRequest("GET", fullURL, nil)
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", fc.PatientToken))
		resp, err := client.Do(req)

		jsonResp, err := xj.Convert(resp.Body)
		if err != nil {
			log.Println(err)
			log.Println("could not convert xml to jsonResp")
		}
		log.Println("This is the jsonResp returned: ")
		//log.Println(jsonResp.String())

		bucket := "amoss-mhealth"

		//s3key := "moyo-utsw" + "/" + fc.AmossParticipantID + "/" + startOfWeekMillis + key + ".json"
		s3key := setKey(fc.AmossParticipantID, startOfWeekMillis, key)

		log.Println(s3key)
		var jsonFiltered = getJsonFiltered(key, jsonResp)
		log.Println("This is the jsonFiltered returned: ")
		//log.Println(jsonFiltered)

		body := bytes.NewReader(jsonFiltered)

		_, err = svc.PutObject(&s3.PutObjectInput{
			Bucket: &bucket,
			Key:    &s3key,
			Body:   body,
		})
		if err != nil {
			log.Println(err)
			s3Failure := "{\"error\":\"unable to upload to s3\"}"
			//return events.APIGatewayProxyResponse{
			//	Body:       s3Failure,
			//	StatusCode: 200,
			//}, nil
			log.Println("unable to upload to s3")
			w.Write([]byte(s3Failure))
		}
	}

	s3Response := "{\"success\":\"patient fhir data upload to S3 successful\"}"
	log.Println("fhir_data upload to s3 successful")
	w.Write([]byte(s3Response))

	//return events.APIGatewayProxyResponse{
	//	Body:       string(s3Response),
	//	StatusCode: 200,
	//}, nil
}

func getJsonFiltered(category string, jsonResp *bytes.Buffer) []byte {
	switch category {
	case "allergy":
		log.Println("filtering allergy")

		allergy := fhir_categories.Allergy{}
		allergyByte := allergy.GetFilteredCategory([]byte(jsonResp.String()))
		return allergyByte
	case "med_order":
		log.Println("filtering med_order")

		medication := fhir_categories.Medication{}
		medByte := medication.GetFilteredCategory([]byte(jsonResp.String()))
		return medByte
	case "condition":
		log.Println("filtering condition")

		condition := fhir_categories.Condition{}
		conditionByte := condition.GetFilteredCategory([]byte(jsonResp.String()))
		return conditionByte
	case "observation":
		log.Println("filtering observation")

		observation := fhir_categories.Observation{}
		observationByte := observation.GetFilteredCategory([]byte(jsonResp.String()))
		return observationByte
	case "diagnostic":
		log.Println("filtering diagnostic")

		report := fhir_categories.Report{} // diagnostic
		diagnosticByte := report.GetFilteredCategory([]byte(jsonResp.String()))
		return diagnosticByte
	case "immunization":
		log.Println("filtering immunization")

		immunization := fhir_categories.Immunization{}
		immunizationByte := immunization.GetFilteredCategory([]byte(jsonResp.String()))
		return immunizationByte
	case "procedure":
		log.Println("filtering procedure")

		procedure := fhir_categories.Procedure{}
		procedureByte := procedure.GetFilteredCategory([]byte(jsonResp.String()))
		return procedureByte
	case "device":
		log.Println("filtering device")

		device := fhir_categories.Device{}
		deviceByte := device.GetFilteredCategory([]byte(jsonResp.String()))
		return deviceByte
	case "doc_ref":
		log.Println("filtering doc_ref")

		document := fhir_categories.Document{}
		docByte := document.GetFilteredCategory([]byte(jsonResp.String()))
		return docByte
	case "care_plan":
		log.Println("filtering care_plan")

		carePlan := fhir_categories.CarePlan{}
		carePlanByte := carePlan.GetFilteredCategory([]byte(jsonResp.String()))
		return carePlanByte
	default:
		return nil
	}
}
//
//func main() {
//	lambda.Start(FhirFilterHandler)
//}
func setKey(currentParticipant string, startOfWeekMillis string, filename string) string {
	var key string
	switch database.ADB.Environment {
	case "dev":
		key = "dev/" + "moyo-utsw" + "/" + currentParticipant + "/" + startOfWeekMillis + "/" + filename
	case "local":
		key = "test/" + "moyo-utsw" + "/" + currentParticipant + "/" + startOfWeekMillis + "/" + filename
	default:
		key = "moyo-utsw" + "/" + currentParticipant + "/" + startOfWeekMillis + "/" + filename
	}
	return key
}