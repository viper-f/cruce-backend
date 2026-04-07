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
	"strings"
	"time"

	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"

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
	UserId                    int                        `json:"user_id"`
	Username                  string                     `json:"username"`
	Avatar                    *string                    `json:"avatar"`
	RegistrationDate          time.Time                  `json:"registration_date"`
	RegistrationDateLocalized string                     `json:"registration_date_localized"`
	TotalPosts                int                        `json:"total_posts"`
	TotalGeneralPosts         int                        `json:"total_general_posts"`
	Characters                []CharacterProfileListItem `json:"characters"`
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
	Avatar          *string        `json:"avatar"`
	Timezone        *string        `json:"interface_timezone"`
	Language        *string        `json:"interface_language"`
	FontSize        *float64       `json:"interface_font_size"`
	Password        *string        `json:"password"`
	DisableSound    *bool          `json:"disable_sound"`
	InterfaceDesign NullableString `json:"interface_design"`
}

type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
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

	if err := user.HashPassword(user.Password); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to hash password: " + err.Error()})
		c.Abort()
		return
	}

	defaultLang := "en-US"
	defaultTZ := "Europe/London"

	query := "INSERT INTO users (username, password, date_registered, interface_language, interface_timezone) VALUES (?, ?, ?, ?, ?)"
	res, err := db.Exec(query, user.Username, user.Password, time.Now(), defaultLang, defaultTZ)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to create user: " + err.Error()})
		c.Abort()
		return
	}

	id, err := res.LastInsertId()
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get user Id: " + err.Error()})
		c.Abort()
		return
	}
	user.Id = int(id)
	user.InterfaceLanguage = &defaultLang
	user.InterfaceTimezone = &defaultTZ

	// Get default role ID (assuming role with name "user" exists)
	var defaultRoleID int
	err = db.QueryRow("SELECT id FROM roles WHERE name = ?", "user").Scan(&defaultRoleID)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get default role: " + err.Error()})
		c.Abort()
		return
	}

	// Assign default role to user
	_, err = db.Exec("INSERT INTO user_role (user_id, role_id) VALUES (?, ?)", user.Id, defaultRoleID)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to assign role to user: " + err.Error()})
		c.Abort()
		return
	}

	user.Password = "" // Don't return password
	user.Roles = []Entities.Role{{Id: defaultRoleID, Name: "user"}}

	// Emit UserRegistered event
	Events.Publish(db, Events.UserRegistered, Events.UserRegisteredEvent{
		UserID:   user.Id,
		Username: user.Username,
	})

	c.JSON(http.StatusCreated, gin.H{"user": user})
}

func CreateUser(c *gin.Context, db *sql.DB) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid request body: " + err.Error()})
		c.Abort()
		return
	}

	var user Entities.User
	user.Username = req.Username
	user.Password = req.Password
	if err := user.HashPassword(user.Password); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to hash password: " + err.Error()})
		c.Abort()
		return
	}

	defaultLang := "en-US"
	defaultTZ := "Europe/London"

	res, err := db.Exec("INSERT INTO users (username, password, date_registered, interface_language, interface_timezone) VALUES (?, ?, ?, ?, ?)",
		user.Username, user.Password, time.Now(), defaultLang, defaultTZ)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to create user: " + err.Error()})
		c.Abort()
		return
	}

	id, err := res.LastInsertId()
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get user ID: " + err.Error()})
		c.Abort()
		return
	}

	var defaultRoleID int
	err = db.QueryRow("SELECT id FROM roles WHERE name = ?", "user").Scan(&defaultRoleID)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get default role: " + err.Error()})
		c.Abort()
		return
	}

	_, err = db.Exec("INSERT INTO user_role (user_id, role_id) VALUES (?, ?)", id, defaultRoleID)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to assign role to user: " + err.Error()})
		c.Abort()
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":       id,
		"username": user.Username,
	})
}

