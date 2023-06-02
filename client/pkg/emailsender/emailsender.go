package emailsender

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
)

type PostmarkRequest struct {
	From          string `json:"From"`
	To            string `json:"To"`
	Subject       string `json:"Subject"`
	HtmlBody      string `json:"HtmlBody"`
	MessageStream string `json:"MessageStream"`
}

type EmailSender struct {
	serverToken string
}

func NewEmailSender(token string) *EmailSender {
	return &EmailSender{
		serverToken: token,
	}
}

func (e *EmailSender) SendEmail(emailAddress string, subject string, body string) {
	requestData := &PostmarkRequest{
		From:          "support@cryptocoinpulse.com",
		To:            emailAddress,
		Subject:       subject,
		HtmlBody:      "<p>" + body + "</p>",
		MessageStream: "notification",
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		log.Fatalf("Failed to marshal request data: %v", err)
		return
	}

	req, err := http.NewRequest("POST", "https://api.postmarkapp.com/email", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
		return
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Postmark-Server-Token", e.serverToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to send request: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Failed to send email, status: %v", resp.StatusCode)
	}
}
