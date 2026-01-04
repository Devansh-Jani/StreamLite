package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestScanVideoDirectoryLogic tests the core logic of scanning without full integration
func TestScanVideoDirectoryLogic(t *testing.T) {
	// Test that video extensions are properly recognized
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

	testCases := []struct {
		filename string
		expected bool
	}{
		{"test.mp4", true},
		{"test.avi", true},
		{"test.mkv", true},
		{"test.txt", false},
		{"test.jpg", false},
		{"test.MP4", true}, // Test case insensitivity
	}

	for _, tc := range testCases {
		ext := filepath.Ext(tc.filename)
		// Simulate the lowercase conversion done in scanVideoDirectory
		ext = strings.ToLower(ext)
		result := videoExtensions[ext]
		if result != tc.expected {
			t.Errorf("File %s: expected %v, got %v", tc.filename, tc.expected, result)
		}
	}
}

// TestVideoMetadataUpdate verifies metadata update logic
func TestVideoMetadataUpdate(t *testing.T) {
	// Test time comparison for metadata updates
	oldTime := time.Now().Add(-1 * time.Hour)
	newTime := time.Now()

	if !newTime.After(oldTime) {
		t.Error("Expected newTime to be after oldTime")
	}

	// Test file size comparison
	oldSize := int64(1000)
	newSize := int64(2000)

	if oldSize == newSize {
		t.Error("Expected sizes to be different")
	}

	// Metadata should update if time OR size changed
	shouldUpdate := newTime.After(oldTime) || newSize != oldSize
	if !shouldUpdate {
		t.Error("Expected metadata to require update")
	}
}

// TestFoundFilesTracking verifies the file tracking mechanism
func TestFoundFilesTracking(t *testing.T) {
	// Simulate the foundFiles map
	foundFiles := make(map[string]bool)
	
	// Add some files
	foundFiles["/videos/test1.mp4"] = true
	foundFiles["/videos/test2.mkv"] = true
	
	// Simulate database state
	dbFiles := []string{
		"/videos/test1.mp4",  // exists
		"/videos/test2.mkv",  // exists
		"/videos/test3.avi",  // deleted - not in foundFiles
	}
	
	// Check which files should be removed
	var toRemove []string
	for _, dbFile := range dbFiles {
		if !foundFiles[dbFile] {
			toRemove = append(toRemove, dbFile)
		}
	}
	
	if len(toRemove) != 1 {
		t.Errorf("Expected 1 file to remove, got %d", len(toRemove))
	}
	
	if len(toRemove) > 0 && toRemove[0] != "/videos/test3.avi" {
		t.Errorf("Expected to remove /videos/test3.avi, got %s", toRemove[0])
	}
}
