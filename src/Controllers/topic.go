package Controllers

import (
	"cuento-backend/src/Entities"
	"cuento-backend/src/Events"
	"cuento-backend/src/Middlewares"
	"cuento-backend/src/Services"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type ViewforumRow struct {
	Id                     int                  `json:"id"`
	Status                 Entities.TopicStatus `json:"status"`
	Name                   string               `json:"name"`
	Type                   Entities.TopicType   `json:"type"`
	DateLastPost           *time.Time           `json:"date_last_post"`
	PostNumber             int                  `json:"post_number"`
	AuthorUserId           int                  `json:"author_user_id"`
	AuthorUsername         string               `json:"author_username"`
	LastPostAuthorUserId   *int                 `json:"last_post_author_user_id"`
	LastPostAuthorUsername *string              `json:"last_post_author_username"`
	LastPostId             *int                 `json:"last_post_id"`
	NotViewed              bool                 `json:"not_viewed"`
	LastViewedId           *int                 `json:"last_viewed_id"`
}

type CreateTopicRequest struct {
	SubforumId int    `json:"subforum_id" binding:"required"`
	Title      string `json:"title" binding:"required"`
	Content    string `json:"content" binding:"required"`
}

type CreatePostRequest struct {
	TopicID             int     `json:"topic_id" binding:"required"`
	Content             string  `json:"content" binding:"required"`
	UseCharacterProfile bool    `json:"use_character_profile"`
	CharacterProfileID  *int    `json:"character_profile_id"`
	GuestName           *string `json:"guest_name"`
}

type UpdatePostRequest struct {
	Content string `json:"content" binding:"required"`
}

type UpdateTopicRequest struct {
	Name string `json:"name" binding:"required"`
}

type PostRow struct {
	Id             int       `json:"id"`
	AuthorUserId   int       `json:"author_user_id"`
	AuthorUsername string    `json:"author_username"`
	Content        string    `json:"content"`
	ContentHtml    string    `json:"content_html"`
	DatePosted     time.Time `json:"date_posted"`
}

func GetTopicsBySubforum(c *gin.Context, db *sql.DB) {
	subforumStr := c.Param("subforum")
	pageStr := c.Param("page")
	subforum64, err := strconv.ParseInt(subforumStr, 10, 0)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Incorrect subforum parameter"})
		return
	}
	subforum := int(subforum64)
	page64, err := strconv.ParseInt(pageStr, 10, 0)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Incorrect page parameter"})
		return
	}
	page := int(page64) - 1

	userID := Services.GetUserIdFromContext(c)

	var topics []ViewforumRow

	limit := 30
	query := `
		SELECT topics.id, topics.status, topics.name, topics.type, topics.date_last_post, topics.post_number, 
		       topics.author_user_id, u.username as author_username, 
		       topics.last_post_author_user_id, u2.username as last_post_author_username, 
		       COALESCE(topics.last_post_id, (SELECT MAX(id) FROM posts WHERE topic_id = topics.id)) as last_post_id,
		       (CASE WHEN ? != 0 AND (utv.post_id IS NULL OR utv.post_id < COALESCE(topics.last_post_id, (SELECT MAX(id) FROM posts WHERE topic_id = topics.id))) THEN 1 ELSE 0 END) as not_viewed,
		       utv.post_id as last_viewed_id
		FROM topics 
		JOIN users u ON topics.author_user_id = u.id 
		LEFT JOIN users u2 ON topics.last_post_author_user_id = u2.id 
		LEFT JOIN user_topic_view utv ON topics.id = utv.topic_id AND utv.user_id = ?
		WHERE subforum_id = ? 
		ORDER BY date_last_post DESC
		LIMIT ? OFFSET ?
	`
	rows, err := db.Query(query, userID, userID, subforum, limit, page*limit)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get topics: " + err.Error()})
		return
	}

	defer rows.Close()

	for rows.Next() {
		var topic ViewforumRow
		err := rows.Scan(
			&topic.Id,
			&topic.Status,
			&topic.Name,
			&topic.Type,
			&topic.DateLastPost,
			&topic.PostNumber,
			&topic.AuthorUserId,
			&topic.AuthorUsername,
			&topic.LastPostAuthorUserId,
			&topic.LastPostAuthorUsername,
			&topic.LastPostId,
			&topic.NotViewed,
			&topic.LastViewedId,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan topics: " + err.Error()})
			return
		}
		topics = append(topics, topic)
	}

	c.JSON(http.StatusOK, topics)
}

