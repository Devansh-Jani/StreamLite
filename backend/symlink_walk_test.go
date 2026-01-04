package main

import (
"os"
"path/filepath"
"strings"
"testing"
)

// TestWalkWithSymlinksTraversesSiblings ensures that when a symlinked directory
// is encountered, the walk continues to process sibling files and directories
func TestWalkWithSymlinksTraversesSiblings(t *testing.T) {
// Create a temporary directory structure
tmpDir := t.TempDir()

// Create main directory with a video
mainDir := filepath.Join(tmpDir, "main")
if err := os.Mkdir(mainDir, 0755); err != nil {
t.Fatalf("Failed to create main dir: %v", err)
}
mainVideo := filepath.Join(mainDir, "main_video.mp4")
if err := os.WriteFile(mainVideo, []byte("main"), 0644); err != nil {
t.Fatalf("Failed to create main video: %v", err)
}

// Create external directory with videos
externalDir := filepath.Join(tmpDir, "external")
if err := os.Mkdir(externalDir, 0755); err != nil {
t.Fatalf("Failed to create external dir: %v", err)
}
externalVideo := filepath.Join(externalDir, "external_video.mp4")
if err := os.WriteFile(externalVideo, []byte("external"), 0644); err != nil {
t.Fatalf("Failed to create external video: %v", err)
}

// Create a subdirectory with a video
subDir := filepath.Join(tmpDir, "subdir")
if err := os.Mkdir(subDir, 0755); err != nil {
t.Fatalf("Failed to create subdir: %v", err)
}
subVideo := filepath.Join(subDir, "sub_video.mp4")
if err := os.WriteFile(subVideo, []byte("sub"), 0644); err != nil {
t.Fatalf("Failed to create sub video: %v", err)
}

// Create a symlink to external directory inside tmpDir
symlinkPath := filepath.Join(tmpDir, "linked")
if err := os.Symlink(externalDir, symlinkPath); err != nil {
t.Skipf("Cannot create symlink: %v", err)
}

// Walk the directory and collect all .mp4 files
foundFiles := make(map[string]bool)
visitedDirs := make(map[string]bool)

err := walkWithSymlinks(tmpDir, visitedDirs, func(path string, info os.FileInfo, err error) error {
if err != nil {
return nil
}
if info.IsDir() {
return nil
}
if strings.ToLower(filepath.Ext(info.Name())) == ".mp4" {
foundFiles[filepath.Base(path)] = true
}
return nil
})

if err != nil {
t.Fatalf("walkWithSymlinks failed: %v", err)
}

// We expect to find all three videos:
// 1. main_video.mp4 from main/
// 2. sub_video.mp4 from subdir/
// 3. external_video.mp4 from linked/ (symlink to external/)
expectedFiles := map[string]bool{
"main_video.mp4":     true,
"sub_video.mp4":      true,
"external_video.mp4": true,
}

if len(foundFiles) != len(expectedFiles) {
t.Errorf("Expected %d files, found %d", len(expectedFiles), len(foundFiles))
t.Logf("Expected: %v", expectedFiles)
t.Logf("Found: %v", foundFiles)
}

for expectedFile := range expectedFiles {
if !foundFiles[expectedFile] {
t.Errorf("Expected file %s not found", expectedFile)
}
}

// Verify that all expected files were found
for foundFile := range foundFiles {
if !expectedFiles[foundFile] {
t.Errorf("Unexpected file found: %s", foundFile)
}
}
}
