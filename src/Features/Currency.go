package Features

import (
	"cuento-backend/src/Services"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type CurrencyFeature struct {
}

func (CurrencyFeature) Name() string {
	return "Currency"
}

func (CurrencyFeature) Description() string {
	return "Currency feature"
}

func (CurrencyFeature) Key() string {
	return "currency"
}

func (CurrencyFeature) IsActive() bool {
	return false
}

type Currency struct {
	Name string
}

type CurrencyTransactionType int

const (
	CurrencyTransactionIncome CurrencyTransactionType = 1
	CurrencyTransactionSpend  CurrencyTransactionType = -1
)

type CurrencyTransactionStatus int

const (
	CurrencyTransactionPending  CurrencyTransactionStatus = 0
	CurrencyTransactionApproved CurrencyTransactionStatus = 1
	CurrencyTransactionRejected CurrencyTransactionStatus = 2
)

type CurrencyIncomeType struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Amount      int    `json:"amount"`
	IsActive    bool   `json:"is_active"`
}

var currencyIncomeTypeMeta = map[string]struct{ Name, Description string }{
	"currency_income_game_post":        {Name: "Game post", Description: "Currency for every game post"},
	"currency_income_wanted_character": {Name: "Wanted character", Description: "Currency for wanted character"},
}

func (CurrencyIncomeType) GetIncomeTypes(db *sql.DB) []CurrencyIncomeType {
	rows, err := db.Query("SELECT `key`, amount, is_active FROM currency_income_types")
	if err != nil {
		return []CurrencyIncomeType{}
	}
	defer rows.Close()

	var result []CurrencyIncomeType
	for rows.Next() {
		var t CurrencyIncomeType
		if err := rows.Scan(&t.Key, &t.Amount, &t.IsActive); err != nil {
			continue
		}
		if meta, ok := currencyIncomeTypeMeta[t.Key]; ok {
			t.Name = meta.Name
			t.Description = meta.Description
		}
		result = append(result, t)
	}
	return result
}

func IsCurrencyActive(db *sql.DB) bool {
	var isActive bool
	if err := db.QueryRow("SELECT is_active FROM features WHERE `key` = 'currency'").Scan(&isActive); err != nil {
		return false
	}
	return isActive
}

func GetCurrencyIncomeTypesHandler(c *gin.Context, db *sql.DB) {
	c.JSON(http.StatusOK, CurrencyIncomeType{}.GetIncomeTypes(db))
}

func GetCurrencySettingsHandler(c *gin.Context, db *sql.DB) {
	var name *string
	var iconURL *string
	err := db.QueryRow("SELECT currency_name, icon_url FROM currency_settings LIMIT 1").Scan(&name, &iconURL)
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get currency settings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"currency_name": name, "icon_url": iconURL})
}

func UpdateCurrencySettingsHandler(c *gin.Context, db *sql.DB) {
	var req struct {
		CurrencyName *string `json:"currency_name"`
		IconURL      *string `json:"icon_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}
	if req.CurrencyName == nil && req.IconURL == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one field must be provided"})
		return
	}
	if req.CurrencyName != nil {
		if _, err := db.Exec("UPDATE currency_settings SET currency_name = ?", *req.CurrencyName); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update currency name"})
			return
		}
	}
	if req.IconURL != nil {
		if _, err := db.Exec("UPDATE currency_settings SET icon_url = ?", *req.IconURL); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update currency icon"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "Currency settings updated"})
}

type CurrencyTransaction struct {
	Id                int                       `json:"id"`
	UserID            int                       `json:"user_id"`
	Type              CurrencyTransactionType   `json:"type"`
	Amount            int                       `json:"amount"`
	Datetime          time.Time                 `json:"datetime"`
	DatetimeLocalized string                    `json:"datetime_localized"`
	Status            CurrencyTransactionStatus `json:"status"`
	IncomeTypeKey     *string                   `json:"income_type_key"`
	Metadata          *json.RawMessage          `json:"metadata"`
}

func GetUserCurrencyTransactionsHandler(c *gin.Context, db *sql.DB) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id"})
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	const perPage = 20
	offset := (page - 1) * perPage

	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM currency_user_transactions WHERE user_id = ?", userID).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count transactions"})
		return
	}
	totalPages := (total + perPage - 1) / perPage

	rows, err := db.Query(
		"SELECT id, user_id, type, amount, datetime, status, income_type_key, metadata FROM currency_user_transactions WHERE user_id = ? ORDER BY datetime DESC LIMIT ? OFFSET ?",
		userID, perPage, offset,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
		return
	}
	defer rows.Close()

	tz := Services.GetUserTimezone(userID, db)

	items := []CurrencyTransaction{}
	for rows.Next() {
		var t CurrencyTransaction
		if err := rows.Scan(&t.Id, &t.UserID, &t.Type, &t.Amount, &t.Datetime, &t.Status, &t.IncomeTypeKey, &t.Metadata); err != nil {
			continue
		}
		t.DatetimeLocalized = Services.LocalizeTime(t.Datetime, tz)
		items = append(items, t)
	}

	c.JSON(http.StatusOK, gin.H{"items": items, "total_pages": totalPages})
}

func GetUserCurrencyAmountHandler(c *gin.Context, db *sql.DB) {
	userID, _ := c.Get("user_id")
	var amount int
	err := db.QueryRow("SELECT amount FROM currency_user_account WHERE user_id = ?", userID).Scan(&amount)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, gin.H{"amount": 0})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get currency amount"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"amount": amount})
}

func UpdateCurrencyIncomeTypesHandler(c *gin.Context, db *sql.DB) {
	var req []struct {
		Key      string `json:"key" binding:"required"`
		Amount   int    `json:"amount"`
		IsActive bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}
	for _, item := range req {
		if _, err := db.Exec(
			"UPDATE currency_income_types SET amount = ?, is_active = ? WHERE `key` = ?",
			item.Amount, item.IsActive, item.Key,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update income type: " + item.Key})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "Income types updated"})
}
