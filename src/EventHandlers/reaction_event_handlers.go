package EventHandlers

import (
	"cuento-backend/src/Events"
	"cuento-backend/src/Services"
	"cuento-backend/src/Websockets"
	"database/sql"
)

func RegisterReactionEventHandlers() {
	Events.Subscribe(Events.ReactionCreated, func(db *sql.DB, data Events.EventData) {
		event, ok := data.(Events.ReactionCreatedEvent)
		if !ok {
			return
		}

		users := Services.ActivityStorage.GetUsersOnPage("topic", event.TopicID)

		notification := map[string]interface{}{
			"type": "reaction_created",
			"data": map[string]interface{}{
				"topic_id":    event.TopicID,
				"post_id":     event.PostID,
				"reaction_id": event.ReactionID,
				"url":         event.Url,
				"user_id":     event.UserID,
				"user_name":   event.UserName,
			},
		}

		for _, u := range users {
			Websockets.MainHub.SendNotification(u.UserID, notification)
		}
	})
}
