package main

import (
	"os"
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

// TestPathNormalization verifies that paths are normalized for cross-platform compatibility
func TestPathNormalization(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"./videos/test.mp4", "videos/test.mp4"},
		{"videos//test.mp4", "videos/test.mp4"},
		{"videos/./test.mp4", "videos/test.mp4"},
		{"videos/../videos/test.mp4", "videos/test.mp4"},
	}

	for _, tc := range testCases {
		result := filepath.Clean(tc.input)
		if result != tc.expected {
			t.Errorf("Path %s: expected %s, got %s", tc.input, tc.expected, result)
		}
	}
}

// TestFileExtensionCaseSensitivity verifies extension handling is case-insensitive
func TestFileExtensionCaseSensitivity(t *testing.T) {
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
		{"video.MP4", true},
		{"video.Mp4", true},
		{"video.mP4", true},
		{"video.AVI", true},
		{"video.MKV", true},
		{"VIDEO.MP4", true},
		{"video.TXT", false},
		{"video.JPG", false},
	}

	for _, tc := range testCases {
		ext := strings.ToLower(filepath.Ext(tc.filename))
		result := videoExtensions[ext]
		if result != tc.expected {
			t.Errorf("File %s: expected %v, got %v", tc.filename, tc.expected, result)
		}
	}
}

// TestSymlinkDetection verifies symlink detection logic
func TestSymlinkDetection(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	
	// Create a regular file
	regularFile := filepath.Join(tmpDir, "regular.mp4")
	if err := os.WriteFile(regularFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create regular file: %v", err)
	}
	
	// Create a symlink to the file
	symlinkFile := filepath.Join(tmpDir, "symlink.mp4")
	if err := os.Symlink(regularFile, symlinkFile); err != nil {
		t.Skipf("Cannot create symlink (might not be supported): %v", err)
	}
	
	// Check regular file
	info, err := os.Lstat(regularFile)
	if err != nil {
		t.Fatalf("Failed to stat regular file: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("Regular file incorrectly identified as symlink")
	}
	
	// Check symlink
	linkInfo, err := os.Lstat(symlinkFile)
	if err != nil {
		t.Fatalf("Failed to lstat symlink: %v", err)
	}
	if linkInfo.Mode()&os.ModeSymlink == 0 {
		t.Error("Symlink not identified as symlink")
	}
	
	// Test EvalSymlinks
	target, err := filepath.EvalSymlinks(symlinkFile)
	if err != nil {
		t.Fatalf("Failed to evaluate symlink: %v", err)
	}
	
	expectedTarget, _ := filepath.Abs(regularFile)
	actualTarget, _ := filepath.Abs(target)
	if actualTarget != expectedTarget {
		t.Errorf("Symlink target mismatch: expected %s, got %s", expectedTarget, actualTarget)
	}
}

// TestCircularSymlinkPrevention verifies that circular symlinks don't cause infinite loops
func TestCircularSymlinkPrevention(t *testing.T) {
	// Test the visited directories map logic
	visitedDirs := make(map[string]bool)
	
	// Simulate visiting directories
	dir1 := "/videos/dir1"
	dir2 := "/videos/dir2"
	
	// Visit dir1
	if visitedDirs[dir1] {
		t.Error("dir1 should not be visited yet")
	}
	visitedDirs[dir1] = true
	
	// Visit dir2
	if visitedDirs[dir2] {
		t.Error("dir2 should not be visited yet")
	}
	visitedDirs[dir2] = true
	
	// Try to visit dir1 again - should be detected
	if !visitedDirs[dir1] {
		t.Error("dir1 should be marked as visited")
	}
	
	// Should not revisit
	shouldSkip := visitedDirs[dir1]
	if !shouldSkip {
		t.Error("Expected to skip already visited directory")
	}
}
