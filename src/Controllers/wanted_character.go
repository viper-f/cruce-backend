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
	Name         string                 `json:"name" binding:"required"`
	CustomFields map[string]interface{} `json:"custom_fields"`
	Factions     []Entities.Faction     `json:"factions"`
}

type CreateWantedCharacterRequest struct {
	SubforumID       int                    `json:"subforum_id" binding:"required"`
	Name             string                 `json:"name" binding:"required"`
	CharacterClaimId *int                   `json:"character_claim_id"`
	CustomFields     map[string]interface{} `json:"custom_fields"`
	Factions         []Entities.Faction     `json:"factions"`
}

func GetWantedCharacterList(c *gin.Context, db *sql.DB) {
	rows, err := db.Query(`
		SELECT id, name, is_claimed, author_user_id, date_created, character_claim_id, is_deleted, topic_id
		FROM wanted_character_base
		WHERE is_claimed = false AND (is_deleted IS NULL OR is_deleted = false)
	`)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get wanted characters: " + err.Error()})
		c.Abort()
		return
	}
	defer rows.Close()

	var list []*Entities.WantedCharacter
	for rows.Next() {
		var wc Entities.WantedCharacter
		if err := rows.Scan(&wc.Id, &wc.Name, &wc.IsClaimed, &wc.AuthorUserId, &wc.DateCreated, &wc.CharacterClaimId, &wc.IsDeleted, &wc.TopicId); err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to scan wanted character: " + err.Error()})
			c.Abort()
			return
		}
		list = append(list, &wc)
	}

	for _, wc := range list {
		if entity, err := Services.GetEntity(int64(wc.Id), "wanted_character", db); err == nil {
			if full, ok := entity.(*Entities.WantedCharacter); ok {
				wc.CustomFields = full.CustomFields
			}
		}
		if wc.CharacterClaimId != nil {
			wc.Factions, _ = Services.GetFactionTreeByCharacterClaim(*wc.CharacterClaimId, db)
		} else {
			wc.Factions = []Entities.Faction{}
		}
	}

	if list == nil {
		list = []*Entities.WantedCharacter{}
	}

	c.JSON(http.StatusOK, list)
}

func GetWantedCharacterTreeList(c *gin.Context, db *sql.DB) {
	factions, err := Services.GetFactionTree(db)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get faction tree: " + err.Error()})
		c.Abort()
		return
	}

	factionMap := make(map[int]*Entities.Faction)
	for i := range factions {
		factions[i].Characters = []Entities.CharacterListItem{}
		factionMap[factions[i].Id] = &factions[i]
	}

	rows, err := db.Query(`
		WITH RankedFactions AS (
			SELECT
				cc.id,
				cc.name,
				f.id AS faction_id,
				ROW_NUMBER() OVER(PARTITION BY cc.id ORDER BY f.level DESC) AS rn
			FROM character_claim cc
			JOIN character_claim_faction ccf ON cc.id = ccf.character_claim_id
			JOIN factions f ON ccf.faction_id = f.id
			WHERE cc.is_claimed IS NOT TRUE
		)
		SELECT r.id, r.name, r.faction_id, wc.topic_id
		FROM RankedFactions r
		JOIN wanted_character_base wc ON wc.character_claim_id = r.id
		WHERE r.rn = 1
		  AND wc.is_claimed = false
		  AND (wc.is_deleted IS NULL OR wc.is_deleted = false)
	`)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get wanted characters: " + err.Error()})
		c.Abort()
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item Entities.CharacterListItem
		var factionID int
		if err := rows.Scan(&item.Id, &item.Name, &factionID, &item.WantedCharacterId); err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to scan wanted character: " + err.Error()})
			c.Abort()
			return
		}
		item.IsClaim = true
		if faction, ok := factionMap[factionID]; ok {
			faction.Characters = append(faction.Characters, item)
		}
	}

	keepIDs := make(map[int]bool)
	for _, f := range factions {
		if len(f.Characters) > 0 {
			keepIDs[f.Id] = true
			parentId := f.ParentId
			for parentId != nil {
				if keepIDs[*parentId] {
					break
				}
				keepIDs[*parentId] = true
				if parent, ok := factionMap[*parentId]; ok {
					parentId = parent.ParentId
				} else {
					break
				}
			}
		}
	}

	result := make([]Entities.Faction, 0)
	for _, f := range factions {
		if keepIDs[f.Id] {
			result = append(result, f)
		}
	}

	noFaction := Entities.Faction{Id: 0, Name: "No Faction", Characters: []Entities.CharacterListItem{}}

	noFactionRows, err := db.Query(`
		SELECT cc.id, cc.name, wc.topic_id
		FROM character_claim cc
		JOIN wanted_character_base wc ON wc.character_claim_id = cc.id
		WHERE cc.is_claimed IS NOT TRUE
		AND wc.is_claimed = false
		AND (wc.is_deleted IS NULL OR wc.is_deleted = false)
		AND cc.id NOT IN (SELECT character_claim_id FROM character_claim_faction)
	`)
	if err == nil {
		defer noFactionRows.Close()
		for noFactionRows.Next() {
			var item Entities.CharacterListItem
			if err := noFactionRows.Scan(&item.Id, &item.Name, &item.WantedCharacterId); err == nil {
				item.IsClaim = true
				noFaction.Characters = append(noFaction.Characters, item)
			}
		}
	}

	if len(noFaction.Characters) > 0 {
		result = append(result, noFaction)
	}

	c.JSON(http.StatusOK, result)
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
		if wc.CharacterClaimId != nil {
			wc.Factions, _ = Services.GetFactionTreeByCharacterClaim(*wc.CharacterClaimId, db)
		} else {
			wc.Factions = []Entities.Faction{}
		}
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
		"name":          req.Name,
		"custom_fields": req.CustomFields,
	}
	_, err = Services.PatchEntity(int64(wantedCharacterID), "wanted_character", updates, tx)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to update wanted character entity: " + err.Error()})
		c.Abort()
		return
	}

	if req.Factions != nil {
		var claimID int
		err = tx.QueryRow("SELECT character_claim_id FROM wanted_character_base WHERE id = ?", wantedCharacterID).Scan(&claimID)
		if err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get character claim ID: " + err.Error()})
			c.Abort()
			return
		}

		if _, err = tx.Exec("DELETE FROM character_claim_faction WHERE character_claim_id = ?", claimID); err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to clear factions: " + err.Error()})
			c.Abort()
			return
		}

		for _, faction := range req.Factions {
			if _, err = tx.Exec("INSERT INTO character_claim_faction (character_claim_id, faction_id) VALUES (?, ?)", claimID, faction.Id); err != nil {
				_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: fmt.Sprintf("Failed to insert faction %d: %s", faction.Id, err.Error())})
				c.Abort()
				return
			}
		}
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
