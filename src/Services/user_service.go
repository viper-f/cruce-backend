package Services

import (
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// LocalizeTime converts t into the user's timezone and returns a formatted string.
// Accepts IANA names ("Europe/Moscow") or fixed-offset strings ("UTC+3", "UTC-5").
func LocalizeTime(t time.Time, timezone *string) string {
	loc := time.UTC
	if timezone != nil && *timezone != "" {
		tz := *timezone
		// Try IANA timezone name first (e.g. "Europe/Moscow", "America/New_York")
		if l, err := time.LoadLocation(tz); err == nil {
			loc = l
		} else {
			// Fall back to "UTC+05:00" / "UTC-05:00" / "UTC+5" fixed-offset parsing
			offsetStr := strings.TrimPrefix(tz, "UTC")
			sign := 1
			if strings.HasPrefix(offsetStr, "-") {
				sign = -1
				offsetStr = offsetStr[1:]
			} else if strings.HasPrefix(offsetStr, "+") {
				offsetStr = offsetStr[1:]
			}
			totalSeconds, parsed := 0, false
			if parts := strings.SplitN(offsetStr, ":", 2); len(parts) == 2 {
				hours, errH := strconv.Atoi(parts[0])
				minutes, errM := strconv.Atoi(parts[1])
				if errH == nil && errM == nil {
					totalSeconds = (hours*60 + minutes) * 60
					parsed = true
				}
			} else if hours, err := strconv.Atoi(offsetStr); err == nil {
				totalSeconds = hours * 3600
				parsed = true
			}
			if parsed {
				loc = time.FixedZone(tz, sign*totalSeconds)
			}
		}
	}
	return t.In(loc).Format("2006-01-02 15:04:05")
}

// GetUserTimezone fetches the interface_timezone for a given user from the DB.
// Returns nil if the user is a guest (userID == 0) or the value is not set.
func GetUserTimezone(userID int, db *sql.DB) *string {
	if userID == 0 {
		return nil
	}
	var tz sql.NullString
	if err := db.QueryRow("SELECT interface_timezone FROM users WHERE id = ?", userID).Scan(&tz); err != nil {
		return nil
	}
	if !tz.Valid {
		return nil
	}
	return &tz.String
}

// GetUserIdFromContext safely retrieves the user ID from the Gin context.
// It returns the user ID if it exists and is a valid integer.
// It returns 0 if the user ID does not exist or is not a valid type,
// indicating a guest user or an unauthenticated session.
func GetUserIdFromContext(c *gin.Context) int {
	if id, exists := c.Get("user_id"); exists {
		if userID, ok := id.(int); ok {
			return userID
		}
	}
	return 0
}
