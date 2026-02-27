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

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func TestGetEpisode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer db.Close()

	t.Run("Successful GetEpisode", func(t *testing.T) {
		episodeID := 1
		topicID := 10

		// Mock GetEntity (episode)
		mock.ExpectQuery("SELECT config FROM custom_field_config").
			WillReturnRows(sqlmock.NewRows([]string{"config"}).AddRow("[]"))

		mock.ExpectQuery("SELECT \\* FROM episode_base").
			WithArgs(int64(episodeID)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "topic_id", "name"}).
				AddRow(episodeID, topicID, "Test Episode"))

		// Mock characters for episode
		mock.ExpectQuery("SELECT cb.id, cb.name FROM character_base").
			WithArgs(episodeID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "Char 1"))

		// Mock CanEdit check
		mock.ExpectQuery("SELECT subforum_id FROM topics WHERE id = ?").
			WithArgs(topicID).
			WillReturnRows(sqlmock.NewRows([]string{"subforum_id"}).AddRow(1))

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM role_permission").
			WithArgs(0, "subforum_edit_others_topic:1").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		mock.ExpectQuery("SELECT author_user_id FROM topics WHERE id = ?").
			WithArgs(topicID).
			WillReturnRows(sqlmock.NewRows([]string{"author_user_id"}).AddRow(1))

		r := gin.New()
		r.Use(Middlewares.ErrorMiddleware())
		r.GET("/episode/:id", func(c *gin.Context) {
			Controllers.GetEpisode(c, db)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/episode/1", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var response Entities.Episode
		json.Unmarshal(w.Body.Bytes(), &response)
		if response.Name != "Test Episode" {
			t.Errorf("Expected episode name 'Test Episode', got %s", response.Name)
		}
	})
}

func TestCreateEpisode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer db.Close()

	t.Run("Successful CreateEpisode", func(t *testing.T) {
		reqBody := Controllers.CreateEpisodeRequest{
			SubforumID:   1,
			Name:         "New Episode",
			CharacterIDs: []int{1, 2},
		}
		body, _ := json.Marshal(reqBody)

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO topics").
			WithArgs(1, "New Episode", 1, Entities.EpisodeTopic, 1).
			WillReturnResult(sqlmock.NewResult(10, 1))

		mock.ExpectExec("INSERT INTO episode_base").
			WithArgs(10, "New Episode").
			WillReturnResult(sqlmock.NewResult(1, 1))

		// getColumnTypes is skipped because CustomFields is empty

		// Mock GetEntity after create
		mock.ExpectQuery("SELECT config FROM custom_field_config").
			WillReturnRows(sqlmock.NewRows([]string{"config"}).AddRow("[]"))
		mock.ExpectQuery("SELECT \\* FROM episode_base").
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "topic_id", "name"}).
				AddRow(1, 10, "New Episode"))

		mock.ExpectPrepare("INSERT INTO episode_character")
		mock.ExpectExec("INSERT INTO episode_character").WithArgs(1, 1).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("INSERT INTO episode_character").WithArgs(1, 2).WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectCommit()

		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("user_id", 1)
			c.Next()
		})
		r.Use(Middlewares.ErrorMiddleware())
		r.POST("/episode", func(c *gin.Context) {
			Controllers.CreateEpisode(c, db)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/episode", bytes.NewBuffer(body))
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
		}
	})
}
