package test

import (
	"cuento-backend/src/Services"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetUserIdFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("User ID exists in context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", 123)

		userID := Services.GetUserIdFromContext(c)
		if userID != 123 {
			t.Errorf("Expected user ID 123, got %d", userID)
		}
	})

	t.Run("User ID does not exist in context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())

		userID := Services.GetUserIdFromContext(c)
		if userID != 0 {
			t.Errorf("Expected user ID 0, got %d", userID)
		}
	})

	t.Run("User ID is not an integer", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", "not-an-int")

		userID := Services.GetUserIdFromContext(c)
		if userID != 0 {
			t.Errorf("Expected user ID 0 for non-integer value, got %d", userID)
		}
	})
}
