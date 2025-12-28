package entity

import (
	"testing"
	"time"
)

func TestNewNoteFromFeed(t *testing.T) {
	now := time.Now()
	entry := NewFeedEntry("Test Article", "https://example.tld/article", "Description", now, "guid-1")

	note := NewNoteFromFeed(entry, VisibilityHome)

	expectedText := "📰 Test Article\nhttps://example.tld/article"
	if note.Text != expectedText {
		t.Errorf("expected text '%s', got '%s'", expectedText, note.Text)
	}

	if note.Visibility != VisibilityHome {
		t.Errorf("expected visibility %v, got %v", VisibilityHome, note.Visibility)
	}
}

func TestNewNote(t *testing.T) {
	note := NewNote("Test content", VisibilityPublic)

	if note.Text != "Test content" {
		t.Errorf("expected text 'Test content', got '%s'", note.Text)
	}

	if note.Visibility != VisibilityPublic {
		t.Errorf("expected visibility %v, got %v", VisibilityPublic, note.Visibility)
	}
}

func TestNoteVisibility(t *testing.T) {
	tests := []struct {
		name       string
		visibility NoteVisibility
		expected   string
	}{
		{"public", VisibilityPublic, "public"},
		{"home", VisibilityHome, "home"},
		{"followers", VisibilityFollowers, "followers"},
		{"specified", VisibilitySpecified, "specified"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.visibility) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(tt.visibility))
			}
		})
	}
}
