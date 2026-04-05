package Services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// widgetRegistry maps the func column value to the actual Go function.
var widgetRegistry = map[string]func(map[string]interface{}, *sql.DB) (string, error){
	"WidgetLastPost":       WidgetLastPost,
	"WidgetRandomEntities": WidgetRandomEntities,
}

func WidgetLastPost(config map[string]interface{}, db *sql.DB) (string, error) {
	topicID, err := extractIntValue(config, "topic_id")
	if err != nil {
		return "", err
	}

	var content string
	err = db.QueryRow(
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
	var configTemplateJSON sql.NullString
	err = db.QueryRow("SELECT func, config_template FROM widget_types WHERE id = ?", templateID).Scan(&funcName, &configTemplateJSON)
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

	// For fields marked can_empty in the template, inject a zero default if missing from config.
	if configTemplateJSON.Valid && configTemplateJSON.String != "" {
		var tmpl map[string]map[string]interface{}
		if err := json.Unmarshal([]byte(configTemplateJSON.String), &tmpl); err == nil {
			for fieldName, fieldMeta := range tmpl {
				if canEmpty, ok := fieldMeta["can_empty"].(bool); ok && canEmpty {
					if _, exists := config[fieldName]; !exists {
						switch fieldMeta["type"] {
						case "int":
							config[fieldName] = 0
						default:
							config[fieldName] = ""
						}
					}
				}
			}
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

var safeFieldName = regexp.MustCompile(`^[a-z0-9_]+$`)

func WidgetRandomEntities(config map[string]interface{}, db *sql.DB) (string, error) {
	number, err := extractIntValue(config, "number")
	if err != nil {
		return "", err
	}

	entityType, err := extractStringValue(config, "entity_type")
	if err != nil {
		return "", err
	}
	if entityType != "character" && entityType != "wanted_character" {
		return "", fmt.Errorf("unsupported entity_type: %s", entityType)
	}

	field1, err := extractStringValue(config, "entity_field_1")
	if err != nil {
		return "", err
	}
	field2, err := extractStringValue(config, "entity_field_2")
	if err != nil {
		return "", err
	}

	// Sanitize field names to prevent SQL injection
	if !safeFieldName.MatchString(field1) {
		return "", fmt.Errorf("invalid field name: %s", field1)
	}
	hasField2 := field2 != ""
	if hasField2 && !safeFieldName.MatchString(field2) {
		return "", fmt.Errorf("invalid field name: %s", field2)
	}

	selectFields := fmt.Sprintf("f.%s", field1)
	if hasField2 {
		selectFields += fmt.Sprintf(", f.%s", field2)
	}

	query := fmt.Sprintf(`
		SELECT b.id, b.name, %s
		FROM %s_base b
		JOIN %s_flattened f ON b.id = f.entity_id
		ORDER BY RAND()
		LIMIT ?`,
		selectFields, entityType, entityType,
	)

	rows, err := db.Query(query, number)
	if err != nil {
		return "", fmt.Errorf("failed to query random entities: %w", err)
	}
	defer rows.Close()

	// Look up content_field_type for the selected fields from entity config.
	field1IsImage := false
	field2IsImage := false
	if fieldConfigs, err := GetFieldConfig(entityType, db); err == nil {
		fieldTypeMap := make(map[string]string, len(fieldConfigs))
		for _, fc := range fieldConfigs {
			fieldTypeMap[fc.MachineFieldName] = fc.ContentFieldType
		}
		field1IsImage = fieldTypeMap[field1] == "image"
		field2IsImage = hasField2 && fieldTypeMap[field2] == "image"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<div class="widget-grid widget-grid--cols-%d">`, number))
	for rows.Next() {
		var id int
		var name string
		var val1 sql.NullString
		var val2 sql.NullString

		var scanArgs []interface{}
		scanArgs = append(scanArgs, &id, &name, &val1)
		if hasField2 {
			scanArgs = append(scanArgs, &val2)
		}
		if err := rows.Scan(scanArgs...); err != nil {
			continue
		}
		sb.WriteString(`<div class="widget-grid__item">`)
		sb.WriteString(fmt.Sprintf(`<div class="widget-grid__name">%s</div>`, name))
		if val1.Valid {
			if field1IsImage {
				sb.WriteString(fmt.Sprintf(`<div class="widget-grid__field"><img src="%s" alt="" /></div>`, val1.String))
			} else {
				sb.WriteString(fmt.Sprintf(`<div class="widget-grid__field">%s</div>`, val1.String))
			}
		}
		if hasField2 && val2.Valid {
			if field2IsImage {
				sb.WriteString(fmt.Sprintf(`<div class="widget-grid__field"><img src="%s" alt="" /></div>`, val2.String))
			} else {
				sb.WriteString(fmt.Sprintf(`<div class="widget-grid__field">%s</div>`, val2.String))
			}
		}
		sb.WriteString(`</div>`)
	}
	sb.WriteString(`</div>`)

	return sb.String(), nil
}

func extractIntValue(config map[string]interface{}, key string) (int, error) {
	raw, ok := config[key]
	if !ok {
		return 0, fmt.Errorf("missing %s in config", key)
	}
	switch v := raw.(type) {
	case int:
		return v, nil
	case float64:
		return int(v), nil
	}
	return 0, fmt.Errorf("missing or invalid %s value in config", key)
}

func extractStringValue(config map[string]interface{}, key string) (string, error) {
	raw, ok := config[key]
	if !ok {
		return "", fmt.Errorf("missing %s in config", key)
	}
	if v, ok := raw.(string); ok {
		return v, nil
	}
	return "", fmt.Errorf("missing or invalid %s value in config", key)
}
