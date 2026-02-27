package test

import (
	"cuento-backend/src/Services"
	"testing"
)

func TestSubforumPermissionsMap(t *testing.T) {
	expectedKeys := []string{
		"subforum_read",
		"subforum_create_general_topic",
		"subforum_create_episode_topic",
		"subforum_create_character_topic",
		"subforum_post",
		"subforum_delete_topic",
		"subforum_delete_others_topic",
		"subforum_edit_others_post",
		"subforum_edit_own_post",
		"subforum_edit_others_topic",
		"subforum_edit_own_topic",
	}

	for _, key := range expectedKeys {
		if _, ok := Services.SubforumPermissions[key]; !ok {
			t.Errorf("Expected permission key %s not found in SubforumPermissions", key)
		}
	}
}
