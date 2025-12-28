package repository

import (
	"context"

	"misskeyRSSbot/internal/domain/entity"
)

type NoteRepository interface {
	Post(ctx context.Context, note *entity.Note) error
}
