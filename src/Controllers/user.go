package Controllers

import (
	"cuento-backend/src/Entities"
	"cuento-backend/src/Middlewares"
	"cuento-backend/src/Services"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type UserProfileResponse struct {
	UserId           int                        `json:"user_id"`
	Username         string                     `json:"username"`
	Avatar           *string                    `json:"avatar"`
	RegistrationDate time.Time                  `json:"registration_date"`
	Characters       []CharacterProfileListItem `json:"characters"`
}

type CharacterProfileListItem struct {
	Id            int                `json:"id"`
	Name          string             `json:"name"`
	TotalEpisodes int                `json:"total_episodes"`
	TotalPosts    int                `json:"total_posts"`
	LastPostDate  *time.Time         `json:"last_post_date"`
	Factions      []Entities.Faction `json:"factions"`
}

type UpdateSettingsRequest struct {
	Avatar   *string `json:"avatar"`
	Timezone *string `json:"interface_timezone"`
	Language *string `json:"interface_language"`
	Password *string `json:"password"`
}

type UserListItem struct {
	Id         int                       `json:"id"`
	Username   string                    `json:"username"`
	Characters []Entities.ShortCharacter `json:"characters"`
}

func Register(c *gin.Context, db *sql.DB) {
	var user Entities.User
	if err := c.ShouldBindJSON(&user); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid request body: " + err.Error()})
		c.Abort()
		return
	}

	// Password is already hashed by the frontend
	query := "INSERT INTO users (username, password, date_registered) VALUES (?, ?, ?)"
	res, err := db.Exec(query, user.Username, user.Password, time.Now())
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to create user"})
		c.Abort()
		return
	}

	id, err := res.LastInsertId()
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get user Id"})
		c.Abort()
		return
	}
	user.Id = int(id)

	// Get default role ID (assuming role with name "user" exists)
	var defaultRoleID int
	err = db.QueryRow("SELECT id FROM roles WHERE name = ?", "user").Scan(&defaultRoleID)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get default role"})
		c.Abort()
		return
	}

	// Assign default role to user
	_, err = db.Exec("INSERT INTO user_role (user_id, role_id) VALUES (?, ?)", user.Id, defaultRoleID)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to assign role to user"})
		c.Abort()
		return
	}

	user.Password = "" // Don't return password
	user.Roles = []Entities.Role{{Id: defaultRoleID, Name: "user"}}

	c.JSON(http.StatusCreated, user)
}

func Login(c *gin.Context, db *sql.DB) {
	var creds Credentials
	if err := c.ShouldBindJSON(&creds); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid request body: " + err.Error()})
		c.Abort()
		return
	}

	var user Entities.User
	query := "SELECT id, username, avatar, password, interface_language, interface_timezone, user_status FROM users WHERE username = ?"
	err := db.QueryRow(query, creds.Username).Scan(&user.Id, &user.Username, &user.Avatar, &user.Password, &user.InterfaceLanguage, &user.InterfaceTimezone, &user.UserStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusUnauthorized, Message: "Invalid credentials"})
		} else {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Database error"})
		}
		c.Abort()
		return
	}

	if user.UserStatus == Entities.BlockedUser {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusForbidden, Message: "User is blocked"})
		c.Abort()
		return
	}

	// Fetch user roles from many-to-many relationship
	rolesQuery := `
		SELECT r.id, r.name
		FROM roles r
		INNER JOIN user_role ur ON r.id = ur.role_id
		WHERE ur.user_id = ?`
	rows, err := db.Query(rolesQuery, user.Id)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to fetch user roles: " + err.Error()})
		c.Abort()
		return
	}
	defer rows.Close()

	user.Roles = []Entities.Role{}
	for rows.Next() {
		var role Entities.Role
		if err := rows.Scan(&role.Id, &role.Name); err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to scan role: " + err.Error()})
			c.Abort()
			return
		}
		user.Roles = append(user.Roles, role)
	}

	// Check for errors during iteration
	if err := rows.Err(); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Error iterating roles: " + err.Error()})
		c.Abort()
		return
	}

	// Password is already hashed by the frontend, so we compare directly
	if user.Password != creds.Password {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusUnauthorized, Message: "Invalid credentials"})
		c.Abort()
		return
	}

	// Generate Access Token
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Middlewares.Claims{
		Username: user.Username,
		UserID:   user.Id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			Issuer:    "cuento-backend",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(Middlewares.JwtKey)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to generate token"})
		c.Abort()
		return
	}

	// Generate Refresh Token
	refreshExpirationTime := time.Now().Add(7 * 24 * time.Hour)
	refreshClaims := &Middlewares.Claims{
		Username: user.Username,
		UserID:   user.Id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpirationTime),
			Issuer:    "cuento-backend",
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(Middlewares.JwtKey)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to generate refresh token"})
		c.Abort()
		return
	}

	user.Password = "" // Don't return password

	c.JSON(http.StatusOK, gin.H{
		"access_token":  tokenString,
		"refresh_token": refreshTokenString,
		"user":          user,
	})
}

