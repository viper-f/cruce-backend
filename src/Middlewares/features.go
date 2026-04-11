package Middlewares

import (
	"database/sql"

	"github.com/gin-gonic/gin"
)

func FeatureFlagsMiddleware(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := db.Query("SELECT key, is_active FROM features")
		if err != nil {
			c.Set("features", map[string]bool{})
			c.Next()
			return
		}
		defer rows.Close()

		features := make(map[string]bool)
		for rows.Next() {
			var key string
			var isActive bool
			if err := rows.Scan(&key, &isActive); err == nil {
				features[key] = isActive
			}
		}

		c.Set("features", features)
		c.Next()
	}
}
