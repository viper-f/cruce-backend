package EventHandlers

import (
	"cuento-backend/src/Entities"
	"cuento-backend/src/Events"
	"cuento-backend/src/Websockets"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

func RegisterNotificationEventHandlers() {
	// Subscriber 3: Send Live Notifications and Save to DB
	Events.Subscribe(Events.NotificationCreated, func(db *sql.DB, data Events.EventData) {
		event, ok := data.(Events.NotificationEvent)
		if !ok {
			return
		}

		// Save notification to database
		dataJSON, err := json.Marshal(event.Data)
		if err != nil {
			fmt.Printf("Error marshaling notification data: %v\n", err)
			return
		}

		title := "New Notification"
		if event.Type == "mention" {
			title = "You were mentioned"
		}

		res, err := db.Exec("INSERT INTO notifications (user_id, type, title, message, data, date_created, is_read) VALUES (?, ?, ?, ?, ?, NOW(), FALSE)",
			event.UserID, event.Type, title, event.Message, dataJSON)
		if err != nil {
			fmt.Printf("Error saving notification to DB: %v\n", err)
			return
		}

		notificationID, _ := res.LastInsertId()

		// Construct Notification entity for WebSocket
		notification := Entities.Notification{
			Id:          int(notificationID),
			UserId:      event.UserID,
			Type:        event.Type,
			Title:       title,
			Message:     event.Message,
			DateCreated: time.Now(),
			IsRead:      false,
		}

		// Unmarshal data into the entity based on type
		if len(dataJSON) > 0 {
			switch event.Type {
			case "mention":
				var mention Entities.NotificationMention
				if err := json.Unmarshal(dataJSON, &mention); err == nil {
					notification.Mention = &mention
				}
			case "game":
				var game Entities.NotificationGame
				if err := json.Unmarshal(dataJSON, &game); err == nil {
					notification.Game = &game
				}
			}
		}

		// Send live notification via WebSocket in the standard format
		wsMessage := map[string]interface{}{
			"type": "notification",
			"data": notification,
		}
		Websockets.MainHub.SendNotification(event.UserID, wsMessage)
	})
}
