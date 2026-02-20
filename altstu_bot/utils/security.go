package utils

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"altstu_bot/database"
)

// ValidateInput validates user input to prevent injection attacks
func ValidateInput(input string) bool {
	// Check for SQL injection patterns
	sqlPattern := `(?i)(union|select|insert|update|delete|drop|create|alter|exec|execute|--|/\*|\*/|xp_|sp_)`
	reSql := regexp.MustCompile(sqlPattern)
	if reSql.MatchString(input) {
		return false
	}

	// Check for script tags
	scriptPattern := `(?i)<script[^>]*>.*?</script>`
	reScript := regexp.MustCompile(scriptPattern)
	if reScript.MatchString(input) {
		return false
	}

	// Check for other potentially dangerous patterns
	dangerousPatterns := []string{
		`<iframe`,
		`javascript:`,
		`vbscript:`,
		`onload=`,
		`onerror=`,
		`onclick=`,
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(strings.ToLower(input), pattern) {
			return false
		}
	}

	return true
}

// LogSecurityEvent logs security events to the database
func LogSecurityEvent(telegramID *int64, action, ipAddress, userAgent string, isSuspicious bool) {
	err := database.LogSecurityEvent(telegramID, action, ipAddress, userAgent, isSuspicious)
	if err != nil {
		log.Printf("Error logging security event: %v", err)
	}
}

// GetUserIP extracts user IP address from HTTP request
func GetUserIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	if xForwardedFor != "" {
		// Take the first IP if multiple IPs are present
		ips := strings.Split(xForwardedFor, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	xRealIP := r.Header.Get("X-Real-IP")
	if xRealIP != "" {
		return xRealIP
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	// Remove port if present
	if colonIndex := strings.LastIndex(ip, ":"); colonIndex != -1 {
		ip = ip[:colonIndex]
	}
	return ip
}

// IsRateLimited checks if a user is making requests too frequently
func IsRateLimited(telegramID int64, action string, limit int, window time.Duration) bool {
	// For simplicity, we're checking against the security log
	// In a real implementation, you'd want a more efficient approach (like Redis)
	query := `
	SELECT COUNT(*) FROM security_logs 
	WHERE telegram_id = ? AND action = ? AND occurred_at > datetime('now', '-%d seconds')
	`

	var count int
	err := database.DB.Get(&count, fmt.Sprintf(query, int(window.Seconds())), telegramID, action)
	if err != nil {
		log.Printf("Error checking rate limit: %v", err)
		return false // Fail open
	}

	return count >= limit
}

// SanitizeInput removes potentially harmful characters from input
func SanitizeInput(input string) string {
	// Remove potential SQL injection characters (while preserving legitimate ones)
	sanitized := strings.ReplaceAll(input, "'", "''") // Escape single quotes
	sanitized = strings.ReplaceAll(sanitized, "\"", "") // Remove double quotes
	sanitized = strings.ReplaceAll(sanitized, ";", "")  // Remove semicolons
	sanitized = strings.ReplaceAll(sanitized, "--", "") // Remove comment indicators

	// Remove potential script tags
	reScript := regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
	sanitized = reScript.ReplaceAllString(sanitized, "")

	// Remove other potentially harmful HTML tags
	harmfulTags := []string{"<iframe", "<object", "<embed", "<form"}
	for _, tag := range harmfulTags {
		sanitized = strings.ReplaceAll(strings.ToLower(sanitized), strings.ToLower(tag), "")
	}

	return sanitized
}

// CheckUserAccess verifies if a user is allowed to perform an action
func CheckUserAccess(telegramID int64) (bool, error) {
	// Check if user is blocked
	isBlocked, err := database.IsUserBlocked(telegramID)
	if err != nil {
		return false, err
	}

	if isBlocked {
		return false, fmt.Errorf("user is temporarily blocked due to suspicious activity")
	}

	return true, nil
}