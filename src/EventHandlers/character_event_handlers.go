package EventHandlers

import (
	"cuento-backend/src/Events"
	"database/sql"
	"fmt"
)

func RegisterCharacterEventHandlers() {
	// Subscriber 7: Update Global Stats on Character Created
	Events.Subscribe(Events.CharacterCreated, func(db *sql.DB, data Events.EventData) {
		_, ok := data.(Events.CharacterCreatedEvent)
		if !ok {
			return
		}

		_, err := db.Exec("UPDATE global_stats SET stat_value = stat_value + 1 WHERE stat_name = 'total_character_number'")
		if err != nil {
			fmt.Printf("Error updating global character stats: %v\n", err)
		}
	})

	// Subscriber 8: Update Subforum Stats on Character Created
	Events.Subscribe(Events.CharacterCreated, func(db *sql.DB, data Events.EventData) {
		event, ok := data.(Events.CharacterCreatedEvent)
		if !ok {
			return
		}

		_, err := db.Exec("UPDATE subforums SET topic_number = topic_number + 1 WHERE id = ?", event.SubforumID)
		if err != nil {
			fmt.Printf("Error updating subforum topic count for character: %v\n", err)
		}
	})
}
