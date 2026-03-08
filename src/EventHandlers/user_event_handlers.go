package EventHandlers

import (
	"cuento-backend/src/Events"
	"database/sql"
	"fmt"
)

func RegisterUserEventHandlers() {
	// Subscriber 15: Update Global Stats on User Registered
	Events.Subscribe(Events.UserRegistered, func(db *sql.DB, data Events.EventData) {
		event, ok := data.(Events.UserRegisteredEvent)
		if !ok {
			return
		}

		// 1. Update total user number
		_, err := db.Exec("UPDATE global_stats SET stat_value = stat_value + 1 WHERE stat_name = 'total_user_number'")
		if err != nil {
			fmt.Printf("Error updating global user stats: %v\n", err)
		}

		// 2. Update last user
		_, err = db.Exec("UPDATE global_stats SET stat_value = ?, stat_secondary = ? WHERE stat_name = 'last_user'", event.UserID, event.Username)
		if err != nil {
			fmt.Printf("Error updating last user global stat: %v\n", err)
		}
	})
}
