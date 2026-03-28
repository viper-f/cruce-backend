package EventHandlers

import (
	"cuento-backend/src/Events"
	"database/sql"
	"fmt"
)

func RegisterEpisodeEventHandlers() {
	// Subscriber 9: Update Global Stats on Episode Created
	Events.Subscribe(Events.EpisodeCreated, func(db *sql.DB, data Events.EventData) {
		_, ok := data.(Events.EpisodeCreatedEvent)
		if !ok {
			return
		}

		_, err := db.Exec("UPDATE global_stats SET stat_value = stat_value + 1 WHERE stat_name = 'total_episode_number'")
		if err != nil {
			fmt.Printf("Error updating global episode stats: %v\n", err)
		}

		_, err = db.Exec("UPDATE global_stats SET stat_value = stat_value + 1 WHERE stat_name = 'total_topic_number'")
		if err != nil {
			fmt.Printf("Error updating global topic stats on episode created: %v\n", err)
		}
	})

	// Subscriber 10: Update Subforum Stats on Episode Created
	Events.Subscribe(Events.EpisodeCreated, func(db *sql.DB, data Events.EventData) {
		event, ok := data.(Events.EpisodeCreatedEvent)
		if !ok {
			return
		}

		_, err := db.Exec("UPDATE subforums SET topic_number = topic_number + 1, show_last_topic = true WHERE id = ?", event.SubforumID)
		if err != nil {
			fmt.Printf("Error updating subforum topic count for episode: %v\n", err)
		}
	})
}
