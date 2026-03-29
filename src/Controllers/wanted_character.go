package Controllers

import (
	"cuento-backend/src/Entities"
	"cuento-backend/src/Events"
	"cuento-backend/src/Middlewares"
	"cuento-backend/src/Services"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type UpdateWantedCharacterRequest struct {
	Name             string                 `json:"name" binding:"required"`
	CharacterClaimId *int                   `json:"character_claim_id"`
	CustomFields     map[string]interface{} `json:"custom_fields"`
}

type CreateWantedCharacterRequest struct {
	SubforumID       int                    `json:"subforum_id" binding:"required"`
	Name             string                 `json:"name" binding:"required"`
	CharacterClaimId *int                   `json:"character_claim_id"`
	CustomFields     map[string]interface{} `json:"custom_fields"`
	Factions         []Entities.Faction     `json:"factions"`
}

func GetWantedCharacter(c *gin.Context, db *sql.DB) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid wanted character ID"})
		c.Abort()
		return
	}

	entity, err := Services.GetEntity(int64(id), "wanted_character", db)
	if err != nil {
		if err == sql.ErrNoRows {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusNotFound, Message: "Wanted character not found"})
		} else {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get wanted character: " + err.Error()})
		}
		c.Abort()
		return
	}

	if wc, ok := entity.(*Entities.WantedCharacter); ok {
		wc.Factions, _ = Services.GetFactionTreeByWantedCharacter(wc.Id, db)
	}

	c.JSON(http.StatusOK, entity)
}

func CreateWantedCharacter(c *gin.Context, db *sql.DB) {
	var req CreateWantedCharacterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid request body: " + err.Error()})
		c.Abort()
		return
	}

	userID := Services.GetUserIdFromContext(c)
	if userID == 0 {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusUnauthorized, Message: "Unauthorized"})
		c.Abort()
		return
	}

	tx, err := db.Begin()
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to start transaction"})
		c.Abort()
		return
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		"INSERT INTO topics (subforum_id, name, author_user_id, date_created, date_last_post, status, type, post_number, last_post_author_user_id) VALUES (?, ?, ?, NOW(), NOW(), 0, ?, 0, ?)",
		req.SubforumID, req.Name, userID, Entities.WantedCharacterTopic, userID,
	)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to insert topic: " + err.Error()})
		c.Abort()
		return
	}
	topicID, err := res.LastInsertId()
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get topic ID"})
		c.Abort()
		return
	}

	cfMap := make(map[string]Entities.CustomFieldValue)
	for k, v := range req.CustomFields {
		cfMap[k] = Entities.CustomFieldValue{Content: v}
	}

	wantedCharacter := Entities.WantedCharacter{
		Name:             req.Name,
		IsClaimed:        false,
		AuthorUserId:     userID,
		DateCreated:      time.Now(),
		CharacterClaimId: req.CharacterClaimId,
		TopicId:          int(topicID),
		CustomFields: Entities.CustomFieldEntity{
			CustomFields: cfMap,
		},
	}

	_, wantedCharacterID, err := Services.CreateEntity("wanted_character", &wantedCharacter, tx)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to create wanted character entity: " + err.Error()})
		c.Abort()
		return
	}

	claimRes, err := tx.Exec(
		"INSERT INTO character_claim (name, description, is_claimed, user_id, guest_hash, can_change_name, last_claim_date) VALUES (?, NULL, false, NULL, '', false, NULL)",
		req.Name,
	)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to create character claim: " + err.Error()})
		c.Abort()
		return
	}
	claimID, err := claimRes.LastInsertId()
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get character claim ID"})
		c.Abort()
		return
	}

	for _, faction := range req.Factions {
		if _, err := tx.Exec("INSERT INTO character_claim_faction (character_claim_id, faction_id) VALUES (?, ?)", claimID, faction.Id); err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: fmt.Sprintf("Failed to insert faction %d into character claim: %s", faction.Id, err.Error())})
			c.Abort()
			return
		}
	}

	if _, err := tx.Exec("UPDATE wanted_character_base SET character_claim_id = ? WHERE id = ?", claimID, wantedCharacterID); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to link character claim to wanted character: " + err.Error()})
		c.Abort()
		return
	}

	if err := tx.Commit(); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to commit transaction"})
		c.Abort()
		return
	}

	Events.Publish(db, Events.WantedCharacterCreated, Events.WantedCharacterCreatedEvent{
		WantedCharacterID: wantedCharacterID,
		SubforumID:        req.SubforumID,
		TopicID:           topicID,
		TopicName:         req.Name,
	})

	c.JSON(http.StatusCreated, gin.H{"wanted_character_id": wantedCharacterID, "topic_id": topicID})
}

func UpdateWantedCharacter(c *gin.Context, db *sql.DB) {
	wantedCharacterID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid wanted character ID"})
		c.Abort()
		return
	}

	var req UpdateWantedCharacterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid request body: " + err.Error()})
		c.Abort()
		return
	}

	userID := Services.GetUserIdFromContext(c)
	if userID == 0 {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusUnauthorized, Message: "Unauthorized"})
		c.Abort()
		return
	}

	var topicID int
	var authorUserID int
	var subforumID int
	err = db.QueryRow(`
		SELECT w.topic_id, t.author_user_id, t.subforum_id
		FROM wanted_character_base w
		JOIN topics t ON w.topic_id = t.id
		WHERE w.id = ?
	`, wantedCharacterID).Scan(&topicID, &authorUserID, &subforumID)
	if err != nil {
		if err == sql.ErrNoRows {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusNotFound, Message: "Wanted character not found"})
		} else {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to fetch wanted character details: " + err.Error()})
		}
		c.Abort()
		return
	}

	canEdit := false
	if userID == authorUserID {
		permission := fmt.Sprintf("subforum_edit_own_topic:%d", subforumID)
		if hasPerm, err := Services.HasPermission(userID, permission, db); err == nil && hasPerm {
			canEdit = true
		}
	} else {
		permission := fmt.Sprintf("subforum_edit_others_topic:%d", subforumID)
		if hasPerm, err := Services.HasPermission(userID, permission, db); err == nil && hasPerm {
			canEdit = true
		}
	}

	if !canEdit {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusForbidden, Message: "You do not have permission to edit this wanted character"})
		c.Abort()
		return
	}

	tx, err := db.Begin()
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to start transaction"})
		c.Abort()
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec("UPDATE topics SET name = ? WHERE id = ?", req.Name, topicID)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to update topic name: " + err.Error()})
		c.Abort()
		return
	}

	updates := map[string]interface{}{
		"name":               req.Name,
		"character_claim_id": req.CharacterClaimId,
		"custom_fields":      req.CustomFields,
	}
	_, err = Services.PatchEntity(int64(wantedCharacterID), "wanted_character", updates, tx)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to update wanted character entity: " + err.Error()})
		c.Abort()
		return
	}

	if err := tx.Commit(); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to commit transaction"})
		c.Abort()
		return
	}

	updatedEntity, err := Services.GetEntity(int64(wantedCharacterID), "wanted_character", db)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to fetch updated wanted character: " + err.Error()})
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, updatedEntity)
}
