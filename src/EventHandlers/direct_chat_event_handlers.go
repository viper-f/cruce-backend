package EventHandlers

import (
	"cuento-backend/src/Events"
	"cuento-backend/src/Services"
	"cuento-backend/src/Websockets"
	"database/sql"
	"fmt"
)

func RegisterDirectChatEventHandlers() {
	Events.Subscribe(Events.DirectMessageCreated, func(db *sql.DB, data Events.EventData) {
		event, ok := data.(Events.DirectMessageCreatedEvent)
		if !ok {
			return
		}

		// Get sender info
		var username string
		var avatar *string
		err := db.QueryRow("SELECT username, avatar FROM users WHERE id = ?", event.SenderID).Scan(&username, &avatar)
		if err != nil {
			fmt.Printf("Error fetching sender info for direct message event: %v\n", err)
			return
		}

		// Get all participants of this chat
		rows, err := db.Query(
			"SELECT user_id FROM direct_chat_users WHERE direct_chat_id = ?",
			event.ChatID,
		)
		if err != nil {
			fmt.Printf("Error fetching direct chat participants: %v\n", err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var participantID int
			if err := rows.Scan(&participantID); err != nil {
				continue
			}

			key := event.KeyReceiver
			if participantID == event.SenderID {
				key = event.KeyAuthor
			}

			tz := Services.GetUserTimezone(participantID, db)
			notification := map[string]interface{}{
				"type": "direct_message_created",
				"data": map[string]interface{}{
					"id":                  event.MessageID,
					"chat_id":             event.ChatID,
					"user_id":             event.SenderID,
					"username":            username,
					"avatar":              avatar,
					"date_send":           event.DateSend,
					"date_send_localized": Services.LocalizeTime(event.DateSend, tz),
					"date_received":       nil,
					"ciphertext":          event.Ciphertext,
					"iv":                  event.IV,
					"key":                 key,
				},
			}

			Websockets.MainHub.SendNotification(participantID, notification)
		}
	})
}
