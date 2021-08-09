package moyo_mom_emory

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"log"
	"os"
)

func SendVitalEmail(msg *string) {
	log.Print("Attempting to send vital threshold email to clinician..")

	if *msg == "" {
		fmt.Println("You must supply a message and topic ARN")
		fmt.Println("Usage: go run SnsPublish.go -m MESSAGE -t TOPIC-ARN")
		os.Exit(1)
	}

	svc := sns.New(session.New(&aws.Config{Region: aws.String("us-east-1")}))

	result, err := svc.Publish(&sns.PublishInput{
		Subject:  aws.String("(URGENT)Moyo Mom Emory Study: THRESHOLD REACHED"),
		Message:  msg,
		TopicArn: aws.String("arn:aws:sns:us-east-1:628146180777:MyTopic"),
	})

	//sendSMS(msg, result, err, svc)

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	fmt.Println(*result.MessageId)
}

func SendSymptomsEmail(msg *string) {
	log.Print("Attempting to send symptom threshold email to clinician..")

	if *msg == "" {
		fmt.Println("You must supply a message and topic ARN")
		fmt.Println("Usage: go run SnsPublish.go -m MESSAGE -t TOPIC-ARN")
		os.Exit(1)
	}

	svc := sns.New(session.New(&aws.Config{Region: aws.String("us-east-1")}))

	result, err := svc.Publish(&sns.PublishInput{
		Subject:  aws.String("(URGENT)Moyo Mom Emory Study: THRESHOLD REACHED"),
		Message:  msg,
		TopicArn: aws.String("arn:aws:sns:us-east-1:628146180777:MyTopic"),
	})

	//sendSMS(msg, result, err, svc)

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	fmt.Println(*result.MessageId)
}

//
//func sendSMS(msg *string, result *sns.PublishOutput, err error, svc *sns.SNS) {
//	log.Print("Attempting to send vital threshold SMS to clinician..")
//	result, err = svc.Publish(&sns.PublishInput{
//		PhoneNumber: aws.String("+16786876602"),
//		Message:  msg,
//	})
//	if err != nil {
//		fmt.Println(err.Error())
//		os.Exit(1)
//	}
//}
