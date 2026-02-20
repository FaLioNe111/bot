-- Database schema for Altai State University Telegram Bot

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