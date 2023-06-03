package secrets

import (
	"encoding/json"
	"os"
)

type SecretKeys struct {
	TelegramBotToken       string `json:"TELEGRAM_BOT_TOKEN"`
	TelegramBotTokenSecond string `json:"TELEGRAM_BOT_TOKEN_SECOND"`
	PostmarkToken          string `json:"POSTMARK_TOKEN"`
}

func LoadSecrets() (*SecretKeys, error) {
	var secrets SecretKeys

	firstBotToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	secondBotToken := os.Getenv("TELEGRAM_BOT_TOKEN_SECOND")
	postmarkToken := os.Getenv("POSTMARK_TOKEN")

	if firstBotToken == "" || secondBotToken == "" {
		secretsFile, err := os.ReadFile("/mnt/secrets-store/prod_binance_secret")
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(secretsFile, &secrets)
		if err != nil {
			return nil, err
		}
	} else {
		secrets = SecretKeys{
			TelegramBotToken:       firstBotToken,
			TelegramBotTokenSecond: secondBotToken,
			PostmarkToken:          postmarkToken,
		}
	}
	return &secrets, nil
}
