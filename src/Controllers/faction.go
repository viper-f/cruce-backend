package Controllers

import (
	"cuento-backend/src/Entities"
	"cuento-backend/src/Middlewares"
	"cuento-backend/src/Services"
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type FactionUpdateRequest struct {
	Name          *string                 `json:"name"`
	ParentId      *int                    `json:"parent_id"`
	Level         *int                    `json:"level"`
	Description   *string                 `json:"description"`
	Icon          *string                 `json:"icon"`
	ShowOnProfile *bool                   `json:"show_on_profile"`
	FactionStatus *Entities.FactionStatus `json:"faction_status"`
}

func GetFactionChildren(c *gin.Context, db *sql.DB) {
	parentIDStr := c.Param("parent_id")

	var rows *sql.Rows
	var err error

	if parentIDStr == "" || parentIDStr == "0" {
		rows, err = db.Query("SELECT id, name, parent_id, level, description, icon, show_on_profile, faction_status FROM factions WHERE parent_id IS NULL AND faction_status = ? ORDER BY name", Entities.FactionActive)
	} else {
		parentID, convErr := strconv.Atoi(parentIDStr)
		if convErr != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid parent_id"})
			c.Abort()
			return
		}
		rows, err = db.Query("SELECT id, name, parent_id, level, description, icon, show_on_profile, faction_status FROM factions WHERE parent_id = ? AND faction_status = ? ORDER BY name", parentID, Entities.FactionActive)
	}

	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get factions: " + err.Error()})
		c.Abort()
		return
	}
	defer rows.Close()

	var factions []Entities.Faction
	for rows.Next() {
		var f Entities.Faction
		if err := rows.Scan(&f.Id, &f.Name, &f.ParentId, &f.Level, &f.Description, &f.Icon, &f.ShowOnProfile, &f.FactionStatus); err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to scan faction: " + err.Error()})
			c.Abort()
			return
		}
		factions = append(factions, f)
	}

	if factions == nil {
		factions = []Entities.Faction{}
	}

	c.JSON(http.StatusOK, factions)
}

func GetFactionTree(c *gin.Context, db *sql.DB) {
	factions, err := Services.GetFactionTree(db)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get faction tree: " + err.Error()})
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, factions)
}

func GetActiveFactionTree(c *gin.Context, db *sql.DB) {
	factions, err := Services.GetActiveFactionTree(db)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get faction tree: " + err.Error()})
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, factions)
}

func GetWantedFactionTree(c *gin.Context, db *sql.DB) {
	factions, err := Services.GetWantedFactionTree(db)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get wanted faction tree: " + err.Error()})
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, factions)
}

func UpdateFactionById(c *gin.Context, db *sql.DB) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid id"})
		c.Abort()
		return
	}

	var req FactionUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid request body: " + err.Error()})
		c.Abort()
		return
	}

	setClauses := []string{}
	args := []interface{}{}

	if req.Name != nil {
		setClauses = append(setClauses, "name = ?")
		args = append(args, *req.Name)
	}
	if req.ParentId != nil {
		setClauses = append(setClauses, "parent_id = ?")
		args = append(args, *req.ParentId)
	}
	if req.Level != nil {
		setClauses = append(setClauses, "level = ?")
		args = append(args, *req.Level)
	}
	if req.Description != nil {
		setClauses = append(setClauses, "description = ?")
		args = append(args, *req.Description)
	}
	if req.Icon != nil {
		setClauses = append(setClauses, "icon = ?")
		args = append(args, *req.Icon)
	}
	if req.ShowOnProfile != nil {
		setClauses = append(setClauses, "show_on_profile = ?")
		args = append(args, *req.ShowOnProfile)
	}
	if req.FactionStatus != nil {
		setClauses = append(setClauses, "faction_status = ?")
		args = append(args, *req.FactionStatus)
	}

	if len(setClauses) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "Faction updated"})
		return
	}

	args = append(args, id)
	query := "UPDATE factions SET " + strings.Join(setClauses, ", ") + " WHERE id = ?"
	res, err := db.Exec(query, args...)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to update faction: " + err.Error()})
		c.Abort()
		return
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusNotFound, Message: "Faction not found"})
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Faction updated"})
}

func CreateFaction(c *gin.Context, db *sql.DB) {
	var req Entities.Faction
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid request body: " + err.Error()})
		c.Abort()
		return
	}

	id, err := Services.CreateFaction(req, db)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to create faction: " + err.Error()})
		c.Abort()
		return
	}

	req.Id = int(id)
	c.JSON(http.StatusOK, req)
}

func GetPendingFactions(c *gin.Context, db *sql.DB) {
	rows, err := db.Query("SELECT id, name, parent_id, level, description, icon, show_on_profile, faction_status FROM factions WHERE faction_status = ? ORDER BY name", Entities.FactionPending)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get pending factions: " + err.Error()})
		c.Abort()
		return
	}
	defer rows.Close()

	var factions []Entities.Faction
	for rows.Next() {
		var f Entities.Faction
		if err := rows.Scan(&f.Id, &f.Name, &f.ParentId, &f.Level, &f.Description, &f.Icon, &f.ShowOnProfile, &f.FactionStatus); err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to scan faction: " + err.Error()})
			c.Abort()
			return
		}
		factions = append(factions, f)
	}

	if factions == nil {
		factions = []Entities.Faction{}
	}

	c.JSON(http.StatusOK, factions)
}
