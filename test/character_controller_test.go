package test

import (
	"bytes"
	"cuento-backend/src/Controllers"
	"cuento-backend/src/Entities"
	"cuento-backend/src/Middlewares"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func TestGetCharacter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer db.Close()

	t.Run("Successful GetCharacter", func(t *testing.T) {
		characterID := 1
		now := time.Now()

		// Mock GetEntity (character)
		mock.ExpectQuery("SELECT config FROM custom_field_config").
			WillReturnRows(sqlmock.NewRows([]string{"config"}).AddRow("[]"))

		mock.ExpectQuery("SELECT \\* FROM character_base").
			WithArgs(int64(characterID)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "avatar", "character_status", "topic_id", "total_episodes"}).
				AddRow(characterID, 1, "Test Character", nil, 0, 10, 5))

		// Mock episodes fetch
		mock.ExpectQuery("SELECT e.id, e.name, e.topic_id, t.date_last_post").
			WithArgs(characterID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "topic_id", "date_last_post", "last_post_author_username"}).
				AddRow(1, "Episode 1", 11, &now, "author"))

		// Mock characters for episode 1
		mock.ExpectQuery("SELECT cb.id, cb.name FROM character_base").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(characterID, "Test Character"))

		// Mock factions fetch
		mock.ExpectQuery("SELECT f.id").
			WithArgs(characterID).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		mock.ExpectQuery("SELECT id, name, parent_id, level, description, icon, show_on_profile FROM factions").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "parent_id", "level", "description", "icon", "show_on_profile"}).
				AddRow(1, "Faction A", nil, 0, nil, nil, true))

		// Mock CanEdit check
		mock.ExpectQuery("SELECT subforum_id FROM topics WHERE id = ?").
			WithArgs(10).
			WillReturnRows(sqlmock.NewRows([]string{"subforum_id"}).AddRow(1))

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM role_permission").
			WithArgs(0, "subforum_edit_others_topic:1").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		mock.ExpectQuery("SELECT author_user_id FROM topics WHERE id = ?").
			WithArgs(10).
			WillReturnRows(sqlmock.NewRows([]string{"author_user_id"}).AddRow(1))

		r := gin.New()
		r.Use(Middlewares.ErrorMiddleware())
		r.GET("/character/:id", func(c *gin.Context) {
			Controllers.GetCharacter(c, db)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/character/1", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var response Entities.Character
		json.Unmarshal(w.Body.Bytes(), &response)
		if response.Name != "Test Character" {
			t.Errorf("Expected character name 'Test Character', got %s", response.Name)
		}
	})
}

func TestCreateCharacter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer db.Close()

	t.Run("Successful CreateCharacter", func(t *testing.T) {
		reqBody := Controllers.CreateCharacterRequest{
			SubforumID: 1,
			Name:       "New Character",
			FactionIDs: []Entities.Faction{{Id: 1}},
		}
		body, _ := json.Marshal(reqBody)

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO topics").
			WithArgs(1, "New Character", 1, Entities.CharacterSheetTopic, 1).
			WillReturnResult(sqlmock.NewResult(10, 1))

		mock.ExpectExec("INSERT INTO character_base").
			WithArgs(1, "New Character", nil, 2, 10, 0). // 2 = PendingCharacter
			WillReturnResult(sqlmock.NewResult(1, 1))

		// getColumnTypes is skipped because CustomFields is empty

		// Mock GetEntity after create
		mock.ExpectQuery("SELECT config FROM custom_field_config").
			WillReturnRows(sqlmock.NewRows([]string{"config"}).AddRow("[]"))
		mock.ExpectQuery("SELECT \\* FROM character_base").
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "avatar", "character_status", "topic_id", "total_episodes"}).
				AddRow(1, 1, "New Character", nil, 2, 10, 0))

		mock.ExpectExec("INSERT INTO character_faction").
			WithArgs(1, 1).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectCommit()

		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("user_id", 1)
			c.Next()
		})
		r.Use(Middlewares.ErrorMiddleware())
		r.POST("/character", func(c *gin.Context) {
			Controllers.CreateCharacter(c, db)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/character", bytes.NewBuffer(body))
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
		}
	})
}
