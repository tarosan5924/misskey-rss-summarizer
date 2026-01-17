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

// NewNoteFromFeedWithSummary は要約付きでFeedEntryからNoteを生成します
func NewNoteFromFeedWithSummary(entry *FeedEntry, summary string, visibility NoteVisibility) *Note {
	var text string

	if summary != "" {
		text = fmt.Sprintf("📰 %s\n\n【要約】\n%s\n\n%s", entry.Title, summary, entry.Link)
	} else {
		text = fmt.Sprintf("📰 %s\n%s", entry.Title, entry.Link)
	}

	return &Note{
		Text:       text,
		Visibility: visibility,
	}
}