func Login(c *gin.Context, db *sql.DB) {
	var creds Credentials
	if err := c.ShouldBindJSON(&creds); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid request body: " + err.Error()})
		c.Abort()
		return
	}

	var user Entities.User
	query := "SELECT id, username, avatar, password, interface_language, interface_timezone, interface_font_size, user_status, interface_design FROM users WHERE username = ?"
	err := db.QueryRow(query, creds.Username).Scan(&user.Id, &user.Username, &user.Avatar, &user.Password, &user.InterfaceLanguage, &user.InterfaceTimezone, &user.InterfaceFontSize, &user.UserStatus, &user.InterfaceDesign)
	if err != nil {
		if err == sql.ErrNoRows {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusUnauthorized, Message: "Invalid credentials"})
		} else {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Database error: " + err.Error()})
		}
		c.Abort()
		return
	}

	if user.UserStatus == Entities.BlockedUser {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusForbidden, Message: "User is blocked"})
		c.Abort()
		return
	}

	if err := user.CheckPassword(creds.Password); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusUnauthorized, Message: "Invalid credentials"})
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
	query := "SELECT id, username, avatar, interface_language, interface_timezone, interface_font_size, user_status, total_posts, total_general_posts, disable_sound, interface_design FROM users WHERE id = ?"
	err = db.QueryRow(query, claims.UserID).Scan(&user.Id, &user.Username, &user.Avatar, &user.InterfaceLanguage, &user.InterfaceTimezone, &user.InterfaceFontSize, &user.UserStatus, &user.TotalPosts, &user.TotalGeneralPosts, &user.DisableSound, &user.InterfaceDesign)
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
	err = db.QueryRow("SELECT id, username, avatar, date_registered, total_posts, total_general_posts FROM users WHERE id = ?", userID).Scan(&profile.UserId, &profile.Username, &profile.Avatar, &profile.RegistrationDate, &profile.TotalPosts, &profile.TotalGeneralPosts)
	if err != nil {
		if err == sql.ErrNoRows {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusNotFound, Message: "User not found"})
		} else {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get user details: " + err.Error()})
		}
		c.Abort()
		return
	}

	currentUserID := Services.GetUserIdFromContext(c)
	viewerTimezone := Services.GetUserTimezone(currentUserID, db)
	profile.RegistrationDateLocalized = Services.LocalizeTime(profile.RegistrationDate, viewerTimezone)

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
	if req.FontSize != nil {
		updates = append(updates, "interface_font_size = ?")
		args = append(args, *req.FontSize)
	}
	if req.DisableSound != nil {
		updates = append(updates, "disable_sound = ?")
		args = append(args, *req.DisableSound)
	}
	if req.InterfaceDesign.IsSet {
		updates = append(updates, "interface_design = ?")
		args = append(args, req.InterfaceDesign.Value)
	}
	if req.Password != nil {
		// Hash the password before updating
		dummyUser := Entities.User{}
		if err := dummyUser.HashPassword(*req.Password); err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to hash password"})
			c.Abort()
			return
		}
		updates = append(updates, "password = ?")
		args = append(args, dummyUser.Password)
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
	err = db.QueryRow("SELECT id, username, avatar, interface_language, interface_timezone, interface_font_size, user_status, total_posts, total_general_posts, disable_sound, interface_design FROM users WHERE id = ?", userID).Scan(&user.Id, &user.Username, &user.Avatar, &user.InterfaceLanguage, &user.InterfaceTimezone, &user.InterfaceFontSize, &user.UserStatus, &user.TotalPosts, &user.TotalGeneralPosts, &user.DisableSound, &user.InterfaceDesign)
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
			if err := rows.Scan(&role.Id, &role.Name); err == nil {
				user.Roles = append(user.Roles, role)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Settings updated successfully",
		"user":    user,
	})
}

type AdminUserListItem struct {
	Id               int        `json:"id"`
	Username         string     `json:"username"`
	UserStatus       int        `json:"user_status"`
	DateRegistered   *time.Time `json:"date_registered"`
	DateLastVisit    *time.Time `json:"date_last_visit"`
	CharacterCount   int        `json:"character_count"`
	LastGamePostDate *time.Time `json:"last_game_post_date"`
}

