package Controllers

import (
	"cuento-backend/src/Entities"
	"cuento-backend/src/Middlewares"
	"cuento-backend/src/Services"
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type PageResponse struct {
	Id                   int                 `json:"id"`
	Content              string              `json:"content"`
	DateCreated          time.Time           `json:"date_created"`
	DateCreatedLocalized string              `json:"date_created_localized"`
	Author               *Entities.ShortUser `json:"author"`
}

func GetPage(c *gin.Context, db *sql.DB) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid page id"})
		c.Abort()
		return
	}

	var page PageResponse
	var authorId int
	var authorUsername string

	err = db.QueryRow(`
		SELECT p.id, p.content, p.date_created, u.id, u.username
		FROM pages p
		JOIN users u ON u.id = p.author_id
		WHERE p.id = ?`, id).
		Scan(&page.Id, &page.Content, &page.DateCreated, &authorId, &authorUsername)

	if err == sql.ErrNoRows {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusNotFound, Message: "Page not found"})
		c.Abort()
		return
	}
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get page: " + err.Error()})
		c.Abort()
		return
	}

	page.Author = &Entities.ShortUser{Id: authorId, Username: authorUsername}

	userID := Services.GetUserIdFromContext(c)
	page.DateCreatedLocalized = Services.LocalizeTime(page.DateCreated, Services.GetUserTimezone(userID, db))

	c.JSON(http.StatusOK, page)
}
