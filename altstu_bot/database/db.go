package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// DB represents the database connection
var DB *sqlx.DB

// InitDB initializes the database connection and creates tables
func InitDB(dataSourceName string) error {
	var err error
	DB, err = sqlx.Connect("sqlite3", dataSourceName)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set database connection pool settings
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(25)
	DB.SetConnMaxLifetime(5 * time.Minute)

	// Create tables
	err = createTables()
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	return nil
}

// createTables creates all required tables if they don't exist
func createTables() error {
	schemaSQL := `
	-- Users table to store registered users
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		telegram_id BIGINT UNIQUE NOT NULL,
		username TEXT,
		first_name TEXT,
		last_name TEXT,
		student_id TEXT,
		group_name TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- User sessions to store authentication data for personal cabinet access
	CREATE TABLE IF NOT EXISTS user_sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		telegram_id BIGINT NOT NULL,
		session_data TEXT, -- JSON encoded session data
		expires_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (telegram_id) REFERENCES users(telegram_id)
	);

	-- Academic debts table
	CREATE TABLE IF NOT EXISTS academic_debts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		telegram_id BIGINT NOT NULL,
		subject_name TEXT NOT NULL,
		debt_description TEXT,
		status TEXT DEFAULT 'active', -- active, resolved, pending
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (telegram_id) REFERENCES users(telegram_id)
	);

	-- Schedule table
	CREATE TABLE IF NOT EXISTS schedule (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		telegram_id BIGINT NOT NULL,
		subject_name TEXT NOT NULL,
		teacher_name TEXT,
		classroom TEXT,
		date DATE NOT NULL,
		start_time TIME NOT NULL,
		end_time TIME NOT NULL,
		type TEXT, -- lecture, practice, lab
		week_type TEXT, -- upper, lower, both
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (telegram_id) REFERENCES users(telegram_id)
	);

	-- Reports table for class teacher and lecturer reports
	CREATE TABLE IF NOT EXISTS reports (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		telegram_id BIGINT NOT NULL,
		report_type TEXT NOT NULL, -- class_teacher, lecturer
		report_data TEXT NOT NULL, -- JSON encoded report data
		created_by_telegram_id BIGINT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (telegram_id) REFERENCES users(telegram_id)
	);

	-- Security logs table to track suspicious activities
	CREATE TABLE IF NOT EXISTS security_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		telegram_id BIGINT,
		action TEXT NOT NULL,
		ip_address TEXT,
		user_agent TEXT,
		occurred_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		is_suspicious BOOLEAN DEFAULT FALSE
	);

	-- Create indexes for better performance
	CREATE INDEX IF NOT EXISTS idx_users_telegram_id ON users(telegram_id);
	CREATE INDEX IF NOT EXISTS idx_user_sessions_telegram_id ON user_sessions(telegram_id);
	CREATE INDEX IF NOT EXISTS idx_academic_debts_telegram_id ON academic_debts(telegram_id);
	CREATE INDEX IF NOT EXISTS idx_schedule_telegram_id_date ON schedule(telegram_id, date);
	CREATE INDEX IF NOT EXISTS idx_reports_telegram_id ON reports(telegram_id);
	CREATE INDEX IF NOT EXISTS idx_security_logs_telegram_id ON security_logs(telegram_id);
	CREATE INDEX IF NOT EXISTS idx_security_logs_occurred_at ON security_logs(occurred_at);
	`

	_, err := DB.Exec(schemaSQL)
	if err != nil {
		return fmt.Errorf("failed to execute schema SQL: %w", err)
	}

	return nil
}

// CloseDB closes the database connection
func CloseDB() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

// User represents a user in the system
type User struct {
	ID         int            `db:"id"`
	TelegramID int64          `db:"telegram_id"`
	Username   sql.NullString `db:"username"`
	FirstName  sql.NullString `db:"first_name"`
	LastName   sql.NullString `db:"last_name"`
	StudentID  sql.NullString `db:"student_id"`
	GroupName  sql.NullString `db:"group_name"`
	CreatedAt  time.Time      `db:"created_at"`
	UpdatedAt  time.Time      `db:"updated_at"`
}

// InsertUser inserts a new user into the database
func InsertUser(user *User) error {
	query := `
	INSERT INTO users (telegram_id, username, first_name, last_name, student_id, group_name, created_at, updated_at)
	VALUES (:telegram_id, :username, :first_name, :last_name, :student_id, :group_name, :created_at, :updated_at)
	ON CONFLICT(telegram_id) DO UPDATE SET
		username = excluded.username,
		first_name = excluded.first_name,
		last_name = excluded.last_name,
		student_id = excluded.student_id,
		group_name = excluded.group_name,
		updated_at = CURRENT_TIMESTAMP
	`

	_, err := DB.NamedExec(query, user)
	return err
}

// GetUserByTelegramID retrieves a user by their Telegram ID
func GetUserByTelegramID(telegramID int64) (*User, error) {
	var user User
	err := DB.Get(&user, "SELECT * FROM users WHERE telegram_id = ?", telegramID)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// LogSecurityEvent logs a security event
func LogSecurityEvent(telegramID *int64, action, ipAddress, userAgent string, isSuspicious bool) error {
	query := `
	INSERT INTO security_logs (telegram_id, action, ip_address, user_agent, is_suspicious)
	VALUES (?, ?, ?, ?, ?)
	`

	var tgid *int64
	if telegramID != nil {
		tgid = telegramID
	}

	_, err := DB.Exec(query, tgid, action, ipAddress, userAgent, isSuspicious)
	return err
}

// IsUserBlocked checks if a user is blocked based on security events
func IsUserBlocked(telegramID int64) (bool, error) {
	// For now, we'll check if there are too many suspicious events recently
	query := `
	SELECT COUNT(*) FROM security_logs 
	WHERE telegram_id = ? AND is_suspicious = TRUE AND occurred_at > datetime('now', '-1 hour')
	`

	var count int
	err := DB.Get(&count, query, telegramID)
	if err != nil {
		return false, err
	}

	// If more than 5 suspicious events in the last hour, consider user blocked
	return count > 5, nil
}