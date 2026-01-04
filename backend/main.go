package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
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

// Playlist represents a group of related videos
type Playlist struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	VideoIDs    []int   `json:"video_ids"`
	VideoCount  int     `json:"video_count"`
	ThumbnailID int     `json:"thumbnail_id"`
	Directory   string  `json:"directory"`
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

const (
	// thumbnailPlaceholderSVG is the default placeholder thumbnail for videos
	thumbnailPlaceholderSVG = `<svg width="320" height="180" xmlns="http://www.w3.org/2000/svg">
		<rect width="320" height="180" fill="#1a1a1a"/>
		<circle cx="160" cy="90" r="30" fill="#404040"/>
		<polygon points="150,75 150,105 175,90" fill="#ffffff"/>
	</svg>`
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
	api.HandleFunc("/videos/refresh", refreshVideos).Methods("POST")
	api.HandleFunc("/videos/{id}", getVideo).Methods("GET")
	api.HandleFunc("/videos/{id}/stream", streamVideo).Methods("GET")
	api.HandleFunc("/videos/{id}/thumbnail", getThumbnail).Methods("GET")
	api.HandleFunc("/videos/{id}/view", incrementView).Methods("POST")
	api.HandleFunc("/videos/{id}/like", toggleLike).Methods("POST")
	api.HandleFunc("/videos/{id}/comments", getComments).Methods("GET")
	api.HandleFunc("/videos/{id}/comments", addComment).Methods("POST")
	api.HandleFunc("/playlists", getPlaylists).Methods("GET")
	api.HandleFunc("/playlists/{id}", getPlaylist).Methods("GET")

	// Setup CORS
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "*"
	}

	var c *cors.Cors
	if allowedOrigins == "*" {
		// Use AllowAll for wildcard
		c = cors.AllowAll()
	} else {
		// Use specific origins
		c = cors.New(cors.Options{
			AllowedOrigins:   strings.Split(allowedOrigins, ","),
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"*"},
			AllowCredentials: true,
		})
	}

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

	// Verify video directory exists and is accessible
	if _, err := os.Stat(config.VideoDir); err != nil {
		if os.IsNotExist(err) {
			logger.Printf("Video directory does not exist: %s", config.VideoDir)
			return fmt.Errorf("video directory does not exist: %s", config.VideoDir)
		} else if os.IsPermission(err) {
			logger.Printf("Permission denied accessing video directory: %s", config.VideoDir)
			return fmt.Errorf("permission denied accessing video directory: %s", config.VideoDir)
		}
		logger.Printf("Error accessing video directory %s: %v", config.VideoDir, err)
		return fmt.Errorf("error accessing video directory: %w", err)
	}

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

	// Track found files to detect removals
	foundFiles := make(map[string]bool)
	addedCount := 0
	updatedCount := 0

	// Track visited directories to avoid infinite loops with circular symlinks
	visitedDirs := make(map[string]bool)

	err := walkWithSymlinks(config.VideoDir, visitedDirs, func(path string, info os.FileInfo, err error) error {
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

		// Normalize the path for cross-platform compatibility
		path = filepath.Clean(path)

		// Verify file is readable before adding to database
		if _, err := os.Stat(path); err != nil {
			logger.Printf("Warning: Cannot access video file %s: %v", path, err)
			return nil
		}

		// Mark this file as found
		foundFiles[path] = true

		// Check if video already exists in database
		var existingID int
		var existingModTime time.Time
		var existingFileSize int64
		err = db.QueryRow("SELECT id, modified_at, file_size FROM videos WHERE filepath = $1", path).Scan(&existingID, &existingModTime, &existingFileSize)

		if err == sql.ErrNoRows {
			// New video - insert it
			filename := info.Name()
			title := strings.TrimSuffix(filename, ext)
			title = strings.ReplaceAll(title, "_", " ")
			title = strings.ReplaceAll(title, "-", " ")

			_, err = db.Exec(`
				INSERT INTO videos (filename, filepath, title, file_size, modified_at)
				VALUES ($1, $2, $3, $4, $5)
			`, filename, path, title, info.Size(), info.ModTime())

			if err != nil {
				logger.Printf("Error inserting video %s: %v", filename, err)
				return nil
			}

			addedCount++
			logger.Printf("Added new video: %s", filename)
		} else if err != nil {
			logger.Printf("Error checking video existence: %v", err)
			return nil
		} else {
			// Video exists - check if metadata needs updating
			if info.ModTime().After(existingModTime) || info.Size() != existingFileSize {
				_, err = db.Exec(`
					UPDATE videos 
					SET file_size = $1, modified_at = $2
					WHERE id = $3
				`, info.Size(), info.ModTime(), existingID)

				if err != nil {
					logger.Printf("Error updating video metadata: %v", err)
				} else {
					updatedCount++
					logger.Printf("Updated metadata for video ID %d", existingID)
				}
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Remove videos from database that no longer exist in filesystem
	// Only perform cleanup if we found at least some files (avoid cleanup on scan errors)
	if len(foundFiles) > 0 {
		rows, err := db.Query("SELECT id, filepath, filename FROM videos")
		if err != nil {
			logger.Printf("Error querying videos for cleanup: %v", err)
		} else {
			defer rows.Close()
			removedCount := 0

			for rows.Next() {
				var id int
				var filepath, filename string
				if err := rows.Scan(&id, &filepath, &filename); err != nil {
					logger.Printf("Error scanning video row: %v", err)
					continue
				}

				if !foundFiles[filepath] {
					// File no longer exists - remove from database
					_, err = db.Exec("DELETE FROM videos WHERE id = $1", id)
					if err != nil {
						logger.Printf("Error removing video %s: %v", filename, err)
					} else {
						removedCount++
						logger.Printf("Removed deleted video: %s", filename)
					}
				}
			}

			logger.Printf("Scan complete: %d added, %d updated, %d removed", addedCount, updatedCount, removedCount)
		}
	} else {
		logger.Printf("Scan complete: %d added, %d updated (no cleanup performed - no files found)", addedCount, updatedCount)
	}

	return nil
}

// walkWithSymlinks walks the file tree following symbolic links
func walkWithSymlinks(root string, visitedDirs map[string]bool, walkFn filepath.WalkFunc) error {
	// Get absolute path to handle symlinks properly
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}

	// Evaluate symlinks to get the real path
	realRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		// If we can't resolve the symlink, log and continue with the original path
		if logger != nil {
			logger.Printf("Warning: Cannot resolve path %s: %v", absRoot, err)
		}
		realRoot = absRoot
	}

	// Check if we've already visited this directory to avoid infinite loops
	if visitedDirs[realRoot] {
		return nil
	}
	visitedDirs[realRoot] = true

	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return walkFn(path, info, err)
		}

		// If this is a symlink, follow it
		if info.Mode()&os.ModeSymlink != 0 {
			// Get the target of the symlink
			targetPath, err := filepath.EvalSymlinks(path)
			if err != nil {
				if logger != nil {
					logger.Printf("Warning: Cannot resolve symlink %s: %v", path, err)
				}
				return nil // Skip this symlink but continue walking
			}

			// Get info about the target
			targetInfo, err := os.Stat(targetPath)
			if err != nil {
				if logger != nil {
					logger.Printf("Warning: Cannot stat symlink target %s: %v", targetPath, err)
				}
				return nil // Skip this symlink but continue walking
			}

			// If target is a directory, recursively walk it
			if targetInfo.IsDir() {
				if logger != nil {
					logger.Printf("Following symlink directory: %s -> %s", path, targetPath)
				}
				// Walk the symlinked directory but don't return the error,
				// allowing filepath.Walk to continue with siblings
				if err := walkWithSymlinks(targetPath, visitedDirs, walkFn); err != nil {
					if logger != nil {
						logger.Printf("Warning: Error walking symlinked directory %s: %v", targetPath, err)
					}
				}
				return nil // Continue walking siblings
			} else {
				// If target is a file, call walkFn with the original symlink path
				// but use the target's info
				return walkFn(path, targetInfo, nil)
			}
		}

		// For regular files and directories, use the normal walk function
		return walkFn(path, info, err)
	})
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

