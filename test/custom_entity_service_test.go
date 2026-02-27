package test

import (
	"cuento-backend/src/Services"
	"testing"
)

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"TopicId", "topic_id"},
		{"UserId", "user_id"},
		{"CharacterStatus", "character_status"},
		{"Name", "name"},
		{"Avatar", "avatar"},
		{"TotalEpisodes", "total_episodes"},
		{"Already_Snake", "already_snake"},
		{"MultipleWordsInPascalCase", "multiple_words_in_pascal_case"},
	}

	for _, test := range tests {
		result := Services.ToSnakeCase(test.input)
		if result != test.expected {
			t.Errorf("ToSnakeCase(%s) = %s; expected %s", test.input, result, test.expected)
		}
	}
}