func GetAdminUserList(c *gin.Context, db *sql.DB) {
	rows, err := db.Query(`
		SELECT
			u.id,
			u.username,
			u.user_status,
			u.date_registered,
			u.date_last_visit,
			COUNT(c.id) AS character_count,
			MAX(c.date_last_post) AS last_game_post_date
		FROM users u
		LEFT JOIN character_base c ON c.user_id = u.id
		WHERE u.id > 1
		GROUP BY u.id, u.username, u.user_status, u.date_registered, u.date_last_visit
		ORDER BY u.username ASC
	`)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get users: " + err.Error()})
		c.Abort()
		return
	}
	defer rows.Close()

	users := []AdminUserListItem{}
	for rows.Next() {
		var u AdminUserListItem
		if err := rows.Scan(&u.Id, &u.Username, &u.UserStatus, &u.DateRegistered, &u.DateLastVisit, &u.CharacterCount, &u.LastGamePostDate); err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to scan user: " + err.Error()})
			c.Abort()
			return
		}
		users = append(users, u)
	}

	c.JSON(http.StatusOK, users)
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

type SaveKeysPrivateKeyItem struct {
	PrivateKey string `json:"private_key" binding:"required"`
	IV         string `json:"iv" binding:"required"`
	Salt       string `json:"salt" binding:"required"`
}

type SaveKeysRequest struct {
	Codes               []string                 `json:"codes" binding:"required"`
	PrivateKey          SaveKeysPrivateKeyItem   `json:"private_key" binding:"required"`
	RecoveryPrivateKeys []SaveKeysPrivateKeyItem `json:"recovery_private_keys" binding:"required"`
	PublicKey           Entities.PublicKey       `json:"public_key" binding:"required"`
}

func SaveKeys(c *gin.Context, db *sql.DB) {
	userID := Services.GetUserIdFromContext(c)
	if userID == 0 {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusUnauthorized, Message: "Unauthorized"})
		c.Abort()
		return
	}

	var req SaveKeysRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid request body: " + err.Error()})
		c.Abort()
		return
	}

	if len(req.Codes) != len(req.RecoveryPrivateKeys) {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "codes and recovery_private_keys must have the same length"})
		c.Abort()
		return
	}

	// Insert the active (password-encrypted) private key
	_, err := db.Exec(
		"INSERT INTO private_keys (user_id, private_key, salt, iv, recovery_code_id, is_active) VALUES (?, ?, ?, ?, NULL, true)",
		userID, req.PrivateKey.PrivateKey, req.PrivateKey.Salt, req.PrivateKey.IV,
	)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to save private key: " + err.Error()})
		c.Abort()
		return
	}

	// Insert recovery codes and their linked private keys
	for i, code := range req.Codes {
		hashBytes := sha256.Sum256([]byte(code))
		hashHex := fmt.Sprintf("%x", hashBytes)

		res, err := db.Exec(
			"INSERT INTO recovery_codes (user_id, recovery_code) VALUES (?, ?)",
			userID, hashHex,
		)
		if err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to save recovery code: " + err.Error()})
			c.Abort()
			return
		}

		codeID, err := res.LastInsertId()
		if err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get recovery code ID: " + err.Error()})
			c.Abort()
			return
		}

		pk := req.RecoveryPrivateKeys[i]
		_, err = db.Exec(
			"INSERT INTO private_keys (user_id, private_key, salt, iv, recovery_code_id, is_active) VALUES (?, ?, ?, ?, ?, false)",
			userID, pk.PrivateKey, pk.Salt, pk.IV, codeID,
		)
		if err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to save recovery private key: " + err.Error()})
			c.Abort()
			return
		}
	}

	_, err = db.Exec(
		"INSERT INTO public_keys (user_id, public_key) VALUES (?, ?)",
		userID, req.PublicKey.PublicKey,
	)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to save public key: " + err.Error()})
		c.Abort()
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Keys saved successfully"})
}

