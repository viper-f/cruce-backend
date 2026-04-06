package EventHandlers

import (
	"cuento-backend/src/Events"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
)

func RegisterStaticFileEventHandlers() {
	Events.Subscribe(Events.StaticFileUploaded, func(db *sql.DB, data Events.EventData) {
		event, ok := data.(Events.StaticFileUploadedEvent)
		if !ok {
			return
		}

		rows, err := db.Query(
			"SELECT file_name FROM static_files WHERE file_type = ? ORDER BY file_created_date DESC",
			event.FileType,
		)
		if err != nil {
			fmt.Printf("Error querying static files for cleanup: %v\n", err)
			return
		}
		defer rows.Close()

		var fileNames []string
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				continue
			}
			fileNames = append(fileNames, name)
		}

		if len(fileNames) <= 3 {
			return
		}

		toDelete := fileNames[3:]
		for _, name := range toDelete {
			path := filepath.Join("./public", name)
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				fmt.Printf("Error deleting static file %s: %v\n", name, err)
			}
			if _, err := db.Exec("DELETE FROM static_files WHERE file_name = ?", name); err != nil {
				fmt.Printf("Error deleting static file record %s: %v\n", name, err)
			}
		}
	})
}
