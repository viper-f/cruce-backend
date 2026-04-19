package EventHandlers

import (
	"cuento-backend/src/Entities"
	"cuento-backend/src/Events"
	"database/sql"
	"fmt"
)

func RegisterReactionEventHandlers() {
	Events.Subscribe(Events.ReactionCreated, func(db *sql.DB, data Events.EventData) {
		event, ok := data.(Events.ReactionCreatedEvent)
		if !ok {
			return
		}

		Events.Publish(db, Events.NotificationCreated, Events.NotificationEvent{
			UserID:  event.PostAuthorID,
			Type:    "reaction",
			Message: fmt.Sprintf("%s reacted to a post in %s", event.UserName, event.TopicName),
			Data: Entities.NotificationReaction{
				PostId:     event.PostID,
				TopicId:    event.TopicID,
				TopicName:  event.TopicName,
				ReactionId: event.ReactionID,
				Url:        event.Url,
				UserId:     event.UserID,
				UserName:   event.UserName,
			},
		})
	})
}
