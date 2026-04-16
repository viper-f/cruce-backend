package Controllers

import (
	"cuento-backend/src/Entities"
	"cuento-backend/src/Middlewares"
	"cuento-backend/src/Services"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func GetReactionList(c *gin.Context, db *sql.DB) {
	rows, err := db.Query("SELECT id, url, is_active FROM reactions ORDER BY id ASC")
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get reactions: " + err.Error()})
		c.Abort()
		return
	}
	defer rows.Close()

	reactions := make([]Entities.Reaction, 0)
	for rows.Next() {
		var r Entities.Reaction
		if err := rows.Scan(&r.Id, &r.Url, &r.IsActive); err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to scan reaction: " + err.Error()})
			c.Abort()
			return
		}
		reactions = append(reactions, r)
	}
	c.JSON(http.StatusOK, reactions)
}

func CreateReaction(c *gin.Context, db *sql.DB) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "file field is required"})
		c.Abort()
		return
	}
	defer file.Close()

	ext := filepath.Ext(header.Filename)
	fileName := fmt.Sprintf("reaction_%d%s", time.Now().UnixNano(), ext)
	publicDir := "./../frontend/reactions"

	if err := os.MkdirAll(publicDir, os.ModePerm); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to create directory"})
		c.Abort()
		return
	}

	dst, err := os.Create(filepath.Join(publicDir, fileName))
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to create file: " + err.Error()})
		c.Abort()
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to write file: " + err.Error()})
		c.Abort()
		return
	}

	if err := changeToWwwData(filepath.Join(publicDir, fileName)); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Permission error: " + err.Error()})
		c.Abort()
		return
	}

	res, err := db.Exec("INSERT INTO reactions (url, is_active) VALUES (?, true)", fileName)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to save reaction: " + err.Error()})
		c.Abort()
		return
	}

	id, _ := res.LastInsertId()
	c.JSON(http.StatusCreated, gin.H{"id": id, "url": fileName})
}

func ActivateReaction(c *gin.Context, db *sql.DB) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid reaction ID"})
		c.Abort()
		return
	}

	result, err := db.Exec("UPDATE reactions SET is_active = true WHERE id = ?", id)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to activate reaction: " + err.Error()})
		c.Abort()
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusNotFound, Message: "Reaction not found"})
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": id, "is_active": true})
}

func DeactivateReaction(c *gin.Context, db *sql.DB) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid reaction ID"})
		c.Abort()
		return
	}

	result, err := db.Exec("UPDATE reactions SET is_active = false WHERE id = ?", id)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to deactivate reaction: " + err.Error()})
		c.Abort()
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusNotFound, Message: "Reaction not found"})
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": id, "is_active": false})
}

func GetActiveReactionList(c *gin.Context, db *sql.DB) {
	rows, err := db.Query("SELECT id, url, is_active FROM reactions WHERE is_active = true ORDER BY id ASC")
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get reactions: " + err.Error()})
		c.Abort()
		return
	}
	defer rows.Close()

	reactions := make([]Entities.Reaction, 0)
	for rows.Next() {
		var r Entities.Reaction
		if err := rows.Scan(&r.Id, &r.Url, &r.IsActive); err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to scan reaction: " + err.Error()})
			c.Abort()
			return
		}
		reactions = append(reactions, r)
	}
	c.JSON(http.StatusOK, reactions)
}

type ReactToPostRequest struct {
	PostId     int `json:"post_id" binding:"required"`
	ReactionId int `json:"reaction_id" binding:"required"`
}

func ReactToPost(c *gin.Context, db *sql.DB) {
	var req ReactToPostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid request body: " + err.Error()})
		c.Abort()
		return
	}

	userID := Services.GetUserIdFromContext(c)
	if userID == 0 {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusUnauthorized, Message: "Authentication required"})
		c.Abort()
		return
	}

	_, err := db.Exec(
		"INSERT INTO post_reaction (post_id, reaction_id, user_id) VALUES (?, ?, ?)",
		req.PostId, req.ReactionId, userID,
	)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to add reaction: " + err.Error()})
		c.Abort()
		return
	}

	c.JSON(http.StatusCreated, gin.H{"post_id": req.PostId, "reaction_id": req.ReactionId, "user_id": userID})
}
