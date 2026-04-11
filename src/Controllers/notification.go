package Controllers

import (
	"cuento-backend/src/Entities"
	"cuento-backend/src/Middlewares"
	"cuento-backend/src/Services"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetUnreadNotifications(c *gin.Context, db *sql.DB) {
	userID := Services.GetUserIdFromContext(c)
	if userID == 0 {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusUnauthorized, Message: "Unauthorized"})
		c.Abort()
		return
	}

	rows, err := db.Query("SELECT id, type, title, message, data, date_created, is_read FROM notifications WHERE user_id = ? AND is_read = FALSE ORDER BY date_created DESC", userID)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to fetch notifications: " + err.Error()})
		c.Abort()
		return
	}
	defer rows.Close()

	result := map[string][]interface{}{
		"system":         {},
		"game":           {},
		"mention":        {},
		"account_update": {},
		"direct_message": {},
	}

	for rows.Next() {
		var base Entities.NotificationBase
		var dataJSON []byte
		if err := rows.Scan(&base.Id, &base.Type, &base.Title, &base.Message, &dataJSON, &base.DateCreated, &base.IsRead); err != nil {
			continue
		}
		base.UserId = userID

		var notification interface{}
		switch base.Type {
		case "mention":
			n := Entities.MentionNotification{NotificationBase: base}
			json.Unmarshal(dataJSON, &n.Data)
			notification = n
		case "game":
			n := Entities.GameNotification{NotificationBase: base}
			json.Unmarshal(dataJSON, &n.Data)
			notification = n
		case "system":
			n := Entities.SystemNotification{NotificationBase: base}
			json.Unmarshal(dataJSON, &n.Data)
			notification = n
		case "account_update":
			n := Entities.AccountUpdateNotification{NotificationBase: base}
			json.Unmarshal(dataJSON, &n.Data)
			notification = n
		case "direct_message":
			n := Entities.DirectMessageNotification{NotificationBase: base}
			json.Unmarshal(dataJSON, &n.Data)
			notification = n
		default:
			notification = base
		}

		if _, ok := result[base.Type]; ok {
			result[base.Type] = append(result[base.Type], notification)
		} else {
			result["system"] = append(result["system"], notification)
		}
	}

	c.JSON(http.StatusOK, result)
}

func DismissNotification(c *gin.Context, db *sql.DB) {
	userID := Services.GetUserIdFromContext(c)
	if userID == 0 {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusUnauthorized, Message: "Unauthorized"})
		c.Abort()
		return
	}

	notificationIDStr := c.Param("id")
	notificationID, err := strconv.Atoi(notificationIDStr)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid notification ID"})
		c.Abort()
		return
	}

	_, err = db.Exec("UPDATE notifications SET is_read = TRUE WHERE id = ? AND user_id = ?", notificationID, userID)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to dismiss notification: " + err.Error()})
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification dismissed"})
}