func RefreshToken(c *gin.Context, db *sql.DB) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid request body: " + err.Error()})
		c.Abort()
		return
	}

	claims := &Middlewares.Claims{}
	token, err := jwt.ParseWithClaims(req.RefreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		return Middlewares.JwtKey, nil
	})

	if err != nil || !token.Valid {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusUnauthorized, Message: "Invalid refresh token"})
		c.Abort()
		return
	}

	// Fetch user details
	var user Entities.User
	query := "SELECT id, username, avatar, interface_language, interface_timezone, user_status FROM users WHERE id = ?"
	err = db.QueryRow(query, claims.UserID).Scan(&user.Id, &user.Username, &user.Avatar, &user.InterfaceLanguage, &user.InterfaceTimezone, &user.UserStatus)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to fetch user details"})
		c.Abort()
		return
	}

	if user.UserStatus == Entities.BlockedUser {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusForbidden, Message: "User is blocked"})
		c.Abort()
		return
	}

	// Fetch user roles
	rolesQuery := `
		SELECT r.id, r.name
		FROM roles r
		INNER JOIN user_role ur ON r.id = ur.role_id
		WHERE ur.user_id = ?`
	rows, err := db.Query(rolesQuery, user.Id)
	if err == nil {
		defer rows.Close()
		user.Roles = []Entities.Role{}
		for rows.Next() {
			var role Entities.Role
			if err := rows.Scan(&role.Id, &role.Name); err == nil {
				user.Roles = append(user.Roles, role)
			}
		}
	}

	// Generate new Access Token
	expirationTime := time.Now().Add(24 * time.Hour)
	newClaims := &Middlewares.Claims{
		Username: user.Username,
		UserID:   user.Id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			Issuer:    "cuento-backend",
		},
	}

	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	newTokenString, err := newToken.SignedString(Middlewares.JwtKey)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to generate new access token"})
		c.Abort()
		return
	}

	// Generate new Refresh Token
	refreshExpirationTime := time.Now().Add(7 * 24 * time.Hour)
	newRefreshClaims := &Middlewares.Claims{
		Username: user.Username,
		UserID:   user.Id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpirationTime),
			Issuer:    "cuento-backend",
		},
	}
	newRefreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, newRefreshClaims)
	newRefreshTokenString, err := newRefreshToken.SignedString(Middlewares.JwtKey)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to generate new refresh token"})
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  newTokenString,
		"refresh_token": newRefreshTokenString,
		"user":          user,
	})
}

func GetUsersByPage(c *gin.Context, db *sql.DB) {
	pageType := c.Param("page_type")
	pageId := c.Param("page_id")

	activeUsers := Services.ActivityStorage.GetUsersOnPage(pageType, pageId)

	var shortUsers []Entities.ShortUser
	for _, u := range activeUsers {
		shortUsers = append(shortUsers, Entities.ShortUser{
			Id:       u.UserID,
			Username: u.Username,
		})
	}

	// Return empty array instead of null
	if shortUsers == nil {
		shortUsers = []Entities.ShortUser{}
	}

	c.JSON(http.StatusOK, shortUsers)
}

