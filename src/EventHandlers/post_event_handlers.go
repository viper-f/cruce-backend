package EventHandlers

import (
	"cuento-backend/src/Entities"
	"cuento-backend/src/Events"
	"cuento-backend/src/Services"
	"cuento-backend/src/Websockets"
	"database/sql"
	"fmt"
	"strconv"
)

func RegisterPostEventHandlers() {
	// Subscriber 5: Notify New Post in Topic
	Events.Subscribe(Events.PostCreated, func(db *sql.DB, data Events.EventData) {
		event, ok := data.(Events.PostCreatedEvent)
		if !ok {
			return
		}

		// Handle both post_created and post_updated
		msgType := event.Type
		if msgType == "" {
			msgType = "post_created"
		}

		// Get all users currently reading this topic
		topicIDStr := strconv.FormatInt(event.TopicID, 10)
		users := Services.ActivityStorage.GetUsersOnPage("topic", topicIDStr)

		// Send to each user on the page with their localized date
		for _, u := range users {
			userPost := event.Post
			userPost.DateCreatedLocalized = Services.LocalizeTime(userPost.DateCreated, Services.GetUserTimezone(u.UserID, db))
			Websockets.MainHub.SendNotification(u.UserID, map[string]interface{}{
				"type": msgType,
				"data": userPost,
			})
		}
	})

	// Subscriber 6: Update Stats on Post Created
	Events.Subscribe(Events.PostCreated, func(db *sql.DB, data Events.EventData) {
		event, ok := data.(Events.PostCreatedEvent)
		if !ok || event.Type == "post_updated" {
			return
		}

		// 1. Update Global Stats
		_, err := db.Exec("UPDATE global_stats SET stat_value = stat_value + 1 WHERE stat_name = 'total_post_number'")
		if err != nil {
			fmt.Printf("Error updating global post stats: %v\n", err)
		}

		// Check if topic is an episode
		var topicType Entities.TopicType
		err = db.QueryRow("SELECT type FROM topics WHERE id = ?", event.TopicID).Scan(&topicType)
		if err == nil && topicType == Entities.EpisodeTopic {
			_, err = db.Exec("UPDATE global_stats SET stat_value = stat_value + 1 WHERE stat_name = 'total_episode_post_number'")
			if err != nil {
				fmt.Printf("Error updating global episode post stats: %v\n", err)
			}
		}

		// 2. Update Topic Stats
		_, err = db.Exec("UPDATE topics SET post_number = post_number + 1, date_last_post = NOW(), last_post_author_user_id = ? WHERE id = ?",
			event.Post.AuthorUserId, event.TopicID)
		if err != nil {
			fmt.Printf("Error updating topic stats: %v\n", err)
		}

		// 3. Update Subforum Stats
		var username string
		err = db.QueryRow("SELECT username FROM users WHERE id = ?", event.Post.AuthorUserId).Scan(&username)
		if err != nil {
			fmt.Printf("Error fetching username for stats: %v\n", err)
			return
		}

		var topicTitle string
		err = db.QueryRow("SELECT name FROM topics WHERE id = ?", event.TopicID).Scan(&topicTitle)
		if err != nil {
			fmt.Printf("Error fetching topic title for stats: %v\n", err)
			return
		}

		_, err = db.Exec("UPDATE subforums SET post_number = post_number + 1, last_post_topic_id = ?, last_post_topic_name = ?, last_post_id = ?, date_last_post = NOW(), last_post_author_user_name = ? WHERE id = ?",
			event.TopicID, topicTitle, event.Post.Id, username, event.SubforumID)
		if err != nil {
			fmt.Printf("Error updating subforum stats: %v\n", err)
		}
	})

	// Subscriber 11: Send Game Notifications for Episode Posts
	Events.Subscribe(Events.PostCreated, func(db *sql.DB, data Events.EventData) {
		event, ok := data.(Events.PostCreatedEvent)
		if !ok || event.Type == "post_updated" {
			return
		}

		// 1. Check if topic type is episode
		var topicType Entities.TopicType
		err := db.QueryRow("SELECT type FROM topics WHERE id = ?", event.TopicID).Scan(&topicType)
		if err != nil || topicType != Entities.EpisodeTopic {
			return
		}

		// 2. Get author character ID
		var authorCharacterID int
		if event.Post.CharacterProfile != nil && event.Post.CharacterProfile.CharacterId != nil {
			authorCharacterID = *event.Post.CharacterProfile.CharacterId
		} else {
			return
		}

		// 3. Get all participants of the episode
		query := `
			SELECT cb.user_id, cb.id as character_id
			FROM character_base cb
			JOIN episode_character ec ON cb.id = ec.character_id
			JOIN episode_base e ON ec.episode_id = e.id
			WHERE e.topic_id = ? AND cb.user_id != ?
		`
		rows, err := db.Query(query, event.TopicID, event.Post.AuthorUserId)
		if err != nil {
			fmt.Printf("Error fetching episode participants: %v\n", err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var participantUserID int
			var participantCharacterID int
			if err := rows.Scan(&participantUserID, &participantCharacterID); err == nil {
				gameData := Entities.NotificationGame{
					TopicId:         int(event.TopicID),
					Type:            "post_created",
					UserCharacterId: participantCharacterID,
					CharacterId:     authorCharacterID,
				}

				Events.Publish(db, Events.NotificationCreated, Events.NotificationEvent{
					UserID:  participantUserID,
					Type:    "game",
					Message: fmt.Sprintf("New post in episode by %s", event.Post.CharacterProfile.CharacterName),
					Data:    gameData,
				})
			}
		}
	})
}