func refreshVideos(w http.ResponseWriter, r *http.Request) {
	logger.Println("Refreshing video directory scan")

	if err := scanVideoDirectory(); err != nil {
		logger.Printf("Error during video refresh: %v", err)
		http.Error(w, "Failed to refresh videos", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Videos refreshed successfully"})
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

	// Normalize the video path for cross-platform compatibility
	videoPath = filepath.Clean(videoPath)

	// Verify file exists and is accessible
	fileInfo, err := os.Stat(videoPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Printf("Video file not found: %s", videoPath)
			http.Error(w, "Video file not found", http.StatusNotFound)
		} else if os.IsPermission(err) {
			logger.Printf("Permission denied accessing video file: %s", videoPath)
			http.Error(w, "Permission denied accessing video file", http.StatusForbidden)
		} else {
			logger.Printf("Error accessing video file %s: %v", videoPath, err)
			http.Error(w, "Failed to access video file", http.StatusInternalServerError)
		}
		return
	}

	// Open video file
	file, err := os.Open(videoPath)
	if err != nil {
		logger.Printf("Error opening video file %s: %v", videoPath, err)
		http.Error(w, "Failed to open video file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Get file info (already obtained from os.Stat above)
	// fileInfo is used below for content-length and range handling

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

func getThumbnail(w http.ResponseWriter, r *http.Request) {
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

	// Normalize the video path
	videoPath = filepath.Clean(videoPath)

	// Verify file exists
	if _, err := os.Stat(videoPath); err != nil {
		logger.Printf("Video file not found for thumbnail: %s", videoPath)
		servePlaceholderThumbnail(w)
		return
	}

	// Serve placeholder thumbnail
	// In production, you could generate real thumbnails using ffmpeg
	servePlaceholderThumbnail(w)
}

func servePlaceholderThumbnail(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	if _, err := w.Write([]byte(thumbnailPlaceholderSVG)); err != nil {
		logger.Printf("Error writing placeholder thumbnail: %v", err)
	}
}

// normalizePlaylistName removes common variations to group similar videos
func normalizePlaylistName(filename string) string {
	// Remove extension
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	
	// Convert to lowercase for comparison
	name = strings.ToLower(name)
	
	// Remove patterns using simple string operations
	for _, pattern := range []string{
		"_v1", "_v2", "_v3", "_v4", "_v5", "_v6", "_v7", "_v8", "_v9",
		" v1", " v2", " v3", " v4", " v5", " v6", " v7", " v8", " v9",
		"-v1", "-v2", "-v3", "-v4", "-v5", "-v6", "-v7", "-v8", "-v9",
		"_edited", "_final", "_draft",
		" edited", " final", " draft",
		"-edited", "-final", "-draft",
		"(edited)", "(final)", "(draft)",
	} {
		name = strings.TrimSuffix(name, pattern)
	}
	
	// Remove trailing numbers (e.g., _1, _2, -1, -2, etc.)
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] >= '0' && name[i] <= '9' {
			continue
		}
		if name[i] == '_' || name[i] == '-' || name[i] == ' ' {
			name = name[:i]
			break
		}
		break
	}
	
	// Normalize separators
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")
	
	// Trim spaces
	name = strings.TrimSpace(name)
	
	return name
}

// generatePlaylists groups videos by similar names within the same directory
func generatePlaylists() []Playlist {
	rows, err := db.Query(`
		SELECT id, filename, filepath, title
		FROM videos
		ORDER BY filepath
	`)
	if err != nil {
		logger.Printf("Error querying videos for playlists: %v", err)
		return []Playlist{}
	}
	defer rows.Close()

	// Group videos by directory and normalized name
	type videoInfo struct {
		ID       int
		Filename string
		Filepath string
		Title    string
	}
	
	var videos []videoInfo
	for rows.Next() {
		var v videoInfo
		if err := rows.Scan(&v.ID, &v.Filename, &v.Filepath, &v.Title); err != nil {
			logger.Printf("Error scanning video: %v", err)
			continue
		}
		videos = append(videos, v)
	}

	// Group by directory and normalized name
	playlistMap := make(map[string][]videoInfo)
	for _, v := range videos {
		dir := filepath.Dir(v.Filepath)
		normalized := normalizePlaylistName(v.Filename)
		key := dir + "||" + normalized
		playlistMap[key] = append(playlistMap[key], v)
	}

	// Create playlists only for groups with more than one video
	var playlists []Playlist
	for key, vids := range playlistMap {
		if len(vids) < 2 {
			continue
		}

		parts := strings.Split(key, "||")
		dir := parts[0]
		normalized := parts[1]

		// Sort videos alphabetically by filename using built-in sort
		sortedVids := make([]videoInfo, len(vids))
		copy(sortedVids, vids)
		sort.Slice(sortedVids, func(i, j int) bool {
			return sortedVids[i].Filename < sortedVids[j].Filename
		})

		var videoIDs []int
		for _, v := range sortedVids {
			videoIDs = append(videoIDs, v.ID)
		}

		// Use first video's title as playlist name (cleaned up)
		playlistName := normalized
		if len(playlistName) > 0 {
			// Capitalize first letter
			playlistName = strings.ToUpper(string(playlistName[0])) + playlistName[1:]
		}

		// Generate deterministic playlist ID based on directory and normalized name
		hashInput := dir + "||" + normalized
		hash := md5.Sum([]byte(hashInput))
		playlistID := "pl_" + hex.EncodeToString(hash[:])[:12]

		playlist := Playlist{
			ID:          playlistID,
			Name:        playlistName,
			VideoIDs:    videoIDs,
			VideoCount:  len(videoIDs),
			ThumbnailID: videoIDs[0],
			Directory:   dir,
		}
		playlists = append(playlists, playlist)
	}

	return playlists
}

func getPlaylists(w http.ResponseWriter, r *http.Request) {
	playlists := generatePlaylists()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(playlists)
}

func getPlaylist(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	playlistID := vars["id"]

	playlists := generatePlaylists()
	for _, playlist := range playlists {
		if playlist.ID == playlistID {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(playlist)
			return
		}
	}

	http.Error(w, "Playlist not found", http.StatusNotFound)
}
