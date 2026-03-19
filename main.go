package main

import (
	"cuento-backend/src/Controllers"
	"cuento-backend/src/EventHandlers"
	"cuento-backend/src/Install"
	"cuento-backend/src/Middlewares"
	"cuento-backend/src/Router"
	"cuento-backend/src/Services"
	"cuento-backend/src/Websockets"
	"fmt"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	Services.InitDB()
	EventHandlers.RegisterEventHandlers(Services.DB)

	// Start WebSocket Hub
	go Websockets.MainHub.Run()

	r := gin.Default()
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	r.Use(cors.New(config))

	// Apply error middleware globally
	r.Use(Middlewares.ErrorMiddleware())

	// Public routes
	publicRouter := Router.NewCustomRouter(r.Group("/"))

	// User routes (Public)
	publicRouter.POST("/register", "Register a new user account", func(c *gin.Context) {
		Controllers.Register(c, Services.DB)
	})
	publicRouter.POST("/login", "Login with user credentials", func(c *gin.Context) {
		Controllers.Login(c, Services.DB)
	})
	publicRouter.POST("/refresh", "Refresh access token", func(c *gin.Context) {
		Controllers.RefreshToken(c, Services.DB)
	})
	publicRouter.GET("/board/info", "Get board information", func(c *gin.Context) {
		Controllers.GetBoard(c, Services.DB)
	})
	publicRouter.GET("/user/profile/:userID", "Get user profile details", func(c *gin.Context) {
		Controllers.GetUserProfile(c, Services.DB)
	})
	publicRouter.GET("/user/list", "Get list of active users and their characters", func(c *gin.Context) {
		Controllers.GetUserList(c, Services.DB)
	})
	publicRouter.GET("/user/autocomplete/:term", "Get users matching search term", func(c *gin.Context) {
		Controllers.UserAutocomplete(c, Services.DB)
	})
	publicRouter.GET("/character/get/:id", "Get character details by ID", func(c *gin.Context) {
		Controllers.GetCharacter(c, Services.DB)
	})
	publicRouter.GET("/character-profile/get/:id", "Get character profile details by ID", func(c *gin.Context) {
		Controllers.GetCharacterProfile(c, Services.DB)
	})

	// Optional Auth routes (Context populated if token present, otherwise Guest)
	optionalAuthGroup := r.Group("/")
	optionalAuthGroup.Use(Middlewares.OptionalAuthMiddleware())
	optionalAuthRouter := Router.NewCustomRouter(optionalAuthGroup)
	optionalAuthRouter.GET("/categories/home", "Get home page categories", func(c *gin.Context) {
		Controllers.GetHomeCategories(c, Services.DB)
	})
	optionalAuthRouter.GET("/ping", "Health check endpoint", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	optionalAuthRouter.GET("/install", "Install default database tables", func(c *gin.Context) {
		err := Install.ExecuteSQLFile(Services.DB, "./src/Install/default_tables.sql")
		if err != nil {
			fmt.Println(err.Error())
			return
		}
	})
	optionalAuthRouter.GET("/viewforum/:subforum/:page", "Get topics in a subforum by page", func(c *gin.Context) {
		Controllers.GetTopicsBySubforum(c, Services.DB)
	})
	optionalAuthRouter.GET("/topic/get/:id", "Get topic details by ID", func(c *gin.Context) {
		Controllers.GetTopic(c, Services.DB)
	})
	optionalAuthRouter.GET("/character-list", "Get list of all characters", func(c *gin.Context) {
		Controllers.GetCharacterList(c, Services.DB)
	})
	optionalAuthRouter.GET("/subforum/list-short", "Get list of all subforums", func(c *gin.Context) {
		Controllers.GetShortSubforumList(c, Services.DB)
	})
	optionalAuthRouter.GET("/character-autocomplete/:term", "Get list of characters matching search term", func(c *gin.Context) {
		Controllers.GetCharacterAutocomplete(c, Services.DB)
	})
	optionalAuthRouter.GET("/mask-autocomplete/:term", "Get list of masks matching search term", func(c *gin.Context) {
		Controllers.GetMaskAutocomplete(c, Services.DB)
	})
	optionalAuthRouter.GET("/factions/get", "Get faction tree", func(c *gin.Context) {
		Controllers.GetFactionTree(c, Services.DB)
	})
	optionalAuthRouter.POST("/episodes/get", "Get episode list", func(c *gin.Context) {
		Controllers.GetEpisodes(c, Services.DB)
	})
	optionalAuthRouter.GET("/subforum/get/:id", "Get subforum details by ID", func(c *gin.Context) {
		Controllers.GetSubforum(c, Services.DB)
	})
	optionalAuthRouter.GET("/topic-posts/:id", "Get posts in a topic by page", func(c *gin.Context) {
		Controllers.GetPostsByTopic(c, Services.DB)
	})
	optionalAuthRouter.GET("/users/page/:page_type/:page_id", "Get users currently viewing a page", func(c *gin.Context) {
		Controllers.GetUsersByPage(c, Services.DB)
	})
	optionalAuthRouter.GET("/user/character-profiles-topic/:topicID", "Get current user's character profiles for a topic", func(c *gin.Context) {
		Controllers.GetCharacterProfilesByUserAndTopic(c, Services.DB)
	})
	optionalAuthRouter.POST("/post/create", "Create a new post in a topic", func(c *gin.Context) {
		Controllers.CreatePost(c, Services.DB)
	})
	optionalAuthRouter.GET("/active-topics", "Get list of active topics", func(c *gin.Context) {
		Controllers.GetActiveTopics(c, Services.DB)
	})
	optionalAuthRouter.GET("/active-topic-count", "Get count of active topics", func(c *gin.Context) {
		Controllers.GetActiveTopicCount(c, Services.DB)
	})
	optionalAuthRouter.GET("/mask/:id", "Get mask by ID", func(c *gin.Context) {
		Controllers.GetMask(c, Services.DB)
	})
	optionalAuthRouter.GET("/user-masks/:userID", "Get user's masks", func(c *gin.Context) {
		Controllers.GetUserMasks(c, Services.DB)
	})

	// Protected routes
	protectedGroup := r.Group("/")
	protectedGroup.Use(Middlewares.AuthMiddleware())
	protectedGroup.Use(Middlewares.PermissionsMiddleware(Services.DB))
	protectedRouter := Router.NewCustomRouter(protectedGroup)

	protectedRouter.POST("/character/create", "Create a new character", func(c *gin.Context) {
		Controllers.CreateCharacter(c, Services.DB)
	})
	protectedRouter.POST("/character/update/:id", "Update character by ID", func(c *gin.Context) {
		Controllers.UpdateCharacter(c, Services.DB)
	})
	protectedRouter.GET("/user/characters", "Get current user's characters", func(c *gin.Context) {
		Controllers.GetUserCharacters(c, Services.DB)
	})
	protectedRouter.GET("/user/character-profiles", "Get current user's character profiles", func(c *gin.Context) {
		Controllers.GetCharacterProfilesByUser(c, Services.DB)
	})
	protectedRouter.GET("/faction-children/:parent_id/get", "Get child factions by parent ID", func(c *gin.Context) {
		Controllers.GetFactionChildren(c, Services.DB)
	})

	// Character Template routes
	protectedRouter.GET("/template/:type/get", "Get character template by type", func(c *gin.Context) {
		Controllers.GetTemplate(c, Services.DB)
	})
	protectedRouter.POST("/template/:type/update", "Update character template by type", func(c *gin.Context) {
		Controllers.UpdateTemplate(c, Services.DB)
	})
	protectedRouter.POST("/episode/create", "Create a new episode", func(c *gin.Context) {
		Controllers.CreateEpisode(c, Services.DB)
	})
	protectedRouter.POST("/episode/update/:id", "Update episode by ID", func(c *gin.Context) {
		Controllers.UpdateEpisode(c, Services.DB)
	})
	protectedRouter.GET("/permission-matrix/get", "Get permission matrix", func(c *gin.Context) {
		Controllers.GetPermissionMatrix(c, Services.DB)
	})
	protectedRouter.POST("/permission-matrix/update", "Update permission matrix", func(c *gin.Context) {
		Controllers.UpdatePermissionMatrix(c, Services.DB)
	})
	protectedRouter.POST("/post/update/:id", "Update post by ID", func(c *gin.Context) {
		Controllers.UpdatePost(c, Services.DB)
	})
	protectedRouter.POST("/character-profile/update/:id", "Update character profile by ID", func(c *gin.Context) {
		Controllers.CharacterProfileUpdate(c, Services.DB)
	})
	protectedRouter.POST("/topic/create", "Create topic", func(c *gin.Context) {
		Controllers.CreateTopic(c, Services.DB)
	})
	protectedRouter.POST("/topic/update/:id", "Update topic by ID", func(c *gin.Context) {
		Controllers.UpdateTopic(c, Services.DB)
	})
	protectedRouter.GET("/notifications/unread", "Get unread notifications for the current user", func(c *gin.Context) {
		Controllers.GetUnreadNotifications(c, Services.DB)
	})
	protectedRouter.POST("/notifications/dismiss/:id", "Mark a notification as read", func(c *gin.Context) {
		Controllers.DismissNotification(c, Services.DB)
	})
	protectedRouter.POST("/character/accept/:id", "Accept a character", func(c *gin.Context) {
		Controllers.AcceptCharacter(c, Services.DB)
	})
	protectedRouter.POST("/user/settings/update", "Update user settings", func(c *gin.Context) {
		Controllers.UpdateSettings(c, Services.DB)
	})
	protectedRouter.POST("/mask/create", "Create a new mask", func(c *gin.Context) {
		Controllers.CreateMask(c, Services.DB)
	})
	protectedRouter.POST("/mask/update/:id", "Update mask by ID", func(c *gin.Context) {
		Controllers.UpdateMask(c, Services.DB)
	})
	protectedRouter.POST("/user/save-keys", "Save user's public and private keys", func(c *gin.Context) {
		Controllers.SaveKeys(c, Services.DB)
	})
	protectedRouter.GET("/user/private-key", "Get current user's active private key", func(c *gin.Context) {
		Controllers.GetPrivateKey(c, Services.DB)
	})
	protectedRouter.GET("/user/public-key/:userID", "Get public key by user ID", func(c *gin.Context) {
		Controllers.GetPublicKeyByUserId(c, Services.DB)
	})
	protectedRouter.POST("/direct-chat/create", "Create or find a direct chat with a user", func(c *gin.Context) {
		Controllers.CreateDirectChat(c, Services.DB)
	})
	protectedRouter.POST("/direct-chat/message/create", "Send a direct chat message", func(c *gin.Context) {
		Controllers.CreateDirectChatMessage(c, Services.DB)
	})
	protectedRouter.GET("/direct-chat/:chatID", "Get direct chat details", func(c *gin.Context) {
		Controllers.GetDirectChat(c, Services.DB)
	})
	protectedRouter.GET("/direct-chat/:chatID/messages", "Get messages in a direct chat", func(c *gin.Context) {
		Controllers.GetLastMessages(c, Services.DB)
	})
	protectedRouter.GET("/direct-chat/:chatID/messages/:messageID/before", "Get messages before a given message", func(c *gin.Context) {
		Controllers.GetMessagesBefore(c, Services.DB)
	})
	protectedRouter.GET("/direct-chat/:chatID/messages/:messageID/after", "Get messages after a given message", func(c *gin.Context) {
		Controllers.GetMessagesAfter(c, Services.DB)
	})
	protectedRouter.GET("/direct-chats", "Get list of current user's direct chats", func(c *gin.Context) {
		Controllers.GetDirectChatList(c, Services.DB)
	})

	// WebSocket route with special authentication
	wsGroup := r.Group("/")
	wsGroup.Use(Middlewares.WebSocketAuthMiddleware())
	wsRouter := Router.NewCustomRouter(wsGroup)
	wsRouter.GET("/ws", "WebSocket connection endpoint", func(c *gin.Context) {
		Controllers.HandleWebSocket(c, Services.DB)
	})

	r.Run() // listen and serve on 0.0.0.0:8080
}