func CreateTopic(c *gin.Context, db *sql.DB) {
	var req CreateTopicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	userID := Services.GetUserIdFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var username string
	err := db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user details: " + err.Error()})
		return
	}

	var authorUsername string
	// Fetch username for notification purposes
	_ = db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&authorUsername)

	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Insert Topic
	res, err := tx.Exec("INSERT INTO topics (subforum_id, name, author_user_id, date_created, date_last_post, status, type, post_number, last_post_author_user_id) VALUES (?, ?, ?, NOW(), NOW(), 0, 0, 1, ?)",
		req.SubforumId, req.Title, userID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert topic: " + err.Error()})
		return
	}
	topicID, err := res.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get topic ID"})
		return
	}

	// Insert Post
	res, err = tx.Exec("INSERT INTO posts (topic_id, author_user_id, content, date_created) VALUES (?, ?, ?, NOW())",
		topicID, userID, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert post: " + err.Error()})
		return
	}
	postID, err := res.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get post ID"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Publish event to update stats asynchronously
	Events.Publish(db, Events.TopicCreated, Events.TopicCreatedEvent{
		Type:       "topic_created",
		TopicID:    topicID,
		SubforumID: req.SubforumId,
		Title:      req.Title,
		PostID:     postID,
		UserID:     userID,
		Username:   username,
	})

	c.JSON(http.StatusCreated, gin.H{"message": "Topic created successfully", "topic_id": topicID})
}

func GetPostsByTopic(c *gin.Context, db *sql.DB) {
	topicIDStr := c.Param("id")
	pageStr := c.Query("page")
	postIDStr := c.Query("post_id")

	topicID, err := strconv.Atoi(topicIDStr)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid topic ID"})
		c.Abort()
		return
	}

	postsPerPage := Services.GetPostsPerPage(db)
	page := 1

	if pageStr != "" {
		page, _ = strconv.Atoi(pageStr)
	} else if postIDStr != "" {
		postID, err := strconv.Atoi(postIDStr)
		if err == nil {
			// Determine the page based on the post's position in the topic
			var position int
			query := "SELECT COUNT(*) FROM posts WHERE topic_id = ? AND id <= ?"
			err = db.QueryRow(query, topicID, postID).Scan(&position)
			if err == nil {
				page = int(math.Ceil(float64(position) / float64(postsPerPage)))
			}
		}
	}

	if page < 1 {
		page = 1
	}

	offset := (page - 1) * postsPerPage

	// 1. Get custom field columns from the config table
	var configJSON string
	err = db.QueryRow("SELECT config FROM custom_field_config WHERE entity_type = 'character_profile'").Scan(&configJSON)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get character profile config: " + err.Error()})
		c.Abort()
		return
	}

	var customConfig []Entities.CustomFieldConfig
	if err := json.Unmarshal([]byte(configJSON), &customConfig); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to parse character profile config: " + err.Error()})
		c.Abort()
		return
	}

	var flattenedCols []string
	for _, field := range customConfig {
		flattenedCols = append(flattenedCols, "cpf."+field.MachineFieldName)
	}

	colsSelect := ""
	if len(flattenedCols) > 0 {
		colsSelect = ", " + strings.Join(flattenedCols, ", ")
	}

	// 2. Construct the main query
	query := fmt.Sprintf(`
		SELECT
			p.id, p.author_user_id, p.date_created, p.content, p.use_character_profile,
			u.username, u.avatar, p.guest_name,
			cp.id as character_profile_id, cp.character_id, cb.name as character_name, cp.avatar as character_avatar,
			t.subforum_id
			%s
		FROM posts p
		JOIN topics t ON p.topic_id = t.id
		LEFT JOIN users u ON p.author_user_id = u.id
		LEFT JOIN character_profile_base cp ON p.character_profile_id = cp.id
		LEFT JOIN character_base cb ON cp.character_id = cb.id
		LEFT JOIN character_profile_flattened cpf ON cp.id = cpf.entity_id
		WHERE p.topic_id = ?
		ORDER BY p.date_created ASC
		LIMIT ? OFFSET ?
	`, colsSelect)

	rows, err := db.Query(query, topicID, postsPerPage, offset)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get posts: " + err.Error()})
		c.Abort()
		return
	}
	defer rows.Close()

	// 3. Scan and process results
	cols, _ := rows.Columns()
	posts := make([]Entities.Post, 0) // Initialize slice

	currentUserID := Services.GetUserIdFromContext(c)
	var subforumID int

	for rows.Next() {
		// Scan into a map
		values := make([]interface{}, len(cols))
		for i := range values {
			values[i] = new(sql.RawBytes)
		}
		if err := rows.Scan(values...); err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to scan post data: " + err.Error()})
			c.Abort()
			return
		}

		rowMap := make(map[string]interface{})
		for i, colName := range cols {
			val := *(values[i].(*sql.RawBytes))
			if val != nil {
				rowMap[colName] = string(val) // Store as string for now
			}
		}

		// Populate Post struct from map
		var post Entities.Post
		post.Id, _ = strconv.Atoi(rowMap["id"].(string))
		post.AuthorUserId, _ = strconv.Atoi(rowMap["author_user_id"].(string))
		post.DateCreated, _ = time.Parse("2006-01-02 15:04:05", rowMap["date_created"].(string))
		post.Content = rowMap["content"].(string)
		post.ContentHtml = Entities.ParseBBCode(post.Content)
		post.UseCharacterProfile, _ = strconv.ParseBool(rowMap["use_character_profile"].(string))
		subforumID, _ = strconv.Atoi(rowMap["subforum_id"].(string))

		// Check CanEdit
		canEdit := false
		if currentUserID != 0 {
			if currentUserID == post.AuthorUserId {
				// Check for "Edit own post" permission in this subforum
				permission := fmt.Sprintf("subforum_edit_own_post:%d", subforumID)
				if hasPerm, err := Services.HasPermission(currentUserID, permission, db); err == nil && hasPerm {
					canEdit = true
				}
			} else {
				// Check for "Edit others' post" permission in this subforum
				permission := fmt.Sprintf("subforum_edit_others_post:%d", subforumID)
				if hasPerm, err := Services.HasPermission(currentUserID, permission, db); err == nil && hasPerm {
					canEdit = true
				}
			}
		}
		post.CanEdit = &canEdit

		if post.UseCharacterProfile {
			var charProfile Entities.CharacterProfile
			if id, ok := rowMap["character_profile_id"]; ok {
				charProfile.Id, _ = strconv.Atoi(id.(string))
			}
			if id, ok := rowMap["character_id"]; ok {
				charProfile.CharacterId, _ = strconv.Atoi(id.(string))
			}
			if name, ok := rowMap["character_name"]; ok {
				charProfile.CharacterName = name.(string)
			}
			if avatar, ok := rowMap["character_avatar"]; ok {
				avatarStr := avatar.(string)
				charProfile.Avatar = &avatarStr
			}

			// Populate custom fields
			customFields := make(map[string]Entities.CustomFieldValue)
			for _, field := range customConfig {
				if val, ok := rowMap[field.MachineFieldName]; ok {
					cfValue := Entities.CustomFieldValue{Content: val}
					if field.FieldType == "text" {
						if s, ok := val.(string); ok {
							cfValue.ContentHtml = Entities.ParseBBCode(s)
						}
					}
					customFields[field.MachineFieldName] = cfValue
				}
			}
			charProfile.CustomFields.CustomFields = customFields
			charProfile.CustomFields.FieldConfig = customConfig // Add this line
			post.CharacterProfile = &charProfile
		} else {
			var userProfile Entities.UserProfile
			userProfile.UserId = post.AuthorUserId
			if guestName, ok := rowMap["guest_name"]; ok && guestName.(string) != "" {
				userProfile.UserName = guestName.(string)
			} else if username, ok := rowMap["username"]; ok {
				userProfile.UserName = username.(string)
			}
			if avatar, ok := rowMap["avatar"]; ok {
				userProfile.Avatar = avatar.(string)
			}
			post.UserProfile = &userProfile
		}
		posts = append(posts, post)
	}

	c.JSON(http.StatusOK, gin.H{
		"page":  page,
		"posts": posts,
	})
}

func GetTopic(c *gin.Context, db *sql.DB) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid ID"})
		c.Abort()
		return
	}

	var topic Entities.Topic
	query := "SELECT t.id, t.status, t.name, t.type, t.date_created, t.date_last_post, t.post_number, t.author_user_id, u.username, t.last_post_author_user_id, u2.username, t.subforum_id FROM topics t JOIN users u ON t.author_user_id = u.id LEFT JOIN users u2 ON t.last_post_author_user_id = u2.id WHERE t.id = ?"
	err = db.QueryRow(query, id).Scan(
		&topic.Id,
		&topic.Status,
		&topic.Name,
		&topic.Type,
		&topic.DateCreated,
		&topic.DateLastPost,
		&topic.PostNumber,
		&topic.AuthorUserId,
		&topic.AuthorUsername,
		&topic.LastPostAuthorUserId,
		&topic.LastPostAuthorName,
		&topic.SubforumId,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusNotFound, Message: "Topic not found"})
		} else {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get topic: " + err.Error()})
		}
		c.Abort()
		return
	}

	// Check CanEdit
	currentUserID := Services.GetUserIdFromContext(c)
	// Get all permissions for this user in this subforum context
	userPerms, err := Services.GetSubforumPermissions(currentUserID, topic.SubforumId, db)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get subforum permissions: " + err.Error()})
		c.Abort()
		return
	}

	// Populate the topic permissions for the frontend
	topic.Permissions = userPerms

	canEdit := false
	isAuthor := currentUserID != 0 && currentUserID == topic.AuthorUserId

	// Directly check the boolean fields of the SubforumPermissions struct
	if userPerms != nil {
		if (isAuthor && userPerms.SubforumEditOwnTopic) || userPerms.SubforumEditOthersTopic {
			canEdit = true
		}
	}

	topic.CanEdit = &canEdit

	if topic.Type == Entities.EpisodeTopic {
		var episodeID int
		err := db.QueryRow("SELECT id FROM episode_base WHERE topic_id = ?", topic.Id).Scan(&episodeID)
		if err != nil {
			if err != sql.ErrNoRows {
				_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get episode ID for topic: " + err.Error()})
				c.Abort()
			}
			c.JSON(http.StatusOK, topic)
			return
		}

		entity, err := Services.GetEntity(int64(episodeID), "episode", db)
		if err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get episode entity: " + err.Error()})
			c.Abort()
			return
		}

		if episode, ok := entity.(*Entities.Episode); ok {
			// Fetch characters for the episode
			charRows, err := db.Query("SELECT cb.id, cb.name FROM character_base cb JOIN episode_character ec ON cb.id = ec.character_id WHERE ec.episode_id = ?", episode.Id)
			if err == nil {
				var characters []*Entities.ShortCharacter
				for charRows.Next() {
					var char Entities.ShortCharacter
					if err := charRows.Scan(&char.Id, &char.Name); err == nil {
						characters = append(characters, &char)
					}
				}
				episode.Characters = characters
				charRows.Close()
			}
			topic.Episode = episode
		}
	}

	if topic.Type == Entities.CharacterSheetTopic {
		var characterID int
		err := db.QueryRow("SELECT id FROM character_base WHERE topic_id = ?", topic.Id).Scan(&characterID)
		if err != nil {
			if err != sql.ErrNoRows {
				// This is not an error, it just means the topic is not linked to a character sheet yet
				// _ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get character sheet ID for topic: " + err.Error()})
				// c.Abort()
			}
		} else {
			entity, err := Services.GetEntity(int64(characterID), "character", db)
			if err != nil {
				_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get character entity: " + err.Error()})
				c.Abort()
				return
			}
			if character, ok := entity.(*Entities.Character); ok {
				character.Factions, _ = Services.GetFactionTreeByCharacter(characterID, db)
				topic.Character = character
			}
		}
	}

	c.JSON(http.StatusOK, topic)
}

func CreatePost(c *gin.Context, db *sql.DB) {
	var req CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	userID := Services.GetUserIdFromContext(c)

	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	var guestName *string
	if userID == 0 {
		guestName = req.GuestName
	}

	// Insert Post
	res, err := tx.Exec("INSERT INTO posts (topic_id, author_user_id, content, date_created, use_character_profile, character_profile_id, guest_name) VALUES (?, ?, ?, NOW(), ?, ?, ?)",
		req.TopicID, userID, req.Content, req.UseCharacterProfile, req.CharacterProfileID, guestName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert post: " + err.Error()})
		return
	}
	postID, err := res.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get post ID"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Handle Mentions
	// Regex to find @username (assuming alphanumeric + underscore)
	re := regexp.MustCompile(`@([a-zA-Z0-9_]+)`)
	matches := re.FindAllStringSubmatch(req.Content, -1)

	if len(matches) > 0 {
		seen := make(map[string]bool)
		var usernames []string
		for _, match := range matches {
			if len(match) > 1 {
				username := match[1]
				if !seen[username] {
					usernames = append(usernames, username)
					seen[username] = true
				}
			}
		}

		if len(usernames) > 0 {
			query := "SELECT id, username FROM users WHERE username IN (?" + strings.Repeat(",?", len(usernames)-1) + ")"
			args := make([]interface{}, len(usernames))
			for i, u := range usernames {
				args[i] = u
			}

			rows, err := db.Query(query, args...)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var mUserID int
					var mUsername string
					if err := rows.Scan(&mUserID, &mUsername); err == nil {
						if mUserID != userID {
							// Fetch author details for the mention
							var authorName string
							var authorCharacterID *int
							var authorCharacterName *string

							err = db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&authorName)
							if req.UseCharacterProfile && req.CharacterProfileID != nil {
								var charID int
								var charName string
								err = db.QueryRow("SELECT character_id, cb.name FROM character_profile_base cpb JOIN character_base cb ON cpb.character_id = cb.id WHERE cpb.id = ?", *req.CharacterProfileID).Scan(&charID, &charName)
								if err == nil {
									authorCharacterID = &charID
									authorCharacterName = &charName
								}
							}

							mentionData := Entities.NotificationMention{
								UserId:        userID,
								UserName:      authorName,
								CharacterId:   authorCharacterID,
								CharacterName: authorCharacterName,
								PostId:        int(postID),
								TopicId:       req.TopicID,
							}

							Events.Publish(db, Events.NotificationCreated, Events.NotificationEvent{
								UserID:  mUserID,
								Type:    "mention",
								Message: fmt.Sprintf("%s mentioned you in a post", authorName),
								Data:    mentionData,
							})
						}
					}
				}
			}
		}
	}

	// Fetch additional data for the event
	var subforumID int
	var topicName string
	err = db.QueryRow("SELECT subforum_id, name FROM topics WHERE id = ?", req.TopicID).Scan(&subforumID, &topicName)
	if err == nil {
		// Fetch full post data
		fullPost, postErr := Services.GetPostById(int(postID), db)
		if postErr != nil {
			fmt.Printf("Error getting post details for event publishing: %v\n", postErr)
		} else {
			// Publish event to broadcast to all users
			Events.Publish(db, Events.PostCreated, Events.PostCreatedEvent{
				Type:       "post_created",
				TopicID:    int64(req.TopicID),
				SubforumID: subforumID,
				Post:       *fullPost,
			})
		}
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Post created successfully", "post_id": postID})
}

func UpdatePost(c *gin.Context, db *sql.DB) {
	postID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid post ID"})
		c.Abort()
		return
	}

	var req UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid request body: " + err.Error()})
		c.Abort()
		return
	}

	userID := Services.GetUserIdFromContext(c)
	if userID == 0 {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusUnauthorized, Message: "Unauthorized"})
		c.Abort()
		return
	}

	// 1. Fetch post details to check ownership and subforum
	var authorUserID int
	var subforumID int
	query := `
		SELECT p.author_user_id, t.subforum_id 
		FROM posts p 
		JOIN topics t ON p.topic_id = t.id 
		WHERE p.id = ?
	`
	err = db.QueryRow(query, postID).Scan(&authorUserID, &subforumID)
	if err != nil {
		if err == sql.ErrNoRows {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusNotFound, Message: "Post not found"})
		} else {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to fetch post details: " + err.Error()})
		}
		c.Abort()
		return
	}

	// 2. Check permissions
	canEdit := false
	if userID == authorUserID {
		// Check for "Edit own post" permission
		permission := fmt.Sprintf("subforum_edit_own_post:%d", subforumID)
		if hasPerm, err := Services.HasPermission(userID, permission, db); err == nil && hasPerm {
			canEdit = true
		}
	} else {
		// Check for "Edit others' post" permission
		permission := fmt.Sprintf("subforum_edit_others_post:%d", subforumID)
		if hasPerm, err := Services.HasPermission(userID, permission, db); err == nil && hasPerm {
			canEdit = true
		}
	}

	if !canEdit {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusForbidden, Message: "You do not have permission to edit this post"})
		c.Abort()
		return
	}

	// 3. Update post content
	_, err = db.Exec("UPDATE posts SET content = ? WHERE id = ?", req.Content, postID)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to update post: " + err.Error()})
		c.Abort()
		return
	}

	// 4. Fetch updated post and emit event
	updatedPost, err := Services.GetPostById(postID, db)
	if err == nil {
		// We need topicID and subforumID for the event
		var topicID int64
		err = db.QueryRow("SELECT topic_id FROM posts WHERE id = ?", postID).Scan(&topicID)
		if err == nil {
			Events.Publish(db, Events.PostCreated, Events.PostCreatedEvent{ // Reusing PostCreatedEvent for broadcast
				Type:       "post_updated",
				TopicID:    topicID,
				SubforumID: subforumID,
				Post:       *updatedPost,
			})
		}
	}

	c.JSON(http.StatusOK, updatedPost)
}

func UpdateTopic(c *gin.Context, db *sql.DB) {
	topicID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid topic ID"})
		c.Abort()
		return
	}

	var req UpdateTopicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid request body: " + err.Error()})
		c.Abort()
		return
	}

	userID := Services.GetUserIdFromContext(c)
	if userID == 0 {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusUnauthorized, Message: "Unauthorized"})
		c.Abort()
		return
	}

	// 1. Fetch topic details to check ownership and subforum
	var authorUserID int
	var subforumID int
	var topicType Entities.TopicType
	query := "SELECT author_user_id, subforum_id, type FROM topics WHERE id = ?"
	err = db.QueryRow(query, topicID).Scan(&authorUserID, &subforumID, &topicType)
	if err != nil {
		if err == sql.ErrNoRows {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusNotFound, Message: "Topic not found"})
		} else {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to fetch topic details: " + err.Error()})
		}
		c.Abort()
		return
	}

	// Verify topic type is general
	if topicType != Entities.GeneralTopic {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Only general topics can be updated via this endpoint"})
		c.Abort()
		return
	}

	// 2. Check permissions
	canEdit := false
	if userID == authorUserID {
		// Check for "Edit own topic" permission
		permission := fmt.Sprintf("subforum_edit_own_topic:%d", subforumID)
		if hasPerm, err := Services.HasPermission(userID, permission, db); err == nil && hasPerm {
			canEdit = true
		}
	} else {
		// Check for "Edit others' topic" permission
		permission := fmt.Sprintf("subforum_edit_others_topic:%d", subforumID)
		if hasPerm, err := Services.HasPermission(userID, permission, db); err == nil && hasPerm {
			canEdit = true
		}
	}

	if !canEdit {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusForbidden, Message: "You do not have permission to edit this topic"})
		c.Abort()
		return
	}

	// 3. Update topic name
	_, err = db.Exec("UPDATE topics SET name = ? WHERE id = ?", req.Name, topicID)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to update topic: " + err.Error()})
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Topic updated successfully"})
}

func GetActiveTopics(c *gin.Context, db *sql.DB) {
	pageStr := c.Query("page")
	notViewedStr := c.Query("not_viewed")
	subforumIDsStr := c.Query("subforum_ids")

	page := 1
	if pageStr != "" {
		page, _ = strconv.Atoi(pageStr)
	}
	if page < 1 {
		page = 1
	}

	notViewed := false
	if notViewedStr == "true" {
		notViewed = true
	}

	var subforumIDs []int
	if subforumIDsStr != "" {
		for _, idStr := range strings.Split(subforumIDsStr, ",") {
			if id, err := strconv.Atoi(idStr); err == nil {
				subforumIDs = append(subforumIDs, id)
			}
		}
	}

	userID := Services.GetUserIdFromContext(c)

	// Get visible subforum IDs based on user permissions
	visibleSubforumIDs, err := Services.GetVisibleSubforums(userID, "subforum_read", db)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to determine visible subforums: " + err.Error()})
		c.Abort()
		return
	}

	// Filter subforumIDs based on visibility
	var filteredSubforumIDs []int
	if len(subforumIDs) > 0 {
		visibleMap := make(map[int]bool)
		for _, id := range visibleSubforumIDs {
			visibleMap[id] = true
		}
		for _, id := range subforumIDs {
			if visibleMap[id] {
				filteredSubforumIDs = append(filteredSubforumIDs, id)
			}
		}
	} else {
		filteredSubforumIDs = visibleSubforumIDs
	}

	if len(filteredSubforumIDs) == 0 {
		c.JSON(http.StatusOK, []ViewforumRow{})
		return
	}

	limit := Services.GetPostsPerPage(db)
	offset := (page - 1) * limit

	placeholders := strings.Repeat("?,", len(filteredSubforumIDs)-1) + "?"
	query := fmt.Sprintf(`
		SELECT t.id, t.status, t.name, t.type, t.date_last_post, t.post_number, 
		       t.author_user_id, u.username as author_username,
		       t.last_post_author_user_id, u2.username as last_post_author_username, 
		       COALESCE(t.last_post_id, (SELECT MAX(id) FROM posts WHERE topic_id = t.id)) as last_post_id,
		       (CASE WHEN ? != 0 AND (utv.post_id IS NULL OR utv.post_id < COALESCE(t.last_post_id, (SELECT MAX(id) FROM posts WHERE topic_id = t.id))) THEN 1 ELSE 0 END) as not_viewed,
		       utv.post_id as last_viewed_id
		FROM topics t
		JOIN users u ON t.author_user_id = u.id
		LEFT JOIN users u2 ON t.last_post_author_user_id = u2.id
		LEFT JOIN user_topic_view utv ON t.id = utv.topic_id AND utv.user_id = ?
		WHERE t.subforum_id IN (%s)
	`, placeholders)

	var args []interface{}
	args = append(args, userID, userID)
	for _, id := range filteredSubforumIDs {
		args = append(args, id)
	}

	if notViewed && userID != 0 {
		query += " AND (utv.post_id IS NULL OR utv.post_id < COALESCE(t.last_post_id, (SELECT MAX(id) FROM posts WHERE topic_id = t.id)))"
	}

	query += " AND t.date_last_post >= DATE_SUB(NOW(), INTERVAL 10 DAY)"

	query += " ORDER BY t.date_last_post DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get active topics: " + err.Error()})
		c.Abort()
		return
	}
	defer rows.Close()

	var topics []ViewforumRow
	for rows.Next() {
		var topic ViewforumRow
		if err := rows.Scan(
			&topic.Id,
			&topic.Status,
			&topic.Name,
			&topic.Type,
			&topic.DateLastPost,
			&topic.PostNumber,
			&topic.AuthorUserId,
			&topic.AuthorUsername,
			&topic.LastPostAuthorUserId,
			&topic.LastPostAuthorUsername,
			&topic.LastPostId,
			&topic.NotViewed,
			&topic.LastViewedId,
		); err != nil {
			continue
		}
		topics = append(topics, topic)
	}

	if topics == nil {
		topics = []ViewforumRow{}
	}

	c.JSON(http.StatusOK, topics)
}

func GetActiveTopicCount(c *gin.Context, db *sql.DB) {
	notViewedStr := c.Query("not_viewed")
	subforumIDsStr := c.Query("subforum_ids")

	notViewed := false
	if notViewedStr == "true" {
		notViewed = true
	}

	var subforumIDs []int
	if subforumIDsStr != "" {
		for _, idStr := range strings.Split(subforumIDsStr, ",") {
			if id, err := strconv.Atoi(idStr); err == nil {
				subforumIDs = append(subforumIDs, id)
			}
		}
	}

	userID := Services.GetUserIdFromContext(c)

	// Get visible subforum IDs based on user permissions
	visibleSubforumIDs, err := Services.GetVisibleSubforums(userID, "subforum_read", db)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to determine visible subforums: " + err.Error()})
		c.Abort()
		return
	}

	// Filter subforumIDs based on visibility
	var filteredSubforumIDs []int
	if len(subforumIDs) > 0 {
		visibleMap := make(map[int]bool)
		for _, id := range visibleSubforumIDs {
			visibleMap[id] = true
		}
		for _, id := range subforumIDs {
			if visibleMap[id] {
				filteredSubforumIDs = append(filteredSubforumIDs, id)
			}
		}
	} else {
		filteredSubforumIDs = visibleSubforumIDs
	}

	if len(filteredSubforumIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{"total": 0})
		return
	}

	placeholders := strings.Repeat("?,", len(filteredSubforumIDs)-1) + "?"
	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM topics t
		JOIN users u ON t.author_user_id = u.id
		LEFT JOIN user_topic_view utv ON t.id = utv.topic_id AND utv.user_id = ?
		WHERE t.subforum_id IN (%s)
	`, placeholders)

	var args []interface{}
	args = append(args, userID)
	for _, id := range filteredSubforumIDs {
		args = append(args, id)
	}

	if notViewed && userID != 0 {
		query += " AND (utv.post_id IS NULL OR utv.post_id < COALESCE(t.last_post_id, (SELECT MAX(id) FROM posts WHERE topic_id = t.id)))"
	}

	query += " AND t.date_last_post >= DATE_SUB(NOW(), INTERVAL 10 DAY)"

	var count int
	err = db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get active topic count: " + err.Error()})
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": count})
}
