package entity

import (
	"testing"
	"time"
)

func TestNewFeedEntry(t *testing.T) {
	now := time.Now()
	entry := NewFeedEntry("Test Title", "https://example.tld", "Description", now, "guid-123")

	if entry.Title != "Test Title" {
		t.Errorf("expected title 'Test Title', got '%s'", entry.Title)
	}
	if entry.Link != "https://example.tld" {
		t.Errorf("expected link 'https://example.tld', got '%s'", entry.Link)
	}
	if entry.GUID != "guid-123" {
		t.Errorf("expected GUID 'guid-123', got '%s'", entry.GUID)
	}
}

func TestFeedEntry_IsNewerThan(t *testing.T) {
	baseTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		published time.Time
		compare   time.Time
		expected  bool
	}{
		{
			name:      "newer than",
			published: baseTime.Add(1 * time.Hour),
			compare:   baseTime,
			expected:  true,
		},
		{
			name:      "older than",
			published: baseTime.Add(-1 * time.Hour),
			compare:   baseTime,
			expected:  false,
		},
		{
			name:      "same time",
			published: baseTime,
			compare:   baseTime,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := NewFeedEntry("Test", "https://example.tld", "Desc", tt.published, "guid")
			result := entry.IsNewerThan(tt.compare)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
