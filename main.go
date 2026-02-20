package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gocolly/colly/v2"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID       int
	ChatID   int64
	Username string
	Name     string
	Surname  string
	Group    string
	Role     string // student, teacher, curator
	Password string
	Session  string
}

type Bot struct {
	bot      *tgbotapi.BotAPI
	db       *sql.DB
	collector *colly.Collector
}

func NewBot(token string) (*Bot, error) {
	db, err := sql.Open("sqlite3", "./asu_bot.db")
	if err != nil {
		return nil, err
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	// Initialize database tables
	err = initDB(db)
	if err != nil {
		return nil, err
	}

	// Initialize web collector
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
		colly.AllowURLRevisit(),
	)

	return &Bot{
		bot:       bot,
		db:        db,
		collector: c,
	}, nil
}

func initDB(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		chat_id INTEGER UNIQUE,
		username TEXT,
		name TEXT,
		surname TEXT,
		group_name TEXT,
		role TEXT DEFAULT 'student',
		password TEXT,
		session TEXT
	);
	
	CREATE TABLE IF NOT EXISTS schedule (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		day_of_week TEXT,
		time_start TEXT,
		time_end TEXT,
		subject TEXT,
		teacher TEXT,
		classroom TEXT,
		type TEXT,
		FOREIGN KEY(user_id) REFERENCES users(id)
	);
	
	CREATE TABLE IF NOT EXISTS debts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		subject TEXT,
		description TEXT,
		status TEXT,
		FOREIGN KEY(user_id) REFERENCES users(id)
	);
	
	CREATE INDEX IF NOT EXISTS idx_users_chat_id ON users(chat_id);
	CREATE INDEX IF NOT EXISTS idx_schedule_user_id ON schedule(user_id);
	CREATE INDEX IF NOT EXISTS idx_debts_user_id ON debts(user_id);
	`

	_, err := db.Exec(query)
	return err
}

func (b *Bot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.bot.GetUpdatesChan(u)

	log.Printf("Bot authorized on account %s", b.bot.Self.UserName)

	for update := range updates {
		if update.Message != nil {
			b.handleMessage(update.Message)
		} else if update.CallbackQuery != nil {
			b.handleCallback(update.CallbackQuery)
		}
	}
}

func (b *Bot) handleMessage(msg *tgbotapi.Message) {
	if msg.IsCommand() {
		switch msg.Command() {
		case "start":
			b.startCommand(msg)
		case "login":
			b.loginCommand(msg)
		case "schedule":
			b.scheduleCommand(msg)
		case "debts":
			b.debtsCommand(msg)
		case "report_curator":
			b.reportCuratorCommand(msg)
		case "report_teacher":
			b.reportTeacherCommand(msg)
		default:
			b.sendTextMessage(msg.Chat.ID, "Неизвестная команда. Доступные команды:\n/start - начать работу с ботом\n/login - войти в личный кабинет\n/schedule - получить расписание\n/debts - проверить задолженности\n/report_curator - отчет для классного руководителя\n/report_teacher - отчет для преподавателя")
		}
	} else {
		b.handleTextMessage(msg)
	}
}

func (b *Bot) handleTextMessage(msg *tgbotapi.Message) {
	// Обработка текстовых сообщений, например, ответов на запросы логина
	text := strings.TrimSpace(msg.Text)
	chatID := msg.Chat.ID

	// Проверяем, есть ли ожидающие команды для этого чата
	var pendingCommand string
	err := b.db.QueryRow("SELECT command FROM pending_commands WHERE chat_id = ?", chatID).Scan(&pendingCommand)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error checking pending command: %v", err)
		return
	}

	if pendingCommand == "login_username" {
		// Сохраняем введенный логин и запрашиваем пароль
		b.setPendingData(chatID, "login_username_val", text)
		b.sendTextMessage(chatID, "Введите пароль:")
		b.setPendingCommand(chatID, "login_password")
	} else if pendingCommand == "login_password" {
		// Получаем сохраненный логин и текущий пароль
		var username string
		err := b.db.QueryRow("SELECT data_value FROM pending_data WHERE chat_id = ? AND data_key = 'login_username_val'", chatID).Scan(&username)
		if err != nil {
			b.sendTextMessage(chatID, "Произошла ошибка. Начните авторизацию заново.")
			return
		}

		// Попытка входа в личный кабинет
		session, name, surname, group, err := b.loginToPersonalAccount(username, text)
		if err != nil {
			b.sendTextMessage(chatID, fmt.Sprintf("Ошибка авторизации: %v", err))
		} else {
			// Сохраняем информацию о пользователе в базу данных
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(text), bcrypt.DefaultCost)
			if err != nil {
				b.sendTextMessage(chatID, "Произошла ошибка при сохранении данных.")
				return
			}

			_, err = b.db.Exec("INSERT OR REPLACE INTO users (chat_id, username, name, surname, group_name, password, session) VALUES (?, ?, ?, ?, ?, ?, ?)",
				chatID, username, name, surname, group, string(hashedPassword), session)
			if err != nil {
				b.sendTextMessage(chatID, "Произошла ошибка при сохранении данных.")
				return
			}

			b.clearPendingData(chatID)
			b.sendTextMessage(chatID, "Вы успешно вошли в систему!")
		}
	} else {
		b.sendTextMessage(chatID, "Для получения информации используйте команды бота. Введите /help для списка команд.")
	}
}

func (b *Bot) loginToPersonalAccount(username, password string) (string, string, string, string, error) {
	// Проверяем корректность ввода
	err := validateUserInput(username, password)
	if err != nil {
		return "", "", "", "", err
	}

	// Санитизируем ввод
	safeUsername := sanitizeInput(username)
	safePassword := sanitizeInput(password)

	// Создаем парсер и выполняем авторизацию
	parser := NewASUParser()
	session, name, surname, group, err := parser.Login(safeUsername, safePassword)
	if err != nil {
		return "", "", "", "", err
	}

	return session, name, surname, group, nil
}

func (b *Bot) startCommand(msg *tgbotapi.Message) {
	text := "Привет! Я бот для личного кабинета Алтайского государственного университета.\n\n" +
		"Доступные команды:\n" +
		"/login - войти в личный кабинет\n" +
		"/schedule - получить расписание\n" +
		"/debts - проверить задолженности\n" +
		"/report_curator - отчет для классного руководителя\n" +
		"/report_teacher - отчет для преподавателя\n" +
		"/help - список команд"

	b.sendTextMessage(msg.Chat.ID, text)
}

func (b *Bot) loginCommand(msg *tgbotapi.Message) {
	b.sendTextMessage(msg.Chat.ID, "Введите имя пользователя (логин):")
	b.setPendingCommand(msg.Chat.ID, "login_username")
}

func (b *Bot) scheduleCommand(msg *tgbotapi.Message) {
	// Проверяем, авторизован ли пользователь
	user, err := b.getUserByChatID(msg.Chat.ID)
	if err != nil {
		b.sendTextMessage(msg.Chat.ID, "Сначала выполните вход в систему. Используйте команду /login")
		return
	}

	// Получаем расписание через парсер
	schedule, err := b.getSchedule(user.Session)
	if err != nil {
		b.sendTextMessage(msg.Chat.ID, fmt.Sprintf("Ошибка получения расписания: %v", err))
		return
	}

	// Отправляем расписание пользователю
	b.sendTextMessage(msg.Chat.ID, schedule)
}

func (b *Bot) debtsCommand(msg *tgbotapi.Message) {
	// Проверяем, авторизован ли пользователь
	user, err := b.getUserByChatID(msg.Chat.ID)
	if err != nil {
		b.sendTextMessage(msg.Chat.ID, "Сначала выполните вход в систему. Используйте команду /login")
		return
	}

	// Получаем задолженности через парсер
	debts, err := b.getDebts(user.Session)
	if err != nil {
		b.sendTextMessage(msg.Chat.ID, fmt.Sprintf("Ошибка получения задолженностей: %v", err))
		return
	}

	// Отправляем задолженности пользователю
	b.sendTextMessage(msg.Chat.ID, debts)
}

func (b *Bot) reportCuratorCommand(msg *tgbotapi.Message) {
	// Проверяем, авторизован ли пользователь
	user, err := b.getUserByChatID(msg.Chat.ID)
	if err != nil {
		b.sendTextMessage(msg.Chat.ID, "Сначала выполните вход в систему. Используйте команду /login")
		return
	}

	// Формируем отчет для классного руководителя
	report, err := b.generateCuratorReport(user)
	if err != nil {
		b.sendTextMessage(msg.Chat.ID, fmt.Sprintf("Ошибка формирования отчета: %v", err))
		return
	}

	b.sendTextMessage(msg.Chat.ID, report)
}

func (b *Bot) reportTeacherCommand(msg *tgbotapi.Message) {
	// Проверяем, авторизован ли пользователь
	user, err := b.getUserByChatID(msg.Chat.ID)
	if err != nil {
		b.sendTextMessage(msg.Chat.ID, "Сначала выполните вход в систему. Используйте команду /login")
		return
	}

	// Формируем отчет для преподавателя
	report, err := b.generateTeacherReport(user)
	if err != nil {
		b.sendTextMessage(msg.Chat.ID, fmt.Sprintf("Ошибка формирования отчета: %v", err))
		return
	}

	b.sendTextMessage(msg.Chat.ID, report)
}

func (b *Bot) getSchedule(session string) (string, error) {
	// Создаем парсер и получаем расписание
	parser := NewASUParser()
	scheduleItems, err := parser.GetSchedule(session)
	if err != nil {
		return "", err
	}

	if len(scheduleItems) == 0 {
		return "Расписание не найдено.", nil
	}

	result := "Ваше расписание:\n"
	currentDay := ""
	for _, item := range scheduleItems {
		if item.DayOfWeek != currentDay {
			currentDay = item.DayOfWeek
			result += fmt.Sprintf("\n<b>%s:</b>\n", currentDay)
		}
		result += fmt.Sprintf("%s-%s %s (%s) - %s, %s\n", 
			item.TimeStart, item.TimeEnd, 
			item.Subject, item.Type, 
			item.Teacher, item.Classroom)
	}

	return result, nil
}

func (b *Bot) getDebts(session string) (string, error) {
	// Создаем парсер и получаем задолженности
	parser := NewASUParser()
	debtItems, err := parser.GetDebts(session)
	if err != nil {
		return "", err
	}

	if len(debtItems) == 0 {
		return "У вас нет задолженностей.", nil
	}

	result := "Ваши задолженности:\n"
	for i, debt := range debtItems {
		status := debt.Status
		if status == "" {
			status = "Не указан статус"
		}
		result += fmt.Sprintf("%d. %s - %s (%s)\n", i+1, debt.Subject, debt.Description, status)
	}

	return result, nil
}

func (b *Bot) generateCuratorReport(user *User) (string, error) {
	// Получаем дополнительную информацию о студенте
	parser := NewASUParser()
	studentInfo, err := parser.GetStudentInfo(user.Session)
	if err != nil {
		// Если не удалось получить информацию, используем имеющуюся
		studentInfo.FullName = fmt.Sprintf("%s %s", user.Name, user.Surname)
	}

	report := fmt.Sprintf(
		"<b>Отчет для классного руководителя</b>\n"+
			"<b>ФИО:</b> %s\n"+
			"<b>Группа:</b> %s\n"+
			"<b>Статус:</b> %s\n"+
			"<b>Форма обучения:</b> %s\n"+
			"<b>Зачетная книжка:</b> %s\n"+
			"<b>Подразделение:</b> %s\n"+
			"<b>Направление:</b> %s\n",
		studentInfo.FullName, user.Group, user.Role, 
		studentInfo.EducationForm, studentInfo.StudentCard, 
		studentInfo.Department, studentInfo.Direction)
	
	return report, nil
}

func (b *Bot) generateTeacherReport(user *User) (string, error) {
	// Получаем дополнительную информацию о студенте
	parser := NewASUParser()
	studentInfo, err := parser.GetStudentInfo(user.Session)
	if err != nil {
		// Если не удалось получить информацию, используем имеющуюся
		studentInfo.FullName = fmt.Sprintf("%s %s", user.Name, user.Surname)
	}

	// Получаем задолженности студента
	debtItems, err := parser.GetDebts(user.Session)
	if err != nil {
		// Если не удалось получить задолженности, продолжаем без них
		debtItems = []DebtItem{}
	}

	report := fmt.Sprintf(
		"<b>Отчет для преподавателя</b>\n"+
			"<b>ФИО:</b> %s\n"+
			"<b>Группа:</b> %s\n"+
			"<b>Статус:</b> %s\n"+
			"<b>Форма обучения:</b> %s\n"+
			"<b>Направление:</b> %s\n"+
			"<b>Профиль:</b> %s\n"+
			"<b>Задолженности:</b> %d шт.\n",
		studentInfo.FullName, user.Group, user.Role, 
		studentInfo.EducationForm, studentInfo.Direction, 
		studentInfo.Profile, len(debtItems))
	
	return report, nil
}

func (b *Bot) getUserByChatID(chatID int64) (*User, error) {
	var user User
	err := b.db.QueryRow("SELECT id, chat_id, username, name, surname, group_name, role, session FROM users WHERE chat_id = ?", chatID).
		Scan(&user.ID, &user.ChatID, &user.Username, &user.Name, &user.Surname, &user.Group, &user.Role, &user.Session)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (b *Bot) setPendingCommand(chatID int64, command string) {
	b.db.Exec("INSERT OR REPLACE INTO pending_commands (chat_id, command) VALUES (?, ?)", chatID, command)
}

func (b *Bot) setPendingData(chatID int64, key, value string) {
	b.db.Exec("INSERT OR REPLACE INTO pending_data (chat_id, data_key, data_value) VALUES (?, ?, ?)", chatID, key, value)
}

func (b *Bot) clearPendingData(chatID int64) {
	b.db.Exec("DELETE FROM pending_commands WHERE chat_id = ?", chatID)
	b.db.Exec("DELETE FROM pending_data WHERE chat_id = ?", chatID)
}

func (b *Bot) sendTextMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	b.bot.Send(msg)
}

func (b *Bot) handleCallback(callback *tgbotapi.CallbackQuery) {
	// Обработка callback-запросов
}

func main() {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable is required")
	}

	bot, err := NewBot(token)
	if err != nil {
		log.Fatal(err)
	}

	// Создаем таблицы для хранения ожидающих команд
	_, err = bot.db.Exec(`
		CREATE TABLE IF NOT EXISTS pending_commands (
			chat_id INTEGER PRIMARY KEY,
			command TEXT
		);
		
		CREATE TABLE IF NOT EXISTS pending_data (
			chat_id INTEGER,
			data_key TEXT,
			data_value TEXT,
			PRIMARY KEY (chat_id, data_key)
		);
	`)
	if err != nil {
		log.Fatal(err)
	}

	bot.Start()
}