func GetUserProfile(c *gin.Context, db *sql.DB) {
	userIDStr := c.Param("userID")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid user ID"})
		c.Abort()
		return
	}

	var profile UserProfileResponse
	err = db.QueryRow("SELECT id, username, avatar, date_registered FROM users WHERE id = ?", userID).Scan(&profile.UserId, &profile.Username, &profile.Avatar, &profile.RegistrationDate)
	if err != nil {
		if err == sql.ErrNoRows {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusNotFound, Message: "User not found"})
		} else {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get user details: " + err.Error()})
		}
		c.Abort()
		return
	}

	// Fetch characters for this user
	charRows, err := db.Query("SELECT id, name, total_episodes, total_posts, date_last_post FROM character_base WHERE user_id = ?", userID)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get user characters: " + err.Error()})
		c.Abort()
		return
	}
	defer charRows.Close()

	for charRows.Next() {
		var char CharacterProfileListItem
		if err := charRows.Scan(&char.Id, &char.Name, &char.TotalEpisodes, &char.TotalPosts, &char.LastPostDate); err != nil {
			continue
		}

		// Fetch factions of level 0 (root factions) for this character
		factionRows, err := db.Query("SELECT f.id, f.name FROM factions f JOIN character_faction cf ON f.id = cf.faction_id WHERE cf.character_id = ? AND f.level = 0", char.Id)
		if err == nil {
			var factions []Entities.Faction
			for factionRows.Next() {
				var faction Entities.Faction
				if err := factionRows.Scan(&faction.Id, &faction.Name); err == nil {
					factions = append(factions, faction)
				}
			}
			char.Factions = factions
			factionRows.Close()
		}

		profile.Characters = append(profile.Characters, char)
	}

	if profile.Characters == nil {
		profile.Characters = []CharacterProfileListItem{}
	}

	c.JSON(http.StatusOK, profile)
}

func UpdateSettings(c *gin.Context, db *sql.DB) {
	userID := Services.GetUserIdFromContext(c)
	if userID == 0 {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusUnauthorized, Message: "Unauthorized"})
		c.Abort()
		return
	}

	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid request body: " + err.Error()})
		c.Abort()
		return
	}

	var updates []string
	var args []interface{}

	if req.Avatar != nil {
		updates = append(updates, "avatar = ?")
		args = append(args, *req.Avatar)
	}
	if req.Timezone != nil {
		updates = append(updates, "interface_timezone = ?")
		args = append(args, *req.Timezone)
	}
	if req.Language != nil {
		updates = append(updates, "interface_language = ?")
		args = append(args, *req.Language)
	}
	if req.Password != nil {
		updates = append(updates, "password = ?")
		args = append(args, *req.Password)
	}

	if len(updates) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "No changes to update"})
		return
	}

	query := fmt.Sprintf("UPDATE users SET %s WHERE id = ?", strings.Join(updates, ", "))
	args = append(args, userID)

	_, err := db.Exec(query, args...)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to update settings: " + err.Error()})
		c.Abort()
		return
	}

	// Fetch updated user details
	var user Entities.User
	err = db.QueryRow("SELECT id, username, avatar, interface_language, interface_timezone, user_status FROM users WHERE id = ?", userID).Scan(&user.Id, &user.Username, &user.Avatar, &user.InterfaceLanguage, &user.InterfaceTimezone, &user.UserStatus)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to fetch updated user details"})
		c.Abort()
		return
	}

	// Fetch user roles
	rolesQuery := `
		SELECT r.id, r.name
		FROM roles r
		INNER JOIN user_role ur ON r.id = ur.role_id
		WHERE ur.user_id = ?`
	rows, err := db.Query(rolesQuery, user.Id)
	if err == nil {
		defer rows.Close()
		user.Roles = []Entities.Role{}
		for rows.Next() {
			var role Entities.Role
			if err := rows.Scan(&role.Id, &role.Name); err != nil {
				user.Roles = append(user.Roles, role)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Settings updated successfully",
		"user":    user,
	})
}

func GetUserList(c *gin.Context, db *sql.DB) {
	// 1. Fetch active users ordered alphabetically
	query := "SELECT id, username FROM users WHERE user_status = 0 AND id > 1 ORDER BY username ASC"
	rows, err := db.Query(query)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to fetch users: " + err.Error()})
		c.Abort()
		return
	}
	defer rows.Close()

	var users []UserListItem
	for rows.Next() {
		var user UserListItem
		if err := rows.Scan(&user.Id, &user.Username); err != nil {
			continue
		}

		// 2. Fetch active characters for each user
		charQuery := "SELECT id, name FROM character_base WHERE user_id = ? AND character_status = 0 ORDER BY name ASC"
		charRows, err := db.Query(charQuery, user.Id)
		if err == nil {
			user.Characters = []Entities.ShortCharacter{}
			for charRows.Next() {
				var char Entities.ShortCharacter
				if err := charRows.Scan(&char.Id, &char.Name); err == nil {
					user.Characters = append(user.Characters, char)
				}
			}
			charRows.Close()
		}

		users = append(users, user)
	}

	if users == nil {
		users = []UserListItem{}
	}

	c.JSON(http.StatusOK, users)
}
