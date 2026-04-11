package Features

import (
	"database/sql"
	"net/http"

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

type CurrencyIncomeType struct {
	Name        string
	Description string
	IsActive    bool
	Amount      int
	Key         string
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

func GetCurrencyIncomeTypesHandler(c *gin.Context, db *sql.DB) {
	c.JSON(http.StatusOK, CurrencyIncomeType{}.GetIncomeTypes(db))
}

func GetCurrencyNameHandler(c *gin.Context, db *sql.DB) {
	var name string
	if err := db.QueryRow("SELECT currency_name FROM currency_settings LIMIT 1").Scan(&name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get currency name"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"currency_name": name})
}

func UpdateCurrencyNameHandler(c *gin.Context, db *sql.DB) {
	var req struct {
		CurrencyName string `json:"currency_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}
	if _, err := db.Exec("UPDATE currency_settings SET currency_name = ?", req.CurrencyName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update currency name"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"currency_name": req.CurrencyName})
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
