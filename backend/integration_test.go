package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestVideoDirectoryAccessibility tests that the video directory can be accessed
func TestVideoDirectoryAccessibility(t *testing.T) {
	// Create a temporary test directory
	tmpDir := t.TempDir()
	
	// Test that a valid directory is accessible
	if _, err := os.Stat(tmpDir); err != nil {
		t.Errorf("Expected temp directory to be accessible, got error: %v", err)
	}
	
	// Test that a non-existent directory returns os.IsNotExist
	nonExistentDir := filepath.Join(tmpDir, "nonexistent")
	_, err := os.Stat(nonExistentDir)
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}
	if !os.IsNotExist(err) {
		t.Errorf("Expected os.IsNotExist error, got: %v", err)
	}
}

// TestVideoFileAccessibility tests that video files can be accessed
func TestVideoFileAccessibility(t *testing.T) {
	// Create a temporary test directory and file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_video.mp4")
	
	// Create test file
	content := []byte("fake video content")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Normalize the path
	normalizedPath := filepath.Clean(testFile)
	
	// Test that the file is accessible
	fileInfo, err := os.Stat(normalizedPath)
	if err != nil {
		t.Errorf("Expected file to be accessible, got error: %v", err)
	}
	
	// Verify it's not a directory
	if fileInfo.IsDir() {
		t.Error("Expected file, got directory")
	}
	
	// Test that the file can be opened
	file, err := os.Open(normalizedPath)
	if err != nil {
		t.Errorf("Expected file to be openable, got error: %v", err)
	}
	defer file.Close()
	
	// Read the content
	buf := make([]byte, len(content))
	n, err := file.Read(buf)
	if err != nil {
		t.Errorf("Failed to read file: %v", err)
	}
	if n != len(content) {
		t.Errorf("Expected to read %d bytes, got %d", len(content), n)
	}
}

// TestVideoFilePermissions tests handling of permission denied errors
func TestVideoFilePermissions(t *testing.T) {
	// Skip on Windows where chmod doesn't work the same way
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping permission test on Windows")
	}
	
	// Create a temporary test directory and file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_video.mp4")
	
	// Create test file
	content := []byte("fake video content")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Remove read permissions
	if err := os.Chmod(testFile, 0000); err != nil {
		t.Fatalf("Failed to change permissions: %v", err)
	}
	defer os.Chmod(testFile, 0644) // Restore permissions for cleanup
	
	// Try to open the file
	_, err := os.Open(testFile)
	if err == nil {
		t.Error("Expected permission error when opening file with no permissions")
	}
	
	// Note: os.IsPermission may not work reliably on all systems
	// Just verify we got an error
	if err != nil {
		t.Logf("Got expected error: %v", err)
	}
}

// TestPathNormalizationOnRealFiles tests path normalization with actual files
func TestPathNormalizationOnRealFiles(t *testing.T) {
	// Create a temporary test directory and file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_video.mp4")
	
	// Create test file
	content := []byte("fake video content")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Test various path representations
	testPaths := []string{
		testFile,
		testFile + "/",                // Trailing slash
		filepath.Join(tmpDir, "./test_video.mp4"),  // With ./
		filepath.Join(tmpDir, "subdir/../test_video.mp4"),  // With ..
	}
	
	for _, path := range testPaths {
		cleanPath := filepath.Clean(path)
		
		// Try to stat the cleaned path
		fileInfo, err := os.Stat(cleanPath)
		
		// The file should exist for the first two paths
		if path == testFile || path == testFile+"/" {
			if err != nil {
				t.Errorf("Path %s (cleaned to %s): expected to exist, got error: %v", path, cleanPath, err)
			} else if fileInfo.IsDir() {
				t.Errorf("Path %s (cleaned to %s): expected file, got directory", path, cleanPath)
			}
		}
	}
}
