package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// generateSecureToken генерирует безопасный токен для сессии
func generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// hashPassword хэширует пароль
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// checkPassword проверяет, соответствует ли пароль хэшу
func checkPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// validateUserInput проверяет корректность пользовательского ввода
func validateUserInput(username, password string) error {
	// Проверяем длину логина и пароля
	if len(username) < 3 || len(username) > 50 {
		return fmt.Errorf("некорректная длина логина (должно быть от 3 до 50 символов)")
	}

	if len(password) < 6 || len(password) > 128 {
		return fmt.Errorf("некорректная длина пароля (должно быть от 6 до 128 символов)")
	}

	// Проверяем, содержит ли логин только допустимые символы
	for _, char := range username {
		if !isValidCharForUsername(char) {
			return fmt.Errorf("недопустимый символ в логине")
		}
	}

	return nil
}

// isValidCharForUsername проверяет, является ли символ допустимым для использования в логине
func isValidCharForUsername(char rune) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9') ||
		char == '.' || char == '_' || char == '-' || char == '@' || char == '\\'
}

// rateLimitChecker ограничитель частоты запросов
type rateLimitChecker struct {
	attempts map[string][]time.Time
}

// newRateLimitChecker создает новый ограничитель частоты запросов
func newRateLimitChecker() *rateLimitChecker {
	return &rateLimitChecker{
		attempts: make(map[string][]time.Time),
	}
}

// isAllowed проверяет, разрешен ли запрос для данного IP
func (r *rateLimitChecker) isAllowed(ip string) bool {
	now := time.Now()
	cutoff := now.Add(-time.Minute * 5) // ограничение на 5 минут

	// Очищаем старые попытки
	var validAttempts []time.Time
	for _, attempt := range r.attempts[ip] {
		if attempt.After(cutoff) {
			validAttempts = append(validAttempts, attempt)
		}
	}
	r.attempts[ip] = validAttempts

	// Если за последние 5 минут больше 10 попыток, блокируем
	if len(r.attempts[ip]) >= 10 {
		return false
	}

	// Добавляем новую попытку
	r.attempts[ip] = append(r.attempts[ip], now)
	return true
}

// sanitizeInput очищает пользовательский ввод от потенциально опасных символов
func sanitizeInput(input string) string {
	// Убираем теги HTML
	sanitized := stripHTMLTags(input)
	
	// Убираем потенциально опасные символы
	sanitized = removeDangerousChars(sanitized)
	
	return sanitized
}

// stripHTMLTags удаляет HTML-теги из строки
func stripHTMLTags(input string) string {
	// Это упрощенная реализация
	// В реальном приложении лучше использовать специализированную библиотеку
	result := ""
	inTag := false
	
	for _, char := range input {
		if char == '<' {
			inTag = true
			continue
		}
		if char == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result += string(char)
		}
	}
	
	return result
}

// removeDangerousChars удаляет потенциально опасные символы
func removeDangerousChars(input string) string {
	dangerousChars := []rune{'\'', '"', ';', '--', '/*', '*/', '<', '>', '&'}
	
	result := input
	for _, char := range dangerousChars {
		result = removeSubstring(result, string(char))
	}
	
	return result
}

// removeSubstring удаляет все вхождения подстроки из строки
func removeSubstring(str, substr string) string {
	result := ""
	i := 0
	for i < len(str) {
		if i <= len(str)-len(substr) && str[i:i+len(substr)] == substr {
			i += len(substr)
		} else {
			result += string(str[i])
			i++
		}
	}
	return result
}