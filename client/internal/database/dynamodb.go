package database

import (
	"github.com/agopankov/imPulse/client/internal/emailsender"
	"github.com/agopankov/imPulse/client/internal/emailverify"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"log"
	"time"
)

type DynamoDB struct{}

func (d *DynamoDB) SendVerificationEmail(emailAddress string, firstBotID int64, secondBotID int64, postmarkToken string) {
	sess := sess()
	verificationCode := emailverify.GenerateVerificationCode(6)

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

	sender := emailsender.NewEmailSender(postmarkToken)
	sender.SendEmail(emailAddress, "Your verification code", "Your verification code is: "+verificationCode)
}

func (d *DynamoDB) VerifyCode(emailAddress string, code string) bool {
	sess := sess()
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

func (d *DynamoDB) ShouldSendVerificationEmail(emailAddress string) bool {
	sess := sess()
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

func sess() *session.Session {
	return session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
}

func (d *DynamoDB) GetAllUsers() ([]Verification, error) {
	sess := sess()
	db := dynamodb.New(sess)

	input := &dynamodb.ScanInput{
		TableName: aws.String("users"),
	}
	result, err := db.Scan(input)
	if err != nil {
		return nil, err
	}

	var users []Verification
	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &users)
	if err != nil {
		return nil, err
	}

	return users, nil
}
