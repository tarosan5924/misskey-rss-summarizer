package entity

import "time"

type FeedEntry struct {
	Title       string
	Link        string
	Description string
	Published   time.Time
	GUID        string
}

func NewFeedEntry(title, link, description string, published time.Time, guid string) *FeedEntry {
	return &FeedEntry{
		Title:       title,
		Link:        link,
		Description: description,
		Published:   published,
		GUID:        guid,
	}
}

func (f *FeedEntry) IsNewerThan(t time.Time) bool {
	return f.Published.After(t)
}
