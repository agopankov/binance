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
	Email        string
	Code         string
	FirstBotID   int64
	SecondBotID  int64
	LastVerified time.Time
}

func GenerateVerificationCode(length int) string {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = CharSet[rand.Intn(len(CharSet))]
	}
	return string(b)
}

func SendVerificationEmail(sess *session.Session, emailAddress string, firstBotID int64, secondBotID int64) {
	svc := ses.New(sess)

	verificationCode := GenerateVerificationCode(6)

	db := dynamodb.New(sess)
	item := Verification{
		Email:       emailAddress,
		Code:        verificationCode,
		FirstBotID:  firstBotID,
		SecondBotID: secondBotID,
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

	if code == item.Code {
		_, err = db.UpdateItem(&dynamodb.UpdateItemInput{
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":lv": {
					S: aws.String(time.Now().Format(time.RFC3339)),
				},
			},
			TableName: aws.String("users"),
			Key: map[string]*dynamodb.AttributeValue{
				"Email": {
					S: aws.String(emailAddress),
				},
			},
			ReturnValues:     aws.String("UPDATED_NEW"),
			UpdateExpression: aws.String("set LastVerified = :lv"),
		})
		if err != nil {
			log.Fatalf("Got error updating LastVerified: %s", err)
		}
		return true
	} else {
		return false
	}
}

func ShouldSendVerificationEmail(sess *session.Session, emailAddress string) bool {
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

	if item.LastVerified.IsZero() || time.Since(item.LastVerified) > 24*time.Hour {
		return true
	}

	return false
}
