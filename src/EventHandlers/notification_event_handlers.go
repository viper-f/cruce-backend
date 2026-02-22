package EventHandlers

import (
	"cuento-backend/src/Events"
	"cuento-backend/src/Websockets"
	"database/sql"
)

func RegisterNotificationEventHandlers() {
	// Subscriber 3: Send Live Notifications
	Events.Subscribe(Events.NotificationCreated, func(db *sql.DB, data Events.EventData) {
		event, ok := data.(Events.NotificationEvent)
		if !ok {
			return
		}
		Websockets.MainHub.SendNotification(event.UserID, event)
	})
}
