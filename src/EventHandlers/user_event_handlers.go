package EventHandlers

import (
	"cuento-backend/src/Events"
	"database/sql"
	"fmt"
)

func RegisterUserEventHandlers() {
	// Subscriber 15: Update Global Stats on User Registered
	Events.Subscribe(Events.UserRegistered, func(db *sql.DB, data Events.EventData) {
		_, ok := data.(Events.UserRegisteredEvent)
		if !ok {
			return
		}

		_, err := db.Exec("UPDATE global_stats SET stat_value = stat_value + 1 WHERE stat_name = 'total_user_number'")
		if err != nil {
			fmt.Printf("Error updating global user stats: %v\n", err)
		}
	})
}
