package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gocolly/colly/v2"
)

// ASUParser структура для парсера личного кабинета АСУ
type ASUParser struct {
	collector *colly.Collector
	baseURL   string
}

// NewASUParser создает новый экземпляр парсера
func NewASUParser() *ASUParser {
	c := colly.NewCollector(
		colly.AllowedDomains("lk.asu.ru", "personal-cabinet.asu.ru"),
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)

	return &ASUParser{
		collector: c,
		baseURL:   "https://lk.asu.ru",
	}
}

// Login авторизация в личном кабинете
func (p *ASUParser) Login(username, password string) (sessionToken string, name string, surname string, group string, err error) {
	var sessionId string
	var fullName string
	var groupName string

	// Устанавливаем обработчики событий
	p.collector.OnResponse(func(r *colly.Response) {
		// Пытаемся извлечь session token из cookies
		for _, cookie := range r.Request.Cookies() {
			if cookie.Name == "PHPSESSID" || strings.Contains(cookie.Name, "session") {
				sessionId = cookie.Value
			}
		}
	})

	// Обработка HTML-ответа после авторизации
	p.collector.OnHTML("body", func(e *colly.HTMLElement) {
		// Поиск информации о пользователе на странице
		e.ForEach("div.header-user-name div", func(_ int, el *colly.HTMLElement) {
			if fullName == "" {
				fullName = el.Text
			} else {
				fullName += " " + el.Text
			}
		})

		// Поиск информации о группе
		e.ForEach("td:contains('Зачетная книжка'), td:contains('группа')", func(_ int, el *colly.HTMLElement) {
			nextTd := el.DOM.Next()
			if nextTd != nil {
				groupName = strings.TrimSpace(nextTd.Text())
			}
		})
	})

	// Отправляем форму авторизации
	err = p.collector.Post(p.baseURL+"/", map[string]string{
		"form_action":    "auth_login",
		"auth_user_name": username,
		"auth_user_pass": password,
	})

	if err != nil {
		return "", "", "", "", fmt.Errorf("ошибка при попытке авторизации: %v", err)
	}

	// Разбор полного имени
	nameParts := strings.Split(fullName, " ")
	var fName, lName string
	if len(nameParts) >= 2 {
		lName = nameParts[0]
		fName = nameParts[1]
		if len(nameParts) > 2 {
			fName += " " + nameParts[2] // отчество
		}
	}

	return sessionId, fName, lName, groupName, nil
}

// GetSchedule получение расписания
func (p *ASUParser) GetSchedule(sessionToken string) ([]ScheduleItem, error) {
	var schedule []ScheduleItem

	// Устанавливаем сессию в заголовках
	p.collector.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Cookie", fmt.Sprintf("PHPSESSID=%s", sessionToken))
	})

	p.collector.OnHTML("div.timetable-day", func(e *colly.HTMLElement) {
		day := e.ChildText("div.timetable-day-header")
		
		e.ForEach("div.timetable-lesson", func(_ int, lessonEl *colly.HTMLElement) {
			time := lessonEl.ChildText("div.lesson-time")
			subject := lessonEl.ChildText("div.lesson-subject")
			teacher := lessonEl.ChildText("div.lesson-teacher")
			classroom := lessonEl.ChildText("div.lesson-classroom")
			
			// Разбор времени
			startTime, endTime := parseTime(time)
			
			schedule = append(schedule, ScheduleItem{
				DayOfWeek: day,
				TimeStart: startTime,
				TimeEnd:   endTime,
				Subject:   subject,
				Teacher:   teacher,
				Classroom: classroom,
				Type:      parseLessonType(subject),
			})
		})
	})

	err := p.collector.Visit(p.baseURL + "/timetable/")
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении расписания: %v", err)
	}

	return schedule, nil
}

// GetDebts получение задолженностей
func (p *ASUParser) GetDebts(sessionToken string) ([]DebtItem, error) {
	var debts []DebtItem

	// Устанавливаем сессию в заголовках
	p.collector.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Cookie", fmt.Sprintf("PHPSESSID=%s", sessionToken))
	})

	p.collector.OnHTML("div.debt-item, table.debt-table tr", func(e *colly.HTMLElement) {
		subject := e.ChildText("td.subject, div.debt-subject, .debt-item-subject")
		description := e.ChildText("td.description, div.debt-description, .debt-item-description")
		status := e.ChildText("td.status, div.debt-status, .debt-item-status")

		if subject != "" {
			debts = append(debts, DebtItem{
				Subject:     subject,
				Description: description,
				Status:      status,
			})
		}
	})

	// Попробуем разные возможные URL для задолженностей
	urls := []string{p.baseURL + "/student/debts/", p.baseURL + "/debts/", p.baseURL + "/academic-performance/"}
	
	for _, url := range urls {
		err := p.collector.Visit(url)
		if err == nil {
			break
		}
	}

	return debts, nil
}

// GetStudentInfo получение информации о студенте
func (p *ASUParser) GetStudentInfo(sessionToken string) (StudentInfo, error) {
	var info StudentInfo

	// Устанавливаем сессию в заголовках
	p.collector.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Cookie", fmt.Sprintf("PHPSESSID=%s", sessionToken))
	})

	p.collector.OnHTML("body", func(e *colly.HTMLElement) {
		// Ищем информацию о студенте
		info.FullName = e.ChildText("div.content-title")
		if info.FullName == "" {
			// Пробуем другой селектор
			info.FullName = e.ChildText("div.header-user-name div:first-child") + " " + e.ChildText("div.header-user-name div:last-child")
		}
		
		e.ForEach("tr", func(_ int, tr *colly.HTMLElement) {
			header := tr.ChildText("td:first-child")
			value := tr.ChildText("td:last-child")
			
			switch {
			case strings.Contains(header, "Форма обучения"):
				info.EducationForm = value
			case strings.Contains(header, "Зачетная книжка"):
				info.StudentCard = value
			case strings.Contains(header, "Учебное подразделение"):
				info.Department = value
			case strings.Contains(header, "Направление"):
				info.Direction = value
			case strings.Contains(header, "Профиль"):
				info.Profile = value
			}
		})
	})

	err := p.collector.Visit(p.baseURL + "/student/plan/")
	if err != nil {
		return info, fmt.Errorf("ошибка при получении информации о студенте: %v", err)
	}

	return info, nil
}

// parseTime разбор строки времени
func parseTime(timeStr string) (start, end string) {
	re := regexp.MustCompile(`(\d{1,2}:\d{2})\s*-\s*(\d{1,2}:\d{2})`)
	matches := re.FindStringSubmatch(timeStr)
	if len(matches) >= 3 {
		return matches[1], matches[2]
	}
	return "", ""
}

// parseLessonType определение типа занятия
func parseLessonType(subject string) string {
	subjectLower := strings.ToLower(subject)
	switch {
	case strings.Contains(subjectLower, "лек"):
		return "Лекция"
	case strings.Contains(subjectLower, "практ") || strings.Contains(subjectLower, "сем"):
		return "Практика"
	case strings.Contains(subjectLower, "лаб"):
		return "Лабораторная"
	default:
		return "Неизвестно"
	}
}

// ScheduleItem элемент расписания
type ScheduleItem struct {
	DayOfWeek string
	TimeStart string
	TimeEnd   string
	Subject   string
	Teacher   string
	Classroom string
	Type      string
}

// DebtItem элемент задолженности
type DebtItem struct {
	Subject     string
	Description string
	Status      string
}

// StudentInfo информация о студенте
type StudentInfo struct {
	FullName      string
	EducationForm string
	StudentCard   string
	Department    string
	Direction     string
	Profile       string
}