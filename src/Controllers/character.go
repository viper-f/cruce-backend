package Controllers

import (
	"cuento-backend/src/Entities"
	"cuento-backend/src/Events"
	"cuento-backend/src/Middlewares"
	"cuento-backend/src/Services"
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CreateCharacterRequest struct {
	SubforumID   int                                  `json:"subforum_id" binding:"required"`
	Name         string                               `json:"name" binding:"required"`
	Avatar       *string                              `json:"avatar"`
	CustomFields map[string]Entities.CustomFieldValue `json:"custom_fields"`
	FactionIDs   []Entities.Faction                   `json:"factions"`
}

func GetCharacter(c *gin.Context, db *sql.DB) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid Id"})
		c.Abort()
		return
	}

	entity, err := Services.GetEntity(int64(id), "character", db)
	if err != nil {
		if err == sql.ErrNoRows {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusNotFound, Message: "Character not found"})
		} else {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get character: " + err.Error()})
		}
		c.Abort()
		return
	}

	if character, ok := entity.(*Entities.Character); ok {
		// Fetch episodes for this character
		query := `
			SELECT e.id, e.name, e.topic_id, t.date_last_post, u.username as last_post_author_username
			FROM episode_base e
			JOIN episode_character ec ON e.id = ec.episode_id
			JOIN topics t ON e.topic_id = t.id
			LEFT JOIN users u ON t.last_post_author_user_id = u.id
			WHERE ec.character_id = ?
			ORDER BY t.date_last_post DESC
		`
		rows, err := db.Query(query, character.Id)
		if err == nil {
			defer rows.Close()
			var episodes []Entities.EpisodeListItem
			for rows.Next() {
				var ep Entities.EpisodeListItem
				if err := rows.Scan(&ep.Id, &ep.Name, &ep.TopicId, &ep.DateLastPost, &ep.LastPostAuthorUsername); err == nil {
					// Fetch all characters for this episode
					charRows, err := db.Query(`
						SELECT cb.id, cb.name 
						FROM character_base cb 
						JOIN episode_character ec ON cb.id = ec.character_id 
						WHERE ec.episode_id = ?`, ep.Id)
					if err == nil {
						var characters []*Entities.ShortCharacter
						for charRows.Next() {
							var char Entities.ShortCharacter
							if err := charRows.Scan(&char.Id, &char.Name); err == nil {
								characters = append(characters, &char)
							}
						}
						ep.Characters = characters
						charRows.Close()
					}
					episodes = append(episodes, ep)
				}
			}
			character.Episodes = episodes
		}

		// Fetch factions
		character.Factions, _ = Services.GetFactionTreeByCharacter(character.Id, db)

		c.JSON(http.StatusOK, character)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func CreateCharacter(c *gin.Context, db *sql.DB) {
	var req CreateCharacterRequest
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

	tx, err := db.Begin()
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to start transaction"})
		c.Abort()
		return
	}
	defer tx.Rollback()

	// Insert Topic (without first post)
	res, err := tx.Exec("INSERT INTO topics (subforum_id, name, author_user_id, date_created, date_last_post, status, type, post_number, last_post_author_user_id) VALUES (?, ?, ?, NOW(), NOW(), 0, ?, 0, ?)",
		req.SubforumID, req.Name, userID, Entities.CharacterSheetTopic, userID)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to insert topic: " + err.Error()})
		c.Abort()
		return
	}
	topicID, err := res.LastInsertId()
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get topic ID"})
		c.Abort()
		return
	}

	character := Entities.Character{
		UserId:  userID,
		TopicId: int(topicID),
		Name:    req.Name,
		Avatar:  req.Avatar,
		CustomFields: Entities.CustomFieldEntity{
			CustomFields: req.CustomFields,
		},
	}

	createdEntity, characterID, err := Services.CreateEntity("character", &character, tx)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to create character: " + err.Error()})
		c.Abort()
		return
	}

	// Handle factions
	for _, faction := range req.FactionIDs {
		var factionID int

		// If faction ID is negative, create a new faction
		if faction.Id < 0 {
			newFactionID, err := Services.CreateFaction(faction, tx)
			if err != nil {
				_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to create faction: " + err.Error()})
				c.Abort()
				return
			}
			factionID = int(newFactionID)
		} else {
			factionID = faction.Id
		}

		// Add faction to character
		if err := Services.AddFactionCharacter(factionID, int(characterID), tx); err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to add faction to character: " + err.Error()})
			c.Abort()
			return
		}
	}

	if err := tx.Commit(); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to commit transaction"})
		c.Abort()
		return
	}

	// Emit CharacterCreated event
	Events.Publish(db, Events.CharacterCreated, Events.CharacterCreatedEvent{
		CharacterID: characterID,
		SubforumID:  req.SubforumID,
	})

	c.JSON(http.StatusCreated, createdEntity)
}

