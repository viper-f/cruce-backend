package EventHandlers

import (
	"cuento-backend/src/Entities"
	"cuento-backend/src/Events"
	"cuento-backend/src/Services"
	"database/sql"
	"fmt"
	"strconv"
)

func RegisterReactionEventHandlers() {
	Events.Subscribe(Events.ReactionCreated, func(db *sql.DB, data Events.EventData) {
		event, ok := data.(Events.ReactionCreatedEvent)
		if !ok {
			return
		}

		users := Services.ActivityStorage.GetUsersOnPage("topic", strconv.FormatInt(event.TopicID, 10))

		for _, u := range users {
			Events.Publish(db, Events.NotificationCreated, Events.NotificationEvent{
				UserID:  u.UserID,
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
		}
	})
}
