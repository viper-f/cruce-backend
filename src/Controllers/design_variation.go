package Controllers

import (
	"cuento-backend/src/Entities"
	"cuento-backend/src/Middlewares"
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type DesignVariationUpdateRequest struct {
	ClassName *string `json:"class_name"`
	Name      *string `json:"name"`
}

func CreateDesignVariation(c *gin.Context, db *sql.DB) {
	var req Entities.DesignVariation
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid request body: " + err.Error()})
		c.Abort()
		return
	}

	res, err := db.Exec(
		"INSERT INTO design_variations (class_name, name) VALUES (?, ?)",
		req.ClassName, req.Name,
	)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to create design variation: " + err.Error()})
		c.Abort()
		return
	}

	id, _ := res.LastInsertId()
	req.Id = int(id)
	c.JSON(http.StatusOK, req)
}

func DeleteDesignVariation(c *gin.Context, db *sql.DB) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid id"})
		c.Abort()
		return
	}

	res, err := db.Exec("DELETE FROM design_variations WHERE id = ?", id)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to delete design variation: " + err.Error()})
		c.Abort()
		return
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusNotFound, Message: "Design variation not found"})
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Design variation deleted"})
}

func GetDesignVariationList(c *gin.Context, db *sql.DB) {
	rows, err := db.Query("SELECT id, class_name, name FROM design_variations")
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get design variations: " + err.Error()})
		c.Abort()
		return
	}
	defer rows.Close()

	var variations []Entities.DesignVariation
	for rows.Next() {
		var v Entities.DesignVariation
		if err := rows.Scan(&v.Id, &v.ClassName, &v.Name); err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to scan design variation: " + err.Error()})
			c.Abort()
			return
		}
		variations = append(variations, v)
	}

	if variations == nil {
		variations = []Entities.DesignVariation{}
	}

	c.JSON(http.StatusOK, variations)
}

func UpdateDesignVariation(c *gin.Context, db *sql.DB) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid id"})
		c.Abort()
		return
	}

	var req DesignVariationUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid request body: " + err.Error()})
		c.Abort()
		return
	}

	setClauses := []string{}
	args := []interface{}{}

	if req.ClassName != nil {
		setClauses = append(setClauses, "class_name = ?")
		args = append(args, *req.ClassName)
	}
	if req.Name != nil {
		setClauses = append(setClauses, "name = ?")
		args = append(args, *req.Name)
	}

	if len(setClauses) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "Design variation updated"})
		return
	}

	args = append(args, id)
	query := "UPDATE design_variations SET " + strings.Join(setClauses, ", ") + " WHERE id = ?"
	res, err := db.Exec(query, args...)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to update design variation: " + err.Error()})
		c.Abort()
		return
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusNotFound, Message: "Design variation not found"})
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Design variation updated"})
}
