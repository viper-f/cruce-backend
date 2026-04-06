package Controllers

import (
	"cuento-backend/src/Events"
	"cuento-backend/src/Middlewares"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type StaticFile struct {
	FileName        string    `json:"file_name"`
	FileCreatedDate time.Time `json:"file_created_date"`
	FileType        string    `json:"file_type"`
}

func GetStaticFileList(c *gin.Context, db *sql.DB) {
	fileType := c.Param("file_type")

	rows, err := db.Query(
		"SELECT file_name, file_created_date FROM static_files WHERE file_type = ? ORDER BY file_created_date DESC LIMIT 3",
		fileType,
	)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to fetch files: " + err.Error()})
		c.Abort()
		return
	}
	defer rows.Close()

	var files []StaticFile
	for rows.Next() {
		var f StaticFile
		if err := rows.Scan(&f.FileName, &f.FileCreatedDate); err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to scan file: " + err.Error()})
			c.Abort()
			return
		}
		files = append(files, f)
	}

	if files == nil {
		files = []StaticFile{}
	}

	c.JSON(http.StatusOK, files)
}

func UploadFile(c *gin.Context, db *sql.DB) {
	fileType := c.PostForm("file_type")
	if fileType == "" {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "file_type field is required"})
		c.Abort()
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusBadRequest, Message: "file field is required"})
		c.Abort()
		return
	}
	defer file.Close()

	_ = header
	fileName := fileType
	publicDir := "./public"

	var existingCreatedDate time.Time
	err = db.QueryRow("SELECT file_created_date FROM static_files WHERE file_name = ?", fileName).Scan(&existingCreatedDate)
	if err != nil && err != sql.ErrNoRows {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "DB error: " + err.Error()})
		c.Abort()
		return
	}

	if err == nil {
		// File with same name exists — rename the old one on disk and in DB
		ext := filepath.Ext(fileName)
		nameWithoutExt := strings.TrimSuffix(fileName, ext)
		dateStr := existingCreatedDate.Format("2006-01-02_15-04-05")
		renamedFileName := fmt.Sprintf("%s_%s%s", nameWithoutExt, dateStr, ext)

		oldPath := filepath.Join(publicDir, fileName)
		newPath := filepath.Join(publicDir, renamedFileName)

		if renameErr := os.Rename(oldPath, newPath); renameErr != nil && !os.IsNotExist(renameErr) {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to rename file: " + renameErr.Error()})
			c.Abort()
			return
		}

		_, err = db.Exec("UPDATE static_files SET file_name = ? WHERE file_name = ?", renamedFileName, fileName)
		if err != nil {
			_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to update file record: " + err.Error()})
			c.Abort()
			return
		}
	}

	if err := os.MkdirAll(publicDir, os.ModePerm); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to create public directory"})
		c.Abort()
		return
	}

	dst, err := os.Create(filepath.Join(publicDir, fileName))
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to create file: " + err.Error()})
		c.Abort()
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to write file: " + err.Error()})
		c.Abort()
		return
	}

	now := time.Now()
	_, err = db.Exec(
		"INSERT INTO static_files (file_name, file_created_date, file_type) VALUES (?, ?, ?)",
		fileName, now, fileType,
	)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to save file record: " + err.Error()})
		c.Abort()
		return
	}

	rows, err := db.Query(
		"SELECT file_name, file_created_date, file_type FROM static_files WHERE file_type = ? ORDER BY file_created_date DESC LIMIT 3",
		fileType,
	)
	if err != nil {
		_ = c.Error(&Middlewares.AppError{Code: http.StatusInternalServerError, Message: "Failed to fetch files: " + err.Error()})
		c.Abort()
		return
	}
	defer rows.Close()

	var files []StaticFile
	for rows.Next() {
		var f StaticFile
		if scanErr := rows.Scan(&f.FileName, &f.FileCreatedDate, &f.FileType); scanErr != nil {
			continue
		}
		files = append(files, f)
	}

	Events.Publish(db, Events.StaticFileUploaded, Events.StaticFileUploadedEvent{FileType: fileType})

	c.JSON(http.StatusOK, gin.H{"files": files})
}
