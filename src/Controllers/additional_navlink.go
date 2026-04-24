package Controllers

import (
	"cuento-backend/src/Entities"
	"cuento-backend/src/Middlewares"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type NavlinkRequest struct {
	Title      *string                         `json:"title"`
	Type       *Entities.AdditionalNavlinkType `json:"type"`
	Config     json.RawMessage                 `json:"config"`
	Roles      []int                           `json:"roles"`
	IsInactive *bool                           `json:"is_inactive"`
}

func scanNavlink(row *sql.Row) (*Entities.AdditionalNavlink, error) {
	var n Entities.AdditionalNavlink
	var rolesRaw sql.NullString
	var config []byte
	if err := row.Scan(&n.Id, &n.Title, &n.Type, &config, &n.IsInactive, &rolesRaw); err != nil {
		return nil, err
	}
	if len(config) > 0 {
		n.Config = json.RawMessage(config)
	}
	if rolesRaw.Valid && rolesRaw.String != "" {
		n.Roles = strings.Split(rolesRaw.String, ",")
	} else {
		n.Roles = []string{}
	}
	return &n, nil
}

func scanNavlinkRows(rows *sql.Rows) ([]Entities.AdditionalNavlink, error) {
	var list []Entities.AdditionalNavlink
	for rows.Next() {
		var n Entities.AdditionalNavlink
		var rolesRaw sql.NullString
		var config []byte
		if err := rows.Scan(&n.Id, &n.Title, &n.Type, &config, &n.IsInactive, &rolesRaw); err != nil {
			return nil, err
		}
		if len(config) > 0 {
			n.Config = json.RawMessage(config)
		}
		if rolesRaw.Valid && rolesRaw.String != "" {
			n.Roles = strings.Split(rolesRaw.String, ",")
		} else {
			n.Roles = []string{}
		}
		list = append(list, n)
	}
	return list, nil
}

const navlinkSelectQuery = `
	SELECT n.id, n.title, n.type, n.config, n.is_inactive,
	       GROUP_CONCAT(r.name ORDER BY r.name SEPARATOR ',') as roles
	FROM additional_navlinks n
	LEFT JOIN role_navlink rn ON rn.navlink_id = n.id
	LEFT JOIN roles r ON r.id = rn.role_id`

func syncNavlinkRoles(tx *sql.Tx, navlinkID int, roles []int) error {
	if _, err := tx.Exec("DELETE FROM role_navlink WHERE navlink_id = ?", navlinkID); err != nil {
		return err
	}
	for _, roleID := range roles {
		if _, err := tx.Exec("INSERT INTO role_navlink (role_id, navlink_id) VALUES (?, ?)", roleID, navlinkID); err != nil {
			return err
		}
	}
	return nil
}

func GetRoleList(c *gin.Context, db *sql.DB) {
	rows, err := db.Query("SELECT id, name FROM roles")
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get roles: " + err.Error()})
		c.Abort()
		return
	}
	defer rows.Close()

	var roles []Entities.Role
	for rows.Next() {
		var r Entities.Role
		if err := rows.Scan(&r.Id, &r.Name); err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to scan role: " + err.Error()})
			c.Abort()
			return
		}
		roles = append(roles, r)
	}

	if roles == nil {
		roles = []Entities.Role{}
	}

	c.JSON(http.StatusOK, roles)
}

func CreateAdditionalNavlink(c *gin.Context, db *sql.DB) {
	var req NavlinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid request body: " + err.Error()})
		c.Abort()
		return
	}

	if req.Title == nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "title is required"})
		c.Abort()
		return
	}

	navType := Entities.LinkAdditionalNavlink
	if req.Type != nil {
		navType = *req.Type
	}

	isInactive := false
	if req.IsInactive != nil {
		isInactive = *req.IsInactive
	}

	tx, err := db.Begin()
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to start transaction: " + err.Error()})
		c.Abort()
		return
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		"INSERT INTO additional_navlinks (title, type, config, is_inactive) VALUES (?, ?, ?, ?)",
		*req.Title, navType, req.Config, isInactive,
	)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to create navlink: " + err.Error()})
		c.Abort()
		return
	}

	id, _ := res.LastInsertId()

	if len(req.Roles) > 0 {
		if err := syncNavlinkRoles(tx, int(id), req.Roles); err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to assign roles: " + err.Error()})
			c.Abort()
			return
		}
	}

	if err := tx.Commit(); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to commit transaction: " + err.Error()})
		c.Abort()
		return
	}

	row := db.QueryRow(navlinkSelectQuery+" WHERE n.id = ? GROUP BY n.id", id)
	navlink, err := scanNavlink(row)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to fetch navlink: " + err.Error()})
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, navlink)
}

