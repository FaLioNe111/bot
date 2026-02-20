package parsers

import (
	"fmt"
	"log"
	"strings"
	"time"

	"altstu_bot/database"
)

// PersonalCabinetData represents parsed data from the personal cabinet
type PersonalCabinetData struct {
	Schedule []ScheduleItem
	Debts    []AcademicDebt
	Profile  ProfileInfo
}

// ScheduleItem represents a single schedule item
type ScheduleItem struct {
	SubjectName string
	TeacherName string
	Classroom   string
	Date        time.Time
	StartTime   time.Time
	EndTime     time.Time
	Type        string // lecture, practice, lab
	WeekType    string // upper, lower, both
}

// AcademicDebt represents an academic debt
type AcademicDebt struct {
	SubjectName     string
	Description     string
	Status          string // active, resolved, pending
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ProfileInfo represents user profile information
type ProfileInfo struct {
	StudentID string
	FullName  string
	Group     string
	Faculty   string
	Program   string
}

// ParsePersonalCabinet parses data from the personal cabinet
// This is a placeholder implementation - actual implementation will depend on the university's website structure
func ParsePersonalCabinet(login, password string) (*PersonalCabinetData, error) {
	log.Println("Starting personal cabinet parsing...")
	
	// TODO: Implement actual parsing logic based on the university's website
	// This would typically involve:
	// 1. Making HTTP requests to the login page
	// 2. Submitting login credentials
	// 3. Navigating to different sections (schedule, debts, etc.)
	// 4. Parsing HTML responses to extract relevant data
	
	// For now, return mock data
	data := &PersonalCabinetData{
		Schedule: []ScheduleItem{
			{
				SubjectName: "Mathematics",
				TeacherName: "Dr. Smith",
				Classroom:   "Room 201",
				Date:        time.Now(),
				StartTime:   time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC),
				EndTime:     time.Date(0, 1, 1, 10, 30, 0, 0, time.UTC),
				Type:        "lecture",
				WeekType:    "both",
			},
			{
				SubjectName: "Physics",
				TeacherName: "Prof. Johnson",
				Classroom:   "Lab 105",
				Date:        time.Now().AddDate(0, 0, 1),
				StartTime:   time.Date(0, 1, 1, 11, 0, 0, 0, time.UTC),
				EndTime:     time.Date(0, 1, 1, 12, 30, 0, 0, time.UTC),
				Type:        "lab",
				WeekType:    "upper",
			},
		},
		Debts: []AcademicDebt{
			{
				SubjectName: "Chemistry",
				Description: "Missed laboratory work #3",
				Status:      "active",
				CreatedAt:   time.Now().AddDate(0, 0, -10),
				UpdatedAt:   time.Now(),
			},
		},
		Profile: ProfileInfo{
			StudentID: "STU001234",
			FullName:  "Ivan Petrov",
			Group:     "CS-101",
			Faculty:   "Computer Science",
			Program:   "Software Engineering",
		},
	}
	
	log.Println("Personal cabinet parsing completed")
	return data, nil
}

// SaveParsedData saves parsed data to the database
func SaveParsedData(telegramID int64, data *PersonalCabinetData) error {
	// Update user profile info
	user := &database.User{
		TelegramID: telegramID,
		StudentID:  stringToNullString(data.Profile.StudentID),
		FirstName:  stringToNullString(strings.Split(data.Profile.FullName, " ")[0]),
		GroupName:  stringToNullString(data.Profile.Group),
	}
	
	err := database.InsertUser(user)
	if err != nil {
		return fmt.Errorf("failed to update user profile: %w", err)
	}
	
	// Clear existing schedule for this user and insert new data
	_, err = database.DB.Exec("DELETE FROM schedule WHERE telegram_id = ?", telegramID)
	if err != nil {
		return fmt.Errorf("failed to clear existing schedule: %w", err)
	}
	
	for _, item := range data.Schedule {
		_, err = database.DB.NamedExec(`
			INSERT INTO schedule (
				telegram_id, subject_name, teacher_name, classroom, 
				date, start_time, end_time, type, week_type
			) VALUES (
				:telegram_id, :subject_name, :teacher_name, :classroom,
				:date, :start_time, :end_time, :type, :week_type
			)
		`, map[string]interface{}{
			"telegram_id": telegramID,
			"subject_name": item.SubjectName,
			"teacher_name": item.TeacherName,
			"classroom": item.Classroom,
			"date": item.Date.Format("2006-01-02"),
			"start_time": item.StartTime.Format("15:04:05"),
			"end_time": item.EndTime.Format("15:04:05"),
			"type": item.Type,
			"week_type": item.WeekType,
		})
		
		if err != nil {
			return fmt.Errorf("failed to insert schedule item: %w", err)
		}
	}
	
	// Clear existing debts for this user and insert new data
	_, err = database.DB.Exec("DELETE FROM academic_debts WHERE telegram_id = ?", telegramID)
	if err != nil {
		return fmt.Errorf("failed to clear existing debts: %w", err)
	}
	
	for _, debt := range data.Debts {
		_, err = database.DB.NamedExec(`
			INSERT INTO academic_debts (
				telegram_id, subject_name, debt_description, status, created_at, updated_at
			) VALUES (
				:telegram_id, :subject_name, :debt_description, :status, :created_at, :updated_at
			)
		`, map[string]interface{}{
			"telegram_id": telegramID,
			"subject_name": debt.SubjectName,
			"debt_description": debt.Description,
			"status": debt.Status,
			"created_at": debt.CreatedAt,
			"updated_at": debt.UpdatedAt,
		})
		
		if err != nil {
			return fmt.Errorf("failed to insert debt item: %w", err)
		}
	}
	
	return nil
}

// GetSchedule retrieves schedule for a specific user and date range
func GetSchedule(telegramID int64, startDate, endDate time.Time) ([]ScheduleItem, error) {
	var items []ScheduleItem
	
	err := database.DB.Select(&items, `
		SELECT subject_name, teacher_name, classroom, date, start_time, end_time, type, week_type
		FROM schedule 
		WHERE telegram_id = ? AND date BETWEEN ? AND ?
		ORDER BY date, start_time
	`, telegramID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}
	
	return items, nil
}

// GetDebts retrieves academic debts for a specific user
func GetDebts(telegramID int64) ([]AcademicDebt, error) {
	var debts []AcademicDebt
	
	err := database.DB.Select(&debts, `
		SELECT subject_name, debt_description, status, created_at, updated_at
		FROM academic_debts 
		WHERE telegram_id = ?
		ORDER BY created_at DESC
	`, telegramID)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get debts: %w", err)
	}
	
	return debts, nil
}

// Helper function to convert string to sql.NullString
func stringToNullString(s string) database.sql.NullString {
	if s == "" {
		return database.sql.NullString{Valid: false}
	}
	return database.sql.NullString{String: s, Valid: true}
}