func PatchCharacter(c *gin.Context, db *sql.DB) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid Id"})
		c.Abort()
		return
	}

	var jsonMap map[string]interface{}
	if err := c.ShouldBindJSON(&jsonMap); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid request body: " + err.Error()})
		c.Abort()
		return
	}

	updatedEntity, err := Services.PatchEntity(int64(id), "character", jsonMap, db)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to patch character: " + err.Error()})
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, updatedEntity)
}

func GetCharacterList(c *gin.Context, db *sql.DB) {
	// 1. Get the faction tree
	factions, err := Services.GetFactionTree(db)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get faction tree: " + err.Error()})
		c.Abort()
		return
	}

	// 2. Create a map for easy access to factions by ID
	factionMap := make(map[int]*Entities.Faction)
	for i := range factions {
		factions[i].Characters = []Entities.Character{}
		factionMap[factions[i].Id] = &factions[i]
	}

	// 3. Fetch active characters and their factions
	query := `
		WITH RankedFactions AS (
			SELECT
				c.id,
				c.name,
				c.avatar,
				f.id as faction_id,
				ROW_NUMBER() OVER(PARTITION BY c.id ORDER BY f.level DESC) as rn
			FROM
				character_base c
			JOIN
				character_faction cf ON c.id = cf.character_id
			JOIN
				factions f ON cf.faction_id = f.id
			WHERE
				c.character_status = 0
		)
		SELECT id, name, avatar, faction_id FROM RankedFactions WHERE rn = 1
	`
	rows, err := db.Query(query)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get characters: " + err.Error()})
		c.Abort()
		return
	}
	defer rows.Close()

	// 4. Assign characters to their determined faction
	for rows.Next() {
		var char Entities.Character
		var factionID int
		if err := rows.Scan(&char.Id, &char.Name, &char.Avatar, &factionID); err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to scan character: " + err.Error()})
			c.Abort()
			return
		}
		if faction, ok := factionMap[factionID]; ok {
			faction.Characters = append(faction.Characters, char)
		}
	}

	c.JSON(http.StatusOK, factions)
}

func GetCharacterAutocomplete(c *gin.Context, db *sql.DB) {
	query := `
		SELECT id, name FROM character_base WHERE name LIKE ? ORDER BY name ASC LIMIT 10
	`
	rows, err := db.Query(query, "%"+c.Param("term")+"%")
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get characters: " + err.Error()})
		c.Abort()
		return
	}
	defer rows.Close()
	var characters []Entities.ShortCharacter
	for rows.Next() {
		var tempCharacter Entities.ShortCharacter
		if err := rows.Scan(&tempCharacter.Id, &tempCharacter.Name); err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to scan character: " + err.Error()})
		}
		characters = append(characters, tempCharacter)
	}
	c.JSON(http.StatusOK, characters)
}

func GetUserCharacters(c *gin.Context, db *sql.DB) {
	userID := Services.GetUserIdFromContext(c)
	if userID == 0 {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusUnauthorized, Message: "Unauthorized"})
		c.Abort()
		return
	}

	query := "SELECT id, name FROM character_base WHERE user_id = ?"
	rows, err := db.Query(query, userID)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get user characters: " + err.Error()})
		c.Abort()
		return
	}
	defer rows.Close()

	var characters []Entities.ShortCharacter
	for rows.Next() {
		var char Entities.ShortCharacter
		if err := rows.Scan(&char.Id, &char.Name); err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to scan character: " + err.Error()})
			c.Abort()
			return
		}
		characters = append(characters, char)
	}

	c.JSON(http.StatusOK, characters)
}

func GetCharacterProfilesByUser(c *gin.Context, db *sql.DB) {
	userID := Services.GetUserIdFromContext(c)
	if userID == 0 {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusUnauthorized, Message: "Unauthorized"})
		c.Abort()
		return
	}

	// Get IDs of profiles for this user
	query := "SELECT cp.id, cp.character_id, cb.name, cp.avatar FROM character_profile_base cp JOIN character_base cb ON cp.character_id = cb.id WHERE cb.user_id = ?"
	rows, err := db.Query(query, userID)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get character profiles: " + err.Error()})
		c.Abort()
		return
	}
	defer rows.Close()

	var profiles []Entities.CharacterProfile
	for rows.Next() {
		var id int
		var characterId int
		var name string
		var avatar *string
		if err := rows.Scan(&id, &characterId, &name, &avatar); err != nil {
			continue
		}

		entity, err := Services.GetEntity(int64(id), "character_profile", db)
		if err != nil {
			continue
		}

		if profile, ok := entity.(*Entities.CharacterProfile); ok {
			profile.CharacterId = characterId
			profile.CharacterName = name
			profile.Avatar = avatar
			profiles = append(profiles, *profile)
		}
	}

	c.JSON(http.StatusOK, profiles)
}

func GetCharacterProfile(c *gin.Context, db *sql.DB) {
	characterID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid character ID"})
		c.Abort()
		return
	}

	// Find profile ID for this character
	var profileID int
	err = db.QueryRow("SELECT id FROM character_profile_base WHERE character_id = ?", characterID).Scan(&profileID)
	if err != nil {
		if err == sql.ErrNoRows {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusNotFound, Message: "Character profile not found"})
		} else {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get character profile ID: " + err.Error()})
		}
		c.Abort()
		return
	}

	entity, err := Services.GetEntity(int64(profileID), "character_profile", db)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get character profile: " + err.Error()})
		c.Abort()
		return
	}

	if profile, ok := entity.(*Entities.CharacterProfile); ok {
		// Fetch character name from character_base and avatar from character_profile_base
		err := db.QueryRow("SELECT cb.name, cpb.avatar FROM character_base cb JOIN character_profile_base cpb ON cb.id = cpb.character_id WHERE cpb.id = ?", profile.Id).Scan(&profile.CharacterName, &profile.Avatar)
		if err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get character details: " + err.Error()})
			c.Abort()
			return
		}
		c.JSON(http.StatusOK, profile)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func CharacterProfileUpdate(c *gin.Context, db *sql.DB) {
	characterID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid character ID"})
		c.Abort()
		return
	}

	// Find profile ID for this character
	var profileID int
	err = db.QueryRow("SELECT id FROM character_profile_base WHERE character_id = ?", characterID).Scan(&profileID)
	if err != nil {
		if err == sql.ErrNoRows {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusNotFound, Message: "Character profile not found"})
		} else {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get character profile ID: " + err.Error()})
		}
		c.Abort()
		return
	}

	var jsonMap map[string]interface{}
	if err := c.ShouldBindJSON(&jsonMap); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid request body: " + err.Error()})
		c.Abort()
		return
	}

	updatedEntity, err := Services.PatchEntity(int64(profileID), "character_profile", jsonMap, db)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to update character profile: " + err.Error()})
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, updatedEntity)
}
