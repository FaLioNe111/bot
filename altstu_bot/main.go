package main

import (
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"altstu_bot/database"
	"altstu_bot/handlers"
)

func main() {
	// Get bot token from environment variable
	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		log.Fatal("BOT_TOKEN environment variable is required")
	}

	// Initialize database connection
	err := database.InitDB("./bot_database.db")
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer database.CloseDB()

	// Create bot instance
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatal("Failed to create bot:", err)
	}

	// Set debug mode
	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Configure updates channel
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	// Handle incoming updates
	for update := range updates {
		if update.Message != nil { // Check if it's a message
			handleMessage(bot, update.Message)
		}
	}
}

func handleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	log.Printf("[%s] %s", message.From.UserName, message.Text)

	// TODO: Implement command handling logic
	switch message.Command() {
	case "start":
		handlers.HandleStart(bot, nil, message)
	case "login":
		handlers.HandleLogin(bot, message)
	case "schedule":
		handlers.HandleSchedule(bot, message)
	case "debt":
		handlers.HandleDebt(bot, message)
	case "report_class_teacher":
		handlers.HandleReportClassTeacher(bot, message)
	case "report_lecturer":
		handlers.HandleReportLecturer(bot, message)
	default:
		replyText := "Я не понимаю эту команду. Используйте /start чтобы увидеть доступные команды."
		msg := tgbotapi.NewMessage(message.Chat.ID, replyText)
		bot.Send(msg)
	}
}