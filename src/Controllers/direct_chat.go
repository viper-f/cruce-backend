package Controllers

import (
	"cuento-backend/src/Middlewares"
	"cuento-backend/src/Services"
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func GetLastMessages(c *gin.Context, db *sql.DB) {
	userID := Services.GetUserIdFromContext(c)
	if userID == 0 {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusUnauthorized, Message: "Unauthorized"})
		c.Abort()
		return
	}

	chatID, err := strconv.Atoi(c.Param("chatID"))
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid chat ID"})
		c.Abort()
		return
	}

	// Verify participation
	var count int
	err = db.QueryRow(
		"SELECT COUNT(*) FROM direct_chat_users WHERE direct_chat_id = ? AND user_id = ?",
		chatID, userID,
	).Scan(&count)
	if err != nil || count == 0 {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusForbidden, Message: "You are not a participant in this chat"})
		c.Abort()
		return
	}

	n := 10
	if nStr := c.Query("n"); nStr != "" {
		if parsed, err := strconv.Atoi(nStr); err == nil && parsed > 0 {
			n = parsed
		}
	}

	messageIDStr := c.Query("message_id")

	type Message struct {
		Id              int        `json:"id"`
		UserID          int        `json:"user_id"`
		DateSend        time.Time  `json:"date_send"`
		DateReceived    *time.Time `json:"date_received"`
		ContentAuthor   string     `json:"content_author"`
		ContentReceiver string     `json:"content_receiver"`
	}

	scanRows := func(rows *sql.Rows) ([]Message, error) {
		var msgs []Message
		for rows.Next() {
			var msg Message
			if err := rows.Scan(&msg.Id, &msg.UserID, &msg.DateSend, &msg.DateReceived, &msg.ContentAuthor, &msg.ContentReceiver); err != nil {
				return nil, err
			}
			msgs = append(msgs, msg)
		}
		return msgs, nil
	}

	var messages []Message

	if messageIDStr == "" {
		// Return first N messages
		rows, err := db.Query(`
			SELECT id, user_id, date_send, date_received, content_author, content_receiver
			FROM direct_chat_messages
			WHERE chat_id = ?
			ORDER BY date_send ASC
			LIMIT ?
		`, chatID, n)
		if err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get messages: " + err.Error()})
			c.Abort()
			return
		}
		defer rows.Close()
		messages, err = scanRows(rows)
		if err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to scan messages: " + err.Error()})
			c.Abort()
			return
		}
	} else {
		messageID, err := strconv.Atoi(messageIDStr)
		if err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid message ID"})
			c.Abort()
			return
		}

		// N messages before
		beforeRows, err := db.Query(`
			SELECT id, user_id, date_send, date_received, content_author, content_receiver
			FROM direct_chat_messages
			WHERE chat_id = ? AND id < ?
			ORDER BY date_send DESC
			LIMIT ?
		`, chatID, messageID, n)
		if err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get messages before: " + err.Error()})
			c.Abort()
			return
		}
		defer beforeRows.Close()
		before, err := scanRows(beforeRows)
		if err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to scan messages: " + err.Error()})
			c.Abort()
			return
		}
		// Reverse to chronological order
		for i, j := 0, len(before)-1; i < j; i, j = i+1, j-1 {
			before[i], before[j] = before[j], before[i]
		}

		// N messages after
		afterRows, err := db.Query(`
			SELECT id, user_id, date_send, date_received, content_author, content_receiver
			FROM direct_chat_messages
			WHERE chat_id = ? AND id > ?
			ORDER BY date_send ASC
			LIMIT ?
		`, chatID, messageID, n)
		if err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get messages after: " + err.Error()})
			c.Abort()
			return
		}
		defer afterRows.Close()
		after, err := scanRows(afterRows)
		if err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to scan messages: " + err.Error()})
			c.Abort()
			return
		}

		messages = append(before, after...)
	}

	if messages == nil {
		messages = []Message{}
	}

	c.JSON(http.StatusOK, messages)
}

type CreateDirectChatRequest struct {
	RecipientID int `json:"recipient_id" binding:"required"`
}

func CreateDirectChat(c *gin.Context, db *sql.DB) {
	userID := Services.GetUserIdFromContext(c)
	if userID == 0 {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusUnauthorized, Message: "Unauthorized"})
		c.Abort()
		return
	}

	var req CreateDirectChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid request body: " + err.Error()})
		c.Abort()
		return
	}

	// Find existing chat between the two users
	var chatID int
	err := db.QueryRow(`
		SELECT dcu1.direct_chat_id
		FROM direct_chat_users dcu1
		JOIN direct_chat_users dcu2 ON dcu1.direct_chat_id = dcu2.direct_chat_id
		WHERE dcu1.user_id = ? AND dcu2.user_id = ?
		LIMIT 1
	`, userID, req.RecipientID).Scan(&chatID)

	if err == sql.ErrNoRows {
		res, err := db.Exec(
			"INSERT INTO direct_chats (start_date, status) VALUES (?, 0)",
			time.Now(),
		)
		if err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to create chat: " + err.Error()})
			c.Abort()
			return
		}
		id, err := res.LastInsertId()
		if err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get chat ID: " + err.Error()})
			c.Abort()
			return
		}
		chatID = int(id)

		_, err = db.Exec(
			"INSERT INTO direct_chat_users (direct_chat_id, user_id) VALUES (?, ?), (?, ?)",
			chatID, userID, chatID, req.RecipientID,
		)
		if err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to add chat participants: " + err.Error()})
			c.Abort()
			return
		}
	} else if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to find chat: " + err.Error()})
		c.Abort()
		return
	}

	c.JSON(http.StatusCreated, gin.H{"chat_id": chatID})
}

