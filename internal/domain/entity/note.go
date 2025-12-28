package entity

import "fmt"

type NoteVisibility string

const (
	VisibilityPublic    NoteVisibility = "public"
	VisibilityHome      NoteVisibility = "home"
	VisibilityFollowers NoteVisibility = "followers"
	VisibilitySpecified NoteVisibility = "specified"
)

type Note struct {
	Text       string
	Visibility NoteVisibility
}

func NewNoteFromFeed(entry *FeedEntry, visibility NoteVisibility) *Note {
	text := fmt.Sprintf("📰 %s\n%s", entry.Title, entry.Link)
	return &Note{
		Text:       text,
		Visibility: visibility,
	}
}

func NewNote(text string, visibility NoteVisibility) *Note {
	return &Note{
		Text:       text,
		Visibility: visibility,
	}
}
