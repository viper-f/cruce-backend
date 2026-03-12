package Services

import (
	"database/sql"
	"strconv"
)

func GetGlobalSetting(name string, db *sql.DB) (string, error) {
	var value string
	err := db.QueryRow("SELECT setting_value FROM global_settings WHERE setting_name = ?", name).Scan(&value)
	return value, err
}

func GetPostsPerPage(db *sql.DB) int {
	val, err := GetGlobalSetting("posts_per_page", db)
	if err != nil {
		return 20 // Default fallback
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return 20
	}
	return i
}