type DirectChatParticipant struct {
	Id        int     `json:"id"`
	Username  string  `json:"username"`
	Avatar    *string `json:"avatar"`
	PublicKey *string `json:"public_key"`
}

type DirectChatLastReadMessage struct {
	Id              int        `json:"id"`
	UserID          int        `json:"user_id"`
	DateSend        time.Time  `json:"date_send"`
	DateReceived    *time.Time `json:"date_received"`
	ContentAuthor   string     `json:"content_author"`
	ContentReceiver string     `json:"content_receiver"`
}

type DirectChatResponse struct {
	ChatID          int                        `json:"chat_id"`
	Participant     DirectChatParticipant      `json:"participant"`
	LastReadMessage *DirectChatLastReadMessage `json:"last_read_message"`
}

func GetDirectChat(c *gin.Context, db *sql.DB) {
	userID := Services.GetUserIdFromContext(c)
	if userID == 0 {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusUnauthorized, Message: "Unauthorized"})
		c.Abort()
		return
	}

	chatID, err := strconv.Atoi(c.Param("chatID"))
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid chat ID"})
		c.Abort()
		return
	}

	// Get the other participant's info and public key, while verifying current user is in the chat
	var participant DirectChatParticipant
	err = db.QueryRow(`
		SELECT u.id, u.username, u.avatar, pk.public_key
		FROM direct_chat_users dcu
		JOIN direct_chat_users dcu_self ON dcu_self.direct_chat_id = dcu.direct_chat_id AND dcu_self.user_id = ?
		JOIN users u ON dcu.user_id = u.id
		LEFT JOIN public_keys pk ON pk.user_id = u.id
		WHERE dcu.direct_chat_id = ? AND dcu.user_id != ?
	`, userID, chatID, userID).Scan(&participant.Id, &participant.Username, &participant.Avatar, &participant.PublicKey)
	if err != nil {
		if err == sql.ErrNoRows {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusNotFound, Message: "Chat not found or you are not a participant"})
		} else {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get chat participant: " + err.Error()})
		}
		c.Abort()
		return
	}

	// Get current user's last read message
	var lastReadMessageID *int
	err = db.QueryRow(
		"SELECT last_read_message_id FROM direct_chat_users WHERE direct_chat_id = ? AND user_id = ?",
		chatID, userID,
	).Scan(&lastReadMessageID)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get last read message: " + err.Error()})
		c.Abort()
		return
	}

	var lastReadMessage *DirectChatLastReadMessage
	if lastReadMessageID != nil {
		var msg DirectChatLastReadMessage
		err = db.QueryRow(`
			SELECT id, user_id, date_send, date_received, content_author, content_receiver
			FROM direct_chat_messages WHERE id = ?
		`, *lastReadMessageID).Scan(&msg.Id, &msg.UserID, &msg.DateSend, &msg.DateReceived, &msg.ContentAuthor, &msg.ContentReceiver)
		if err == nil {
			lastReadMessage = &msg
		}
	}

	c.JSON(http.StatusOK, DirectChatResponse{
		ChatID:          chatID,
		Participant:     participant,
		LastReadMessage: lastReadMessage,
	})
}

type CreateDirectChatMessageRequest struct {
	ChatID          int    `json:"chat_id" binding:"required"`
	ContentAuthor   string `json:"content_author" binding:"required"`
	ContentReceiver string `json:"content_receiver" binding:"required"`
}

func CreateDirectChatMessage(c *gin.Context, db *sql.DB) {
	userID := Services.GetUserIdFromContext(c)
	if userID == 0 {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusUnauthorized, Message: "Unauthorized"})
		c.Abort()
		return
	}

	var req CreateDirectChatMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "Invalid request body: " + err.Error()})
		c.Abort()
		return
	}

	// Verify the user is a participant in this chat
	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM direct_chat_users WHERE direct_chat_id = ? AND user_id = ?",
		req.ChatID, userID,
	).Scan(&count)
	if err != nil || count == 0 {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusForbidden, Message: "You are not a participant in this chat"})
		c.Abort()
		return
	}

	res, err := db.Exec(
		"INSERT INTO direct_chat_messages (chat_id, user_id, date_send, content_author, content_receiver) VALUES (?, ?, ?, ?, ?)",
		req.ChatID, userID, time.Now(), req.ContentAuthor, req.ContentReceiver,
	)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to create message: " + err.Error()})
		c.Abort()
		return
	}

	messageID, err := res.LastInsertId()
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get message ID: " + err.Error()})
		c.Abort()
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message_id": messageID})
}

type DirectChatListItem struct {
	ChatID      int    `json:"chat_id"`
	UserId      int    `json:"user_id"`
	Username    string `json:"username"`
	UnreadCount int    `json:"unread_count"`
}

func GetDirectChatList(c *gin.Context, db *sql.DB) {
	userID := Services.GetUserIdFromContext(c)
	if userID == 0 {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusUnauthorized, Message: "Unauthorized"})
		c.Abort()
		return
	}

	rows, err := db.Query(`
		SELECT dcu.direct_chat_id, u.id, u.username, dcu.unread_count
		FROM direct_chat_users dcu
		JOIN direct_chat_users dcu_other ON dcu_other.direct_chat_id = dcu.direct_chat_id AND dcu_other.user_id != dcu.user_id
		JOIN users u ON u.id = dcu_other.user_id
		WHERE dcu.user_id = ?
	`, userID)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to get chat list: " + err.Error()})
		c.Abort()
		return
	}
	defer rows.Close()

	chats := []DirectChatListItem{}
	for rows.Next() {
		var item DirectChatListItem
		if err := rows.Scan(&item.ChatID, &item.UserId, &item.Username, &item.UnreadCount); err != nil {
			continue
		}
		chats = append(chats, item)
	}

	c.JSON(http.StatusOK, chats)
}
