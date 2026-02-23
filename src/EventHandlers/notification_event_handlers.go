package EventHandlers

import (
	"cuento-backend/src/Events"
	"cuento-backend/src/Websockets"
	"database/sql"
	"encoding/json"
	"fmt"
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

		_, err = db.Exec("INSERT INTO notifications (user_id, type, title, message, data, date_created, is_read) VALUES (?, ?, ?, ?, ?, NOW(), FALSE)",
			event.UserID, event.Type, title, event.Message, dataJSON)
		if err != nil {
			fmt.Printf("Error saving notification to DB: %v\n", err)
		}

		// Send live notification via WebSocket
		Websockets.MainHub.SendNotification(event.UserID, event)
	})
}