func GetPrivateKey(c *gin.Context, db *sql.DB) {
	userID := Services.GetUserIdFromContext(c)
	if userID == 0 {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusUnauthorized, Message: "Unauthorized"})
		c.Abort()
		return
	}

	var privateKey, salt, iv string
	err := db.QueryRow(
		"SELECT private_key, salt, iv FROM private_keys WHERE user_id = ? AND is_active = true",
		userID,
	).Scan(&privateKey, &salt, &iv)
	if err != nil {
		if err == sql.ErrNoRows {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusNotFound, Message: "No active private key found"})
		} else {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get private key: " + err.Error()})
		}
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, gin.H{"private_key": privateKey, "salt": salt, "iv": iv})
}

func GetPublicKeyByUserId(c *gin.Context, db *sql.DB) {
	userID, err := strconv.Atoi(c.Param("userID"))
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid user ID"})
		c.Abort()
		return
	}

	var pk Entities.PublicKey
	err = db.QueryRow(
		"SELECT user_id, public_key FROM public_keys WHERE user_id = ?",
		userID,
	).Scan(&pk.UserId, &pk.PublicKey)
	if err != nil {
		if err == sql.ErrNoRows {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusNotFound, Message: "No public key found for user"})
		} else {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get public key: " + err.Error()})
		}
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, pk)
}

type RecoveryKeyItem struct {
	PrivateKey string `json:"private_key" binding:"required"`
	IV         string `json:"iv" binding:"required"`
	Salt       string `json:"salt" binding:"required"`
}

type SaveRecoveryKeysRequest struct {
	Codes       []string          `json:"codes" binding:"required"`
	PrivateKeys []RecoveryKeyItem `json:"private_keys" binding:"required"`
}

func SaveRecoveryKeys(c *gin.Context, db *sql.DB) {
	userID := Services.GetUserIdFromContext(c)
	if userID == 0 {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusUnauthorized, Message: "Unauthorized"})
		c.Abort()
		return
	}

	var req SaveRecoveryKeysRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid request body: " + err.Error()})
		c.Abort()
		return
	}

	if len(req.Codes) != len(req.PrivateKeys) {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "codes and private_keys must have the same length"})
		c.Abort()
		return
	}

	for i, code := range req.Codes {
		res, err := db.Exec(
			"INSERT INTO recovery_codes (user_id, recovery_code) VALUES (?, ?)",
			userID, code,
		)
		if err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to save recovery code: " + err.Error()})
			c.Abort()
			return
		}

		codeID, err := res.LastInsertId()
		if err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get recovery code ID: " + err.Error()})
			c.Abort()
			return
		}

		pk := req.PrivateKeys[i]
		_, err = db.Exec(
			"INSERT INTO private_keys (user_id, private_key, salt, iv, recovery_code_id, is_active) VALUES (?, ?, ?, ?, ?, false)",
			userID, pk.PrivateKey, pk.Salt, pk.IV, codeID,
		)
		if err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to save recovery key: " + err.Error()})
			c.Abort()
			return
		}
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Recovery keys saved successfully"})
}

type RecoveryRequest struct {
	Code string `json:"code" binding:"required"`
}

type UpdatePasswordRequest struct {
	Password     string `json:"password" binding:"required"`
	PrivateKey   string `json:"private_key" binding:"required"`
	IV           string `json:"iv" binding:"required"`
	Salt         string `json:"salt" binding:"required"`
	SecurityCode string `json:"security_code" binding:"required"`
}

