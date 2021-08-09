package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/cliffordlab/amoss_services/capacity"
	jwt "github.com/dgrijalva/jwt-go"
)

//DynamoHandler acts as a proxy between the mobile application and s3
type DynamoHandler struct {
	Name string
	Svc  *dynamodb.DynamoDB
}

// Record to unmarshall dynamo db return
type Record struct {
	Size string
}

//FileQueryRequest payload for time of query needed
type FileQueryRequest struct {
	ParticipantID int    `json:"participantID"`
	Type          string `json:"type"`
	StartTime     string `json:"startTime"`
	EndTime       string `json:"endTime"`
}

const errorResJSON = `{"error":"json parsing error","error description":"key or value of json is formatted incorrectly"}`

func (dh DynamoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//validate header is formatted properly
	headerValue := r.Header.Get("Authorization")
	splitHeaderValue := strings.Split(headerValue, " ")
	if splitHeaderValue[0] != "Mars" {
		http.Error(w, "Invalid token type", http.StatusUnauthorized)
		return
	}
	marsToken := splitHeaderValue[1]
	_, err := jwt.ParseWithClaims(marsToken, &capacity.NonAdminClaims{}, func(token *jwt.Token) (interface{}, error) {
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

	dec := json.NewDecoder(r.Body)
	var fqr FileQueryRequest
	if err := dec.Decode(&fqr); err != nil {
		log.Println(err)
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Write([]byte(errorResJSON))
		return
	}

	ptid := strconv.Itoa(fqr.ParticipantID)
	// Create the service's client with the session.
	svc := dynamodb.New(session.New(&aws.Config{Region: aws.String("us-east-1")}))
	input := &dynamodb.QueryInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":v1": {
				S: aws.String(ptid),
			},
			":v2": {
				S: aws.String(fqr.Type),
			},
			":v3From": {
				S: aws.String(fqr.StartTime),
			},
			":v3To": {
				S: aws.String(fqr.EndTime),
			},
		},
		KeyConditionExpression: aws.String("participant_ID = :v1 AND unix_ts BETWEEN :v3From AND :v3To"),
		FilterExpression:       aws.String("extension = :v2"),
		ProjectionExpression:   aws.String("size"),
		TableName:              aws.String("Participant_Files"),
	}

	result, err := svc.Query(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeProvisionedThroughputExceededException:
				fmt.Println(dynamodb.ErrCodeProvisionedThroughputExceededException, aerr.Error())
			case dynamodb.ErrCodeResourceNotFoundException:
				fmt.Println(dynamodb.ErrCodeResourceNotFoundException, aerr.Error())
			case dynamodb.ErrCodeInternalServerError:
				fmt.Println(dynamodb.ErrCodeInternalServerError, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`"error":"Unable to complete query"`))
		return
	}

	resultCount := *result.Count
	records := make([]Record, resultCount, resultCount)

	dynamodbattribute.UnmarshalListOfMaps(result.Items, &records)
	fmt.Printf("Amount of items: %d\n", *result.Count)
	fmt.Println()

	var totalSize int64
	for _, record := range records {
		num, _ := strconv.ParseInt(record.Size, 10, 64)
		totalSize += num
	}

	resultsMap := map[string]int64{"count": resultCount, "size": totalSize}
	resultsJSON, _ := json.Marshal(resultsMap)
	w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(string(resultsJSON)))
}
