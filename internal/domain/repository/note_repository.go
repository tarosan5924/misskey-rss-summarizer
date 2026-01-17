package repository

import (
	"context"

	"misskey-rss-summarizer/internal/domain/entity"
)

type NoteRepository interface {
	Post(ctx context.Context, note *entity.Note) error
}