func UpdateAdditionalNavlink(c *gin.Context, db *sql.DB) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid id"})
		c.Abort()
		return
	}

	var req NavlinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid request body: " + err.Error()})
		c.Abort()
		return
	}

	setClauses := []string{}
	args := []interface{}{}

	if req.Title != nil {
		setClauses = append(setClauses, "title = ?")
		args = append(args, *req.Title)
	}
	if req.Type != nil {
		setClauses = append(setClauses, "type = ?")
		args = append(args, *req.Type)
	}
	if req.Config != nil {
		setClauses = append(setClauses, "config = ?")
		args = append(args, []byte(req.Config))
	}
	if req.IsInactive != nil {
		setClauses = append(setClauses, "is_inactive = ?")
		args = append(args, *req.IsInactive)
	}

	tx, err := db.Begin()
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to start transaction: " + err.Error()})
		c.Abort()
		return
	}
	defer tx.Rollback()

	if len(setClauses) > 0 {
		args = append(args, id)
		query := "UPDATE additional_navlinks SET " + strings.Join(setClauses, ", ") + " WHERE id = ?"
		res, err := tx.Exec(query, args...)
		if err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to update navlink: " + err.Error()})
			c.Abort()
			return
		}
		rows, _ := res.RowsAffected()
		if rows == 0 {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusNotFound, Message: "Navlink not found"})
			c.Abort()
			return
		}
	}

	if req.Roles != nil {
		if err := syncNavlinkRoles(tx, id, req.Roles); err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to update roles: " + err.Error()})
			c.Abort()
			return
		}
	}

	if err := tx.Commit(); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to commit transaction: " + err.Error()})
		c.Abort()
		return
	}

	row := db.QueryRow(navlinkSelectQuery+" WHERE n.id = ? GROUP BY n.id", id)
	navlink, err := scanNavlink(row)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to fetch navlink: " + err.Error()})
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, navlink)
}

func DeleteAdditionalNavlink(c *gin.Context, db *sql.DB) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid id"})
		c.Abort()
		return
	}

	tx, err := db.Begin()
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to start transaction: " + err.Error()})
		c.Abort()
		return
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM role_navlink WHERE navlink_id = ?", id); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to delete navlink roles: " + err.Error()})
		c.Abort()
		return
	}

	res, err := tx.Exec("DELETE FROM additional_navlinks WHERE id = ?", id)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to delete navlink: " + err.Error()})
		c.Abort()
		return
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusNotFound, Message: "Navlink not found"})
		c.Abort()
		return
	}

	if err := tx.Commit(); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to commit transaction: " + err.Error()})
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Navlink deleted"})
}

func GetAdditionalNavlinkList(c *gin.Context, db *sql.DB) {
	rows, err := db.Query(navlinkSelectQuery + " GROUP BY n.id")
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get navlinks: " + err.Error()})
		c.Abort()
		return
	}
	defer rows.Close()

	list, err := scanNavlinkRows(rows)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to scan navlinks: " + err.Error()})
		c.Abort()
		return
	}

	if list == nil {
		list = []Entities.AdditionalNavlink{}
	}

	c.JSON(http.StatusOK, list)
}

func GetAdditionalNavlinkListByUser(c *gin.Context, db *sql.DB) {
	var rows *sql.Rows
	var err error

	userIDVal, exists := c.Get("user_id")
	if exists {
		userID, ok := userIDVal.(int)
		if !ok {
			userID = 0
		}
		rows, err = db.Query(
			navlinkSelectQuery+`
			WHERE n.id IN (
				SELECT rn2.navlink_id FROM role_navlink rn2
				JOIN user_role ur ON ur.role_id = rn2.role_id
				WHERE ur.user_id = ?
			)
			GROUP BY n.id`,
			userID,
		)
	} else {
		rows, err = db.Query(
			navlinkSelectQuery + `
			WHERE n.id IN (
				SELECT rn2.navlink_id FROM role_navlink rn2
				JOIN roles r2 ON r2.id = rn2.role_id
				WHERE r2.name = 'guest'
			)
			GROUP BY n.id`,
		)
	}
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get navlinks: " + err.Error()})
		c.Abort()
		return
	}
	defer rows.Close()

	list, err := scanNavlinkRows(rows)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to scan navlinks: " + err.Error()})
		c.Abort()
		return
	}

	if list == nil {
		list = []Entities.AdditionalNavlink{}
	}

	c.JSON(http.StatusOK, list)
}