func UpdatePassword(c *gin.Context, db *sql.DB) {
	var req UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid request body: " + err.Error()})
		c.Abort()
		return
	}

	var userID int
	err := db.QueryRow(
		"SELECT user_id FROM recovery_codes WHERE security_code = ?",
		req.SecurityCode,
	).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusNotFound, Message: "Invalid security code"})
		} else {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to validate security code: " + err.Error()})
		}
		c.Abort()
		return
	}

	dummyUser := Entities.User{}
	if err := dummyUser.HashPassword(req.Password); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to hash password: " + err.Error()})
		c.Abort()
		return
	}

	_, err = db.Exec("UPDATE users SET password = ? WHERE id = ?", dummyUser.Password, userID)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to update password: " + err.Error()})
		c.Abort()
		return
	}

	_, err = db.Exec(
		"UPDATE private_keys SET private_key = ?, iv = ?, salt = ? WHERE user_id = ? AND is_active = true",
		req.PrivateKey, req.IV, req.Salt, userID,
	)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to update private key: " + err.Error()})
		c.Abort()
		return
	}

	_, _ = db.Exec("UPDATE recovery_codes SET security_code = NULL, date_used = ? WHERE user_id = ?", time.Now(), userID)

	c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully"})
}

func Recovery(c *gin.Context, db *sql.DB) {
	var req RecoveryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid request body: " + err.Error()})
		c.Abort()
		return
	}

	hashBytes := sha256.Sum256([]byte(req.Code))
	hashHex := fmt.Sprintf("%x", hashBytes)

	var recoveryCodeID int
	err := db.QueryRow(
		"SELECT id FROM recovery_codes WHERE recovery_code = ? AND date_used IS NULL",
		hashHex,
	).Scan(&recoveryCodeID)
	if err != nil {
		if err == sql.ErrNoRows {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusNotFound, Message: "Recovery code not found"})
		} else {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to look up recovery code: " + err.Error()})
		}
		c.Abort()
		return
	}

	var pk Entities.PrivateKey
	err = db.QueryRow(
		"SELECT user_id, private_key, salt, iv, recovery_code_id FROM private_keys WHERE recovery_code_id = ?",
		recoveryCodeID,
	).Scan(&pk.UserId, &pk.PrivateKey, &pk.Salt, &pk.IV, &pk.RecoveryKeyId)
	if err != nil {
		if err == sql.ErrNoRows {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusNotFound, Message: "No private key found for this recovery code"})
		} else {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get private key: " + err.Error()})
		}
		c.Abort()
		return
	}

	securityCodeBytes := make([]byte, 32)
	if _, err := rand.Read(securityCodeBytes); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to generate security code: " + err.Error()})
		c.Abort()
		return
	}
	securityCode := hex.EncodeToString(securityCodeBytes)

	_, err = db.Exec(
		"UPDATE recovery_codes SET security_code = ? WHERE id = ?",
		securityCode, recoveryCodeID,
	)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to save security code: " + err.Error()})
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, gin.H{"private_key": pk, "security_code": securityCode})
}

func PrivateKeyCheck(c *gin.Context, db *sql.DB) {
	userID := Services.GetUserIdFromContext(c)
	if userID == 0 {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusUnauthorized, Message: "Unauthorized"})
		c.Abort()
		return
	}

	var keyCount int
	_ = db.QueryRow("SELECT COUNT(*) FROM private_keys WHERE user_id = ?", userID).Scan(&keyCount)

	var messageCount int
	_ = db.QueryRow("SELECT COUNT(*) FROM direct_chat_messages WHERE user_id = ?", userID).Scan(&messageCount)

	c.JSON(http.StatusOK, gin.H{"result": keyCount > 0 || messageCount > 0})
}

func UserAutocomplete(c *gin.Context, db *sql.DB) {
	term := c.Param("term")

	rows, err := db.Query(
		"SELECT id, username FROM users WHERE username LIKE ? AND user_status = 0 LIMIT 10",
		"%"+term+"%",
	)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get users: " + err.Error()})
		c.Abort()
		return
	}
	defer rows.Close()

	users := []Entities.ShortUser{}
	for rows.Next() {
		var u Entities.ShortUser
		if err := rows.Scan(&u.Id, &u.Username); err != nil {
			continue
		}
		users = append(users, u)
	}

	c.JSON(http.StatusOK, users)
}
