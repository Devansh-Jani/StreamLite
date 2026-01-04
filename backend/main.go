package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

// Video represents a video file
type Video struct {
	ID           int       `json:"id"`
	Filename     string    `json:"filename"`
	Filepath     string    `json:"filepath"`
	Title        string    `json:"title"`
	Views        int       `json:"views"`
	Likes        int       `json:"likes"`
	Duration     int       `json:"duration"`
	FileSize     int64     `json:"file_size"`
	CreatedAt    time.Time `json:"created_at"`
	ModifiedAt   time.Time `json:"modified_at"`
	ThumbnailURL string    `json:"thumbnail_url"`
}

// Comment represents a comment on a video
type Comment struct {
	ID        int       `json:"id"`
	VideoID   int       `json:"video_id"`
	Author    string    `json:"author"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// Config holds application configuration
type Config struct {
	DatabaseURL string
	VideoDir    string
	ConfigDir   string
	Port        string
}

var (
	db     *sql.DB
	config Config
	logger *log.Logger
)

func main() {
	// Load configuration
	config = Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://streamlite:streamlite@localhost:5432/streamlite?sslmode=disable"),
		VideoDir:    getEnv("VIDEO_DIR", "./videos"),
		ConfigDir:   getEnv("CONFIG_DIR", "./config"),
		Port:        getEnv("PORT", "8082"),
	}

	// Setup logging
	setupLogging()

	// Connect to database
	var err error
	db, err = sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		logger.Fatalf("Failed to ping database: %v", err)
	}
	logger.Println("Connected to database successfully")

	// Scan video directory
	if err := scanVideoDirectory(); err != nil {
		logger.Printf("Warning: Failed to scan video directory: %v", err)
	}

	// Setup router
	router := mux.NewRouter()

	// API routes
	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/videos", getVideos).Methods("GET")
	api.HandleFunc("/videos/{id}", getVideo).Methods("GET")
	api.HandleFunc("/videos/{id}/stream", streamVideo).Methods("GET")
	api.HandleFunc("/videos/{id}/view", incrementView).Methods("POST")
	api.HandleFunc("/videos/{id}/like", toggleLike).Methods("POST")
	api.HandleFunc("/videos/{id}/comments", getComments).Methods("GET")
	api.HandleFunc("/videos/{id}/comments", addComment).Methods("POST")

	// Setup CORS
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:3000,http://localhost:80"
	}

	c := cors.New(cors.Options{
		AllowedOrigins:   strings.Split(allowedOrigins, ","),
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(router)

	// Start server
	logger.Printf("Server starting on port %s", config.Port)
	if err := http.ListenAndServe(":"+config.Port, handler); err != nil {
		logger.Fatalf("Failed to start server: %v", err)
	}
}

func setupLogging() {
	logFile := filepath.Join(config.ConfigDir, "streamlite.log")

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(config.ConfigDir, 0755); err != nil {
		log.Fatalf("Failed to create config directory: %v", err)
	}

	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	logger = log.New(io.MultiWriter(os.Stdout, f), "[StreamLite] ", log.Ldate|log.Ltime|log.Lshortfile)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func scanVideoDirectory() error {
	logger.Printf("Scanning video directory: %s", config.VideoDir)

	videoExtensions := map[string]bool{
		".mp4":  true,
		".avi":  true,
		".mkv":  true,
		".mov":  true,
		".wmv":  true,
		".flv":  true,
		".webm": true,
		".m4v":  true,
	}

	count := 0
	err := filepath.Walk(config.VideoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Printf("Error accessing path %s: %v", path, err)
			return nil // Continue walking
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(info.Name()))
		if !videoExtensions[ext] {
			return nil
		}

		// Check if video already exists in database
		var exists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM videos WHERE filepath = $1)", path).Scan(&exists)
		if err != nil {
			logger.Printf("Error checking video existence: %v", err)
			return nil
		}

		if exists {
			return nil
		}

		// Generate title from filename
		filename := info.Name()
		title := strings.TrimSuffix(filename, ext)
		title = strings.ReplaceAll(title, "_", " ")
		title = strings.ReplaceAll(title, "-", " ")

		// Insert video into database
		_, err = db.Exec(`
			INSERT INTO videos (filename, filepath, title, file_size, modified_at)
			VALUES ($1, $2, $3, $4, $5)
		`, filename, path, title, info.Size(), info.ModTime())

		if err != nil {
			logger.Printf("Error inserting video %s: %v", filename, err)
			return nil
		}

		count++
		return nil
	})

	if err != nil {
		return err
	}

	logger.Printf("Scanned video directory: found %d new videos", count)
	return nil
}

func getVideos(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT id, filename, filepath, title, views, likes, duration, file_size, created_at, modified_at
		FROM videos
		ORDER BY modified_at DESC
	`)
	if err != nil {
		logger.Printf("Error querying videos: %v", err)
		http.Error(w, "Failed to fetch videos", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	videos := []Video{}
	for rows.Next() {
		var v Video
		err := rows.Scan(&v.ID, &v.Filename, &v.Filepath, &v.Title, &v.Views, &v.Likes, &v.Duration, &v.FileSize, &v.CreatedAt, &v.ModifiedAt)
		if err != nil {
			logger.Printf("Error scanning video: %v", err)
			continue
		}
		v.ThumbnailURL = fmt.Sprintf("/api/videos/%d/thumbnail", v.ID)
		videos = append(videos, v)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(videos)
}

func getVideo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var v Video
	err := db.QueryRow(`
		SELECT id, filename, filepath, title, views, likes, duration, file_size, created_at, modified_at
		FROM videos
		WHERE id = $1
	`, id).Scan(&v.ID, &v.Filename, &v.Filepath, &v.Title, &v.Views, &v.Likes, &v.Duration, &v.FileSize, &v.CreatedAt, &v.ModifiedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Video not found", http.StatusNotFound)
		return
	} else if err != nil {
		logger.Printf("Error fetching video: %v", err)
		http.Error(w, "Failed to fetch video", http.StatusInternalServerError)
		return
	}

	v.ThumbnailURL = fmt.Sprintf("/api/videos/%d/thumbnail", v.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func streamVideo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var videoPath string
	err := db.QueryRow("SELECT filepath FROM videos WHERE id = $1", id).Scan(&videoPath)
	if err == sql.ErrNoRows {
		http.Error(w, "Video not found", http.StatusNotFound)
		return
	} else if err != nil {
		logger.Printf("Error fetching video filepath: %v", err)
		http.Error(w, "Failed to fetch video", http.StatusInternalServerError)
		return
	}

	// Open video file
	file, err := os.Open(videoPath)
	if err != nil {
		logger.Printf("Error opening video file: %v", err)
		http.Error(w, "Failed to open video file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		logger.Printf("Error getting file info: %v", err)
		http.Error(w, "Failed to get file info", http.StatusInternalServerError)
		return
	}

	// Determine content type based on file extension
	ext := strings.ToLower(filepath.Ext(videoPath))
	contentType := "video/mp4" // default
	switch ext {
	case ".mp4", ".m4v":
		contentType = "video/mp4"
	case ".webm":
		contentType = "video/webm"
	case ".avi":
		contentType = "video/x-msvideo"
	case ".mkv":
		contentType = "video/x-matroska"
	case ".mov":
		contentType = "video/quicktime"
	case ".wmv":
		contentType = "video/x-ms-wmv"
	case ".flv":
		contentType = "video/x-flv"
	}

	// Set headers for video streaming
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Accept-Ranges", "bytes")

	// Handle range requests for seeking
	rangeHeader := r.Header.Get("Range")
	if rangeHeader != "" {
		// Parse range header
		ranges := strings.Split(strings.TrimPrefix(rangeHeader, "bytes="), "-")
		start, _ := strconv.ParseInt(ranges[0], 10, 64)
		var end int64
		if len(ranges) > 1 && ranges[1] != "" {
			end, _ = strconv.ParseInt(ranges[1], 10, 64)
		} else {
			end = fileInfo.Size() - 1
		}

		// Set partial content headers
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileInfo.Size()))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", end-start+1))
		w.WriteHeader(http.StatusPartialContent)

		// Seek to start position
		file.Seek(start, 0)

		// Copy the requested range
		io.CopyN(w, file, end-start+1)
	} else {
		// Serve entire file
		w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
		io.Copy(w, file)
	}
}

func incrementView(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	_, err := db.Exec("UPDATE videos SET views = views + 1 WHERE id = $1", id)
	if err != nil {
		logger.Printf("Error incrementing view count: %v", err)
		http.Error(w, "Failed to increment view count", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func toggleLike(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var body struct {
		Action string `json:"action"` // "like" or "unlike"
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		body.Action = "like" // Default to like
	}

	var query string
	if body.Action == "unlike" {
		query = "UPDATE videos SET likes = GREATEST(likes - 1, 0) WHERE id = $1"
	} else {
		query = "UPDATE videos SET likes = likes + 1 WHERE id = $1"
	}

	_, err := db.Exec(query, id)
	if err != nil {
		logger.Printf("Error updating like count: %v", err)
		http.Error(w, "Failed to update like count", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func getComments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	rows, err := db.Query(`
		SELECT id, video_id, author, content, created_at
		FROM comments
		WHERE video_id = $1
		ORDER BY created_at DESC
	`, id)
	if err != nil {
		logger.Printf("Error querying comments: %v", err)
		http.Error(w, "Failed to fetch comments", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	comments := []Comment{}
	for rows.Next() {
		var c Comment
		err := rows.Scan(&c.ID, &c.VideoID, &c.Author, &c.Content, &c.CreatedAt)
		if err != nil {
			logger.Printf("Error scanning comment: %v", err)
			continue
		}
		comments = append(comments, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comments)
}

func addComment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var comment Comment
	if err := json.NewDecoder(r.Body).Decode(&comment); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Sanitize and validate input
	comment.Author = strings.TrimSpace(comment.Author)
	comment.Content = strings.TrimSpace(comment.Content)

	// Limit author name length
	if len(comment.Author) > 100 {
		comment.Author = comment.Author[:100]
	}

	// Limit content length
	if len(comment.Content) > 5000 {
		http.Error(w, "Comment content too long (max 5000 characters)", http.StatusBadRequest)
		return
	}

	if comment.Author == "" {
		comment.Author = "Anonymous"
	}

	if comment.Content == "" {
		http.Error(w, "Comment content is required", http.StatusBadRequest)
		return
	}

	err := db.QueryRow(`
		INSERT INTO comments (video_id, author, content)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`, id, comment.Author, comment.Content).Scan(&comment.ID, &comment.CreatedAt)

	if err != nil {
		logger.Printf("Error inserting comment: %v", err)
		http.Error(w, "Failed to add comment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(comment)
}
