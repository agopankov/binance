package servicerestartnotification

import (
	"log"

	"github.com/agopankov/imPulse/client/internal/database"
	"github.com/agopankov/imPulse/client/internal/telegram"
	tele "gopkg.in/telebot.v3"
)

func SendServiceRestartNotifications(db database.Database, telegramClient *telegram.Client, secondTelegramClient *telegram.Client) {
	usersFromDB, err := db.GetAllUsers()
	if err != nil {
		log.Fatalf("Failed to retrieve users: %v", err)
	}

	notificationMessage := "⛔️The service has been restarted.\nYou need to resend the /start command in each chatbot."

	for _, usr := range usersFromDB {
		botChat := &tele.User{ID: usr.FirstBotID}
		_, err = telegramClient.SendMessage(botChat, notificationMessage)
		if err != nil {
			log.Printf("Failed to send notification to user %v on first bot: %v", usr.Email, err)
		}

		_, err = secondTelegramClient.SendMessage(botChat, notificationMessage)
		if err != nil {
			log.Printf("Failed to send notification to user %v on second bot: %v", usr.Email, err)
		}
	}
}
