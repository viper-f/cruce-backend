package EventHandlers

import (
	"cuento-backend/src/Events"
	"database/sql"
	"fmt"
)

func RegisterWantedCharacterEventHandlers() {
	Events.Subscribe(Events.WantedCharacterCreated, func(db *sql.DB, data Events.EventData) {
		event, ok := data.(Events.WantedCharacterCreatedEvent)
		if !ok {
			return
		}

		_, err := db.Exec("UPDATE subforums SET topic_number = topic_number + 1, show_last_topic = true, last_post_topic_id = ?, last_post_topic_name = ? WHERE id = ?", event.TopicID, event.TopicName, event.SubforumID)
		if err != nil {
			fmt.Printf("Error updating subforum stats for wanted character: %v\n", err)
		}
	})
}
