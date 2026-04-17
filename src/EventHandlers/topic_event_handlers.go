package EventHandlers

import (
	"cuento-backend/src/Events"
	"cuento-backend/src/Services"
	"cuento-backend/src/Websockets"
	"database/sql"
	"fmt"
)

func RegisterTopicEventHandlers() {
	// Subscriber 1: Update Global Stats
	Events.Subscribe(Events.TopicCreated, func(db *sql.DB, data Events.EventData) {
		_, ok := data.(Events.TopicCreatedEvent)
		if !ok {
			return
		}

		_, err := db.Exec("UPDATE global_stats SET stat_value = stat_value + 1 WHERE stat_name = 'total_post_number'")
		if err != nil {
			fmt.Printf("Error updating global post stats: %v\n", err)
		}
		_, err = db.Exec("UPDATE global_stats SET stat_value = stat_value + 1 WHERE stat_name = 'total_topic_number'")
		if err != nil {
			fmt.Printf("Error updating global topic stats: %v\n", err)
		}
	})

	// Subscriber 2: Update Subforum Stats
	Events.Subscribe(Events.TopicCreated, func(db *sql.DB, data Events.EventData) {
		event, ok := data.(Events.TopicCreatedEvent)
		if !ok {
			return
		}

		_, err := db.Exec("UPDATE subforums SET topic_number = topic_number + 1, post_number = post_number + 1, last_post_topic_id = ?, last_post_topic_name = ?, last_post_id = ?, date_last_post = NOW(), last_post_author_user_name = ? WHERE id = ?",
			event.TopicID, event.Title, event.PostID, event.Username, event.SubforumID)
		if err != nil {
			fmt.Printf("Error updating subforum stats: %v\n", err)
		}
	})

	// Subscriber 4: Notify Topic Viewers
	Events.Subscribe(Events.UserReadingTopic, func(db *sql.DB, data Events.EventData) {
		event, ok := data.(Events.UserReadingTopicEvent)
		if !ok {
			return
		}

		// Get all users currently reading this topic
		users := Services.ActivityStorage.GetUsersOnPage("topic", event.TopicID)

		// Construct the notification message — only visible users appear in the list
		type Viewer struct {
			UserID   int    `json:"user_id"`
			Username string `json:"username"`
		}
		var viewerList []Viewer
		for _, u := range users {
			if u.IsVisible {
				viewerList = append(viewerList, Viewer{
					UserID:   u.UserID,
					Username: u.Username,
				})
			}
		}

		notification := map[string]interface{}{
			"type": "topic_viewers_update",
			"data": viewerList,
		}

		// Send to each user on the page
		for _, u := range users {
			Websockets.MainHub.SendNotification(u.UserID, notification)
		}
	})
}
