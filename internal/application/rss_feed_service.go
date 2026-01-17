package application

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"misskeyRSSbot/internal/domain/entity"
	"misskeyRSSbot/internal/domain/repository"
)

type RSSFeedService struct {
	feedRepo       repository.FeedRepository
	noteRepo       repository.NoteRepository
	cacheRepo      repository.CacheRepository
	summarizerRepo repository.SummarizerRepository
}

func NewRSSFeedService(
	feedRepo repository.FeedRepository,
	noteRepo repository.NoteRepository,
	cacheRepo repository.CacheRepository,
	summarizerRepo repository.SummarizerRepository,
) *RSSFeedService {
	return &RSSFeedService{
		feedRepo:       feedRepo,
		noteRepo:       noteRepo,
		cacheRepo:      cacheRepo,
		summarizerRepo: summarizerRepo,
	}
}

func (s *RSSFeedService) ProcessFeed(ctx context.Context, rssURL string) error {
	entries, err := s.feedRepo.Fetch(ctx, rssURL)
	if err != nil {
		return fmt.Errorf("failed to fetch RSS feed [%s]: %w", rssURL, err)
	}

	if len(entries) == 0 {
		log.Printf("No entries found in RSS URL: %s", rssURL)
		return nil
	}

	latestPublished, err := s.cacheRepo.GetLatestPublishedTime(ctx, rssURL)
	if err != nil {
		return fmt.Errorf("failed to get latest published time: %w", err)
	}

	isFirstRun := latestPublished.IsZero()

	var newEntries []*entity.FeedEntry

	if isFirstRun {
		var mostRecent *entity.FeedEntry
		for _, entry := range entries {
			if mostRecent == nil || entry.Published.After(mostRecent.Published) {
				mostRecent = entry
			}
		}
		if mostRecent != nil {
			newEntries = append(newEntries, mostRecent)
		}
	} else {
		for _, entry := range entries {
			processed, err := s.cacheRepo.IsProcessed(ctx, entry.GUID)
			if err != nil {
				log.Printf("Failed to check if processed [GUID: %s]: %v", entry.GUID, err)
				continue
			}
			if processed {
				continue
			}

			if entry.IsNewerThan(latestPublished) {
				newEntries = append(newEntries, entry)
			}
		}
	}

	sortEntriesByPublishedAsc(newEntries)

	var latestTime time.Time
	for _, entry := range newEntries {
		var summary string

		if s.summarizerRepo != nil && s.summarizerRepo.IsEnabled() {
			var err error
			summary, err = s.summarizerRepo.Summarize(ctx, entry.Link, entry.Title)
			if err != nil {
				log.Printf("Failed to summarize [%s]: %v", entry.Title, err)
			}
		}

		note := entity.NewNoteFromFeedWithSummary(entry, summary, entity.VisibilityHome)
		if err := s.noteRepo.Post(ctx, note); err != nil {
			log.Printf("Failed to post to Misskey [%s]: %v", entry.Title, err)
			continue
		}

		log.Printf("Posted to Misskey: %s", entry.Title)

		if err := s.cacheRepo.MarkAsProcessed(ctx, entry.GUID); err != nil {
			log.Printf("Failed to mark as processed [GUID: %s]: %v", entry.GUID, err)
		}

		if entry.Published.After(latestTime) {
			latestTime = entry.Published
		}
	}

	if !latestTime.IsZero() {
		if err := s.cacheRepo.SaveLatestPublishedTime(ctx, rssURL, latestTime); err != nil {
			return fmt.Errorf("failed to save latest published time: %w", err)
		}
	}

	if len(newEntries) > 0 {
		log.Printf("Processed %d new entries from RSS URL [%s]", len(newEntries), rssURL)
	}

	return nil
}

func (s *RSSFeedService) ProcessAllFeeds(ctx context.Context, rssURLs []string) error {
	for _, url := range rssURLs {
		if err := s.ProcessFeed(ctx, url); err != nil {
			log.Printf("RSS processing error [%s]: %v", url, err)
			continue
		}
	}
	return nil
}

func sortEntriesByPublishedAsc(entries []*entity.FeedEntry) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Published.Before(entries[j].Published)
	})
}
