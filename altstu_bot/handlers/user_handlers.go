package handlers

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"altstu_bot/database"
	"altstu_bot/parsers"
	"altstu_bot/utils"
)

// HandleStart handles the /start command
func HandleStart(bot *tgbotapi.BotAPI, db *database.User, message *tgbotapi.Message) {
	welcomeText := `
	Добро пожаловать в Telegram-бот Алтайского Государственного Университета!
	
	Доступные команды:
	/schedule - Посмотреть ваше расписание занятий
	/debt - Проверить академическую задолженность
	/report_class_teacher - Сформировать отчет для классного руководителя
	/report_lecturer - Сформировать отчет для преподавателя
	
	Для начала работы, пожалуйста, авторизуйтесь с помощью команды /login
	`

	msg := tgbotapi.NewMessage(message.Chat.ID, welcomeText)
	bot.Send(msg)
}

// HandleLogin handles the /login command
func HandleLogin(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	args := strings.Fields(message.Text)
	if len(args) != 3 { // Includes the command itself
		replyText := "Используйте команду в формате: /login <логин> <пароль>"
		msg := tgbotapi.NewMessage(message.Chat.ID, replyText)
		bot.Send(msg)
		return
	}

	login := args[1]
	password := args[2]

	// Validate inputs
	if !utils.ValidateInput(login) || !utils.ValidateInput(password) {
		utils.LogSecurityEvent(&message.From.ID, "invalid_login_input", "", message.From.String(), true)
		replyText := "Недопустимые символы в логине или пароле"
		msg := tgbotapi.NewMessage(message.Chat.ID, replyText)
		bot.Send(msg)
		return
	}

	// Check rate limiting
	if utils.IsRateLimited(message.From.ID, "login", 5, time.Minute*10) {
		utils.LogSecurityEvent(&message.From.ID, "rate_limited_login", "", message.From.String(), true)
		replyText := "Слишком много попыток входа. Попробуйте позже."
		msg := tgbotapi.NewMessage(message.Chat.ID, replyText)
		bot.Send(msg)
		return
	}

	// Parse personal cabinet data
	data, err := parsers.ParsePersonalCabinet(login, password)
	if err != nil {
		utils.LogSecurityEvent(&message.From.ID, "login_failed", "", message.From.String(), false)
		replyText := fmt.Sprintf("Ошибка при входе: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, replyText)
		bot.Send(msg)
		return
	}

	// Save parsed data to database
	err = parsers.SaveParsedData(message.From.ID, data)
	if err != nil {
		replyText := fmt.Sprintf("Ошибка при сохранении данных: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, replyText)
		bot.Send(msg)
		return
	}

	utils.LogSecurityEvent(&message.From.ID, "login_success", "", message.From.String(), false)
	replyText := "Успешный вход! Ваши данные обновлены."
	msg := tgbotapi.NewMessage(message.Chat.ID, replyText)
	bot.Send(msg)
}

// HandleSchedule handles the /schedule command
func HandleSchedule(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	// Check if user is allowed to access this feature
	allowed, err := utils.CheckUserAccess(message.From.ID)
	if err != nil || !allowed {
		replyText := "Доступ запрещен из-за подозрительной активности. Пожалуйста, свяжитесь с администратором."
		msg := tgbotapi.NewMessage(message.Chat.ID, replyText)
		bot.Send(msg)
		return
	}

	// Get schedule for next 7 days
	startDate := time.Now()
	endDate := startDate.AddDate(0, 0, 7)

	scheduleItems, err := parsers.GetSchedule(message.From.ID, startDate, endDate)
	if err != nil {
		log.Printf("Error getting schedule for user %d: %v", message.From.ID, err)
		replyText := "Ошибка при получении расписания"
		msg := tgbotapi.NewMessage(message.Chat.ID, replyText)
		bot.Send(msg)
		return
	}

	if len(scheduleItems) == 0 {
		replyText := "Расписание на ближайшие 7 дней не найдено. Возможно, вам нужно сначала авторизоваться с помощью /login"
		msg := tgbotapi.NewMessage(message.Chat.ID, replyText)
		bot.Send(msg)
		return
	}

	// Format schedule
	var scheduleText strings.Builder
	scheduleText.WriteString("Ваше расписание на ближайшие 7 дней:\n\n")

	currentDate := time.Time{}
	for _, item := range scheduleItems {
		itemDate := item.Date
		if currentDate.Day() != itemDate.Day() || currentDate.Month() != itemDate.Month() {
			currentDate = itemDate
			scheduleText.WriteString(fmt.Sprintf("<b>%s (%02d.%02d)</b>\n", 
				getDayOfWeek(itemDate.Weekday()), 
				itemDate.Day(), 
				itemDate.Month()))
		}

		startTime := item.StartTime
		endTime := item.EndTime
		scheduleText.WriteString(
			fmt.Sprintf("%02d:%02d-%02d:%02d | %s | %s | %s\n", 
				startTime.Hour(), startTime.Minute(),
				endTime.Hour(), endTime.Minute(),
				item.SubjectName,
				item.TeacherName,
				item.Classroom))
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, scheduleText.String())
	msg.ParseMode = "HTML"
	bot.Send(msg)
}

// HandleDebt handles the /debt command
func HandleDebt(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	// Check if user is allowed to access this feature
	allowed, err := utils.CheckUserAccess(message.From.ID)
	if err != nil || !allowed {
		replyText := "Доступ запрещен из-подозрительной активности. Пожалуйста, свяжитесь с администратором."
		msg := tgbotapi.NewMessage(message.Chat.ID, replyText)
		bot.Send(msg)
		return
	}

	debts, err := parsers.GetDebts(message.From.ID)
	if err != nil {
		log.Printf("Error getting debts for user %d: %v", message.From.ID, err)
		replyText := "Ошибка при получении информации о задолженностях"
		msg := tgbotapi.NewMessage(message.Chat.ID, replyText)
		bot.Send(msg)
		return
	}

	var debtText string
	if len(debts) == 0 {
		debtText = "У вас нет академических задолженностей!"
	} else {
		debtText = "Ваши академические задолженности:\n\n"
		for i, debt := range debts {
			debtText += fmt.Sprintf("%d. %s - %s (Статус: %s)\n", 
				i+1, debt.SubjectName, debt.Description, debt.Status)
		}
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, debtText)
	bot.Send(msg)
}

// HandleReportClassTeacher handles the /report_class_teacher command
func HandleReportClassTeacher(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	// Check if user is allowed to access this feature
	allowed, err := utils.CheckUserAccess(message.From.ID)
	if err != nil || !allowed {
		replyText := "Доступ запрещен из-за подозрительной активности. Пожалуйста, свяжитесь с администратором."
		msg := tgbotapi.NewMessage(message.Chat.ID, replyText)
		bot.Send(msg)
		return
	}

	// In a real implementation, this would generate a report for the class teacher
	// For now, we'll just return mock data
	user, err := database.GetUserByTelegramID(message.From.ID)
	if err != nil {
		replyText := "Ошибка при получении данных пользователя. Пожалуйста, сначала авторизуйтесь с помощью /login"
		msg := tgbotapi.NewMessage(message.Chat.ID, replyText)
		bot.Send(msg)
		return
	}

	var reportText strings.Builder
	reportText.WriteString("Отчет для классного руководителя:\n\n")
	reportText.WriteString(fmt.Sprintf("Студент: %s %s\n", user.FirstName.String, user.LastName.String))
	reportText.WriteString(fmt.Sprintf("Студенческий ID: %s\n", user.StudentID.String))
	reportText.WriteString(fmt.Sprintf("Группа: %s\n", user.GroupName.String))

	// Add schedule for today
	today := time.Now()
	tomorrow := today.AddDate(0, 0, 1)
	todaysSchedule, err := parsers.GetSchedule(message.From.ID, today, tomorrow)
	if err == nil && len(todaysSchedule) > 0 {
		reportText.WriteString("\nЗанятия на сегодня:\n")
		for _, item := range todaysSchedule {
			startTime := item.StartTime
			reportText.WriteString(
				fmt.Sprintf("- %02d:%02d %s (%s)\n", 
					startTime.Hour(), startTime.Minute(),
					item.SubjectName,
					item.TeacherName))
		}
	}

	// Add debt information
	debts, err := parsers.GetDebts(message.From.ID)
	if err == nil && len(debts) > 0 {
		reportText.WriteString("\nАкадемические задолженности:\n")
		for _, debt := range debts {
			reportText.WriteString(fmt.Sprintf("- %s: %s\n", debt.SubjectName, debt.Description))
		}
	} else {
		reportText.WriteString("\nАкадемических задолженностей нет.\n")
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, reportText.String())
	bot.Send(msg)
}

// HandleReportLecturer handles the /report_lecturer command
func HandleReportLecturer(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	// Check if user is allowed to access this feature
	allowed, err := utils.CheckUserAccess(message.From.ID)
	if err != nil || !allowed {
		replyText := "Доступ запрещен из-за подозрительной активности. Пожалуйста, свяжитесь с администратором."
		msg := tgbotapi.NewMessage(message.Chat.ID, replyText)
		bot.Send(msg)
		return
	}

	// In a real implementation, this would generate a report for the lecturer
	// For now, we'll just return mock data
	user, err := database.GetUserByTelegramID(message.From.ID)
	if err != nil {
		replyText := "Ошибка при получении данных пользователя. Пожалуйста, сначала авторизуйтесь с помощью /login"
		msg := tgbotapi.NewMessage(message.Chat.ID, replyText)
		bot.Send(msg)
		return
	}

	var reportText strings.Builder
	reportText.WriteString("Отчет для преподавателя:\n\n")
	reportText.WriteString(fmt.Sprintf("Студент: %s %s\n", user.FirstName.String, user.LastName.String))
	reportText.WriteString(fmt.Sprintf("Студенческий ID: %s\n", user.StudentID.String))
	reportText.WriteString(fmt.Sprintf("Группа: %s\n", user.GroupName.String))

	// Add current subject information
	now := time.Now()
	tomorrow := now.AddDate(0, 0, 1)
	todaysSchedule, err := parsers.GetSchedule(message.From.ID, now, tomorrow)
	if err == nil && len(todaysSchedule) > 0 {
		// Find current or next class
		for _, item := range todaysSchedule {
			classStart := time.Date(now.Year(), now.Month(), now.Day(), 
				item.StartTime.Hour(), item.StartTime.Minute(), 0, 0, now.Location())
			classEnd := time.Date(now.Year(), now.Month(), now.Day(), 
				item.EndTime.Hour(), item.EndTime.Minute(), 0, 0, now.Location())

			if now.After(classStart) && now.Before(classEnd) {
				reportText.WriteString(fmt.Sprintf("\nТекущая пара: %s (%s)\n", item.SubjectName, item.TeacherName))
				break
			} else if now.Before(classStart) {
				reportText.WriteString(fmt.Sprintf("\nБлижайшая пара: %s (%s) в %02d:%02d\n", 
					item.SubjectName, item.TeacherName, 
					item.StartTime.Hour(), item.StartTime.Minute()))
				break
			}
		}
	}

	// Add attendance information (mock)
	reportText.WriteString("\nПосещаемость: 85% за последний месяц\n")

	// Add academic performance (mock)
	reportText.WriteString("Успеваемость: Хорошо\n")

	msg := tgbotapi.NewMessage(message.Chat.ID, reportText.String())
	bot.Send(msg)
}

// Helper function to get day of week in Russian
func getDayOfWeek(day time.Weekday) string {
	switch day {
	case time.Monday:
		return "Понедельник"
	case time.Tuesday:
		return "Вторник"
	case time.Wednesday:
		return "Среда"
	case time.Thursday:
		return "Четверг"
	case time.Friday:
		return "Пятница"
	case time.Saturday:
		return "Суббота"
	case time.Sunday:
		return "Воскресенье"
	default:
		return day.String()
	}
}