package Services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
)

// widgetRegistry maps the func column value to the actual Go function.
var widgetRegistry = map[string]func(map[string]interface{}, *sql.DB) (string, error){
	"WidgetLastPost": WidgetLastPost,
}

func WidgetLastPost(config map[string]interface{}, db *sql.DB) (string, error) {
	topicIdRaw, ok := config["topic_id"]
	if !ok {
		return "", fmt.Errorf("missing topic_id in config")
	}

	var topicID int
	switch v := topicIdRaw.(type) {
	case int:
		topicID = v
	case float64:
		topicID = int(v)
	default:
		return "", fmt.Errorf("invalid topic_id type in config")
	}

	var content string
	err := db.QueryRow(
		"SELECT content FROM posts WHERE topic_id = ? AND (is_deleted IS NULL OR is_deleted <> 1) ORDER BY date_created DESC LIMIT 1",
		topicID,
	).Scan(&content)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("failed to get last post: %w", err)
	}

	return ParseBBCode(content), nil
}

// RenderWidget fetches a widget by ID, resolves its type, and calls the registered function.
func RenderWidget(id int, db *sql.DB) (string, error) {
	var templateID int
	var configJSON sql.NullString
	err := db.QueryRow("SELECT template_id, config FROM widgets WHERE id = ?", id).Scan(&templateID, &configJSON)
	if err != nil {
		return "", fmt.Errorf("widget %d not found: %w", id, err)
	}

	var funcName string
	err = db.QueryRow("SELECT func FROM widget_types WHERE id = ?", templateID).Scan(&funcName)
	if err != nil {
		return "", fmt.Errorf("widget type %d not found: %w", templateID, err)
	}

	fn, ok := widgetRegistry[funcName]
	if !ok {
		return "", fmt.Errorf("unknown widget function: %s", funcName)
	}

	config := make(map[string]interface{})
	if configJSON.Valid && configJSON.String != "" {
		if err := json.Unmarshal([]byte(configJSON.String), &config); err != nil {
			return "", fmt.Errorf("invalid widget config: %w", err)
		}
	}

	return fn(config, db)
}

var widgetTagRe = regexp.MustCompile(`\[widget=(\d+)\]`)

// RenderPanelContent parses BB code first, then replaces [widget=N] tags with rendered widget HTML.
func RenderPanelContent(content string, db *sql.DB) string {
	html := ParseBBCode(content)

	return widgetTagRe.ReplaceAllStringFunc(html, func(match string) string {
		sub := widgetTagRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		id, _ := strconv.Atoi(sub[1])
		rendered, err := RenderWidget(id, db)
		if err != nil {
			return fmt.Sprintf("<!-- widget %d error: %s -->", id, err.Error())
		}
		return rendered
	})
}
