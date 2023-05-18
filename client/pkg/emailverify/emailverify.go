package emailverify

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/ses"
	"log"
	"math/rand"
	"time"
)

const (
	CharSet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

type Verification struct {
	Email string
	Code  string
}

func GenerateVerificationCode(length int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, length)
	for i := range b {
		b[i] = CharSet[rand.Intn(len(CharSet))]
	}
	return string(b)
}

func SendVerificationEmail(sess *session.Session, emailAddress string) {
	svc := ses.New(sess)

	verificationCode := GenerateVerificationCode(6)

	db := dynamodb.New(sess)
	item := Verification{
		Email: "gopankov.aa@gmail.com",
		Code:  verificationCode,
	}
	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		log.Fatalf("Got error marshalling map: %s", err)
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String("users"),
	}

	_, err = db.PutItem(input)
	if err != nil {
		log.Fatalf("Got error calling PutItem: %s", err)
	}

	subject := "Your verification code"
	body := "Your verification code is: " + verificationCode
	msg := &ses.Message{
		Body: &ses.Body{
			Text: &ses.Content{
				Charset: aws.String("UTF-8"),
				Data:    aws.String(body),
			},
		},
		Subject: &ses.Content{
			Charset: aws.String("UTF-8"),
			Data:    aws.String(subject),
		},
	}

	_, err = svc.SendEmail(&ses.SendEmailInput{
		Source:      aws.String("notification.service.crypto@gmail.com"),
		Destination: &ses.Destination{ToAddresses: []*string{aws.String(emailAddress)}},
		Message:     msg,
	})
	if err != nil {
		log.Fatalf("Error sending email: %s", err)
	}
}

func VerifyCode(sess *session.Session, emailAddress string, code string) bool {
	db := dynamodb.New(sess)

	result, err := db.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String("users"),
		Key: map[string]*dynamodb.AttributeValue{
			"Email": {
				S: aws.String(emailAddress),
			},
		},
	})
	if err != nil {
		log.Fatalf("Error occurred while fetching data from DynamoDB %v", err)
	}

	item := Verification{}
	err = dynamodbattribute.UnmarshalMap(result.Item, &item)
	if err != nil {
		log.Fatalf("Error occurred while unmarshalling data %v", err)
	}

	_, err = db.DeleteItem(&dynamodb.DeleteItemInput{
		TableName: aws.String("users"),
		Key: map[string]*dynamodb.AttributeValue{
			"Email": {
				S: aws.String(emailAddress),
			},
		},
	})

	if err != nil {
		log.Fatalf("Got error calling DeleteItem: %s", err)
	}

	if code == item.Code {
		return true
	} else {
		return false
	}
}
