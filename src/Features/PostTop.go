package Features

import (
	"cuento-backend/src/Entities"
	"cuento-backend/src/Services"
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	Services.RegisterFeatureWidget("post_top", "WidgetPostTop", WidgetPostTop)
}

type PostTopFeature struct {
}

func (PostTopFeature) Name() string {
	return "Post Top"
}

func (PostTopFeature) Description() string {
	return "Post top feature"
}

func (PostTopFeature) Key() string {
	return "post_top"
}

func (PostTopFeature) IsActive() bool {
	return false
}

func WidgetPostTop(_ map[string]interface{}, _ *sql.DB) (string, error) {
	return "Mock", nil
}

type PostTopCreateRequest struct {
	Name      string  `json:"name" binding:"required"`
	UserCount int     `json:"user_count" binding:"required"`
	Days      *int    `json:"days"`
	IsMonthly bool    `json:"is_monthly"`
	IsOpen    bool    `json:"is_open"`
	StartDate *string `json:"start_date"`
}

type PostTopUpdateRequest struct {
	Name      *string  `json:"name"`
	UserCount *int     `json:"user_count"`
	Days      *int     `json:"days"`
	IsMonthly *bool    `json:"is_monthly"`
	IsOpen    *bool    `json:"is_open"`
	StartDate **string `json:"start_date"`
}

func CreatePostTopHandler(c *gin.Context, db *sql.DB) {
	var req PostTopCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	res, err := db.Exec(
		"INSERT INTO post_tops (name, user_count, days, is_monthly, is_open, start_date) VALUES (?, ?, ?, ?, ?, ?)",
		req.Name, req.UserCount, req.Days, req.IsMonthly, req.IsOpen, req.StartDate,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create post top"})
		return
	}

	id, _ := res.LastInsertId()
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func UpdatePostTopHandler(c *gin.Context, db *sql.DB) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}

	var req PostTopUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	if req.Name == nil && req.UserCount == nil && req.Days == nil && req.IsMonthly == nil && req.IsOpen == nil && req.StartDate == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one field must be provided"})
		return
	}

	if req.Name != nil {
		db.Exec("UPDATE post_tops SET name = ? WHERE id = ?", *req.Name, id)
	}
	if req.UserCount != nil {
		db.Exec("UPDATE post_tops SET user_count = ? WHERE id = ?", *req.UserCount, id)
	}
	if req.Days != nil {
		db.Exec("UPDATE post_tops SET days = ? WHERE id = ?", *req.Days, id)
	}
	if req.IsMonthly != nil {
		db.Exec("UPDATE post_tops SET is_monthly = ? WHERE id = ?", *req.IsMonthly, id)
	}
	if req.IsOpen != nil {
		db.Exec("UPDATE post_tops SET is_open = ? WHERE id = ?", *req.IsOpen, id)
	}
	if req.StartDate != nil {
		db.Exec("UPDATE post_tops SET start_date = ? WHERE id = ?", *req.StartDate, id)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Post top updated"})
}

type PostTopEntry struct {
	UserID         int     `json:"user_id"`
	Username       string  `json:"username"`
	Avatar         *string `json:"avatar"`
	PostCount      int     `json:"post_count"`
	CharacterCount int     `json:"character_count"`
}

func GetPostTopHandler(c *gin.Context, db *sql.DB) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}

	var userCount int
	var days *int
	var isMonthly bool
	var dbStartDate *string
	err = db.QueryRow(
		"SELECT user_count, days, is_monthly, start_date FROM post_tops WHERE id = ?", id,
	).Scan(&userCount, &days, &isMonthly, &dbStartDate)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post top not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load post top"})
		return
	}

	var startDate time.Time
	startDateParam := c.Query("start_date")

	if startDateParam != "" {
		startDate, err = time.Parse("2006-01-02", startDateParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date, expected YYYY-MM-DD"})
			return
		}
	} else {
		now := time.Now().Truncate(24 * time.Hour)
		if isMonthly && dbStartDate != nil {
			dbStart, err := time.Parse("2006-01-02", *dbStartDate)
			if err == nil {
				startDate = time.Date(now.Year(), now.Month(), dbStart.Day(), 0, 0, 0, 0, time.UTC)
				if startDate.After(now) {
					startDate = startDate.AddDate(0, -1, 0)
				}
			}
		} else if days != nil && *days > 0 && dbStartDate != nil {
			dbStart, err := time.Parse("2006-01-02", *dbStartDate)
			if err == nil {
				daysSinceStart := int(now.Sub(dbStart).Hours() / 24)
				startDate = now.AddDate(0, 0, -(daysSinceStart % *days))
			}
		} else if dbStartDate != nil {
			startDate, _ = time.Parse("2006-01-02", *dbStartDate)
		} else {
			startDate = now
		}
	}

	var endDate time.Time
	if isMonthly {
		endDate = startDate.AddDate(0, 1, 0)
	} else if days != nil {
		endDate = startDate.AddDate(0, 0, *days)
	} else {
		endDate = time.Now()
	}

	rows, err := db.Query(`
		SELECT u.id, u.username, u.avatar, COUNT(p.id) AS post_count,
		       COUNT(DISTINCT p.character_profile_id) AS character_count
		FROM posts p
		JOIN topics t ON p.topic_id = t.id
		JOIN users u ON p.author_user_id = u.id
		WHERE t.type = ?
		  AND p.date_created >= ?
		  AND p.date_created < ?
		  AND (p.is_deleted IS NULL OR p.is_deleted = 0)
		GROUP BY u.id, u.username, u.avatar
		ORDER BY post_count DESC
		LIMIT ?
	`, Entities.EpisodeTopic, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"), userCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query post top"})
		return
	}
	defer rows.Close()

	entries := []PostTopEntry{}
	for rows.Next() {
		var e PostTopEntry
		if err := rows.Scan(&e.UserID, &e.Username, &e.Avatar, &e.PostCount, &e.CharacterCount); err != nil {
			continue
		}
		entries = append(entries, e)
	}

	c.JSON(http.StatusOK, gin.H{
		"items":      entries,
		"start_date": startDate.Format("2006-01-02"),
		"end_date":   endDate.Format("2006-01-02"),
	})
}
