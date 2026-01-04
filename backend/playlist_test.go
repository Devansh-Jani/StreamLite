package main

import "testing"

func TestNormalizePlaylistName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"video_v1.mp4", "video"},
		{"video_v2.mp4", "video"},
		{"video_edited.mp4", "video"},
		{"video_final.mp4", "video"},
		{"my_video v1.mp4", "my video"},
		{"tutorial-v2.mp4", "tutorial"},
		{"episode_1.mp4", "episode"},
		{"episode_2.mp4", "episode"},
		{"episode_10.mp4", "episode"},
		{"movie (edited).mp4", "movie"},
		{"movie-final.mp4", "movie"},
		{"documentary_draft.mp4", "documentary"},
		{"unique_video.mp4", "unique video"},
		{"Video_Name_v3.mp4", "video name"},
		{"test-video-1.mp4", "test video"},
	}

	for _, test := range tests {
		result := normalizePlaylistName(test.input)
		if result != test.expected {
			t.Errorf("normalizePlaylistName(%q) = %q; expected %q", test.input, result, test.expected)
		}
	}
}
