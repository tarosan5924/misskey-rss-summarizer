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
	feedRepo           repository.FeedRepository
	noteRepo           repository.NoteRepository
	cacheRepo          repository.CacheRepository
	summarizerRepo     repository.SummarizerRepository
	firstRunLatestOnly bool
}

type RSSFeedServiceOption func(*RSSFeedService)

func WithFirstRunLatestOnly(enabled bool) RSSFeedServiceOption {
	return func(s *RSSFeedService) {
		s.firstRunLatestOnly = enabled
	}
}

func NewRSSFeedService(
	feedRepo repository.FeedRepository,
	noteRepo repository.NoteRepository,
	cacheRepo repository.CacheRepository,
	summarizerRepo repository.SummarizerRepository,
	opts ...RSSFeedServiceOption,
) *RSSFeedService {
	s := &RSSFeedService{
		feedRepo:           feedRepo,
		noteRepo:           noteRepo,
		cacheRepo:          cacheRepo,
		summarizerRepo:     summarizerRepo,
		firstRunLatestOnly: true,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
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
	newEntries := s.filterNewEntries(ctx, entries, latestPublished, isFirstRun)

	if len(newEntries) == 0 {
		return nil
	}

	sortEntriesByPublishedAsc(newEntries)
	latestTime := s.postEntries(ctx, newEntries)

	if !latestTime.IsZero() {
		if err := s.cacheRepo.SaveLatestPublishedTime(ctx, rssURL, latestTime); err != nil {
			return fmt.Errorf("failed to save latest published time: %w", err)
		}
	}

	log.Printf("Processed %d new entries from RSS URL [%s]", len(newEntries), rssURL)
	return nil
}

func (s *RSSFeedService) filterNewEntries(
	ctx context.Context,
	entries []*entity.FeedEntry,
	latestPublished time.Time,
	isFirstRun bool,
) []*entity.FeedEntry {
	if isFirstRun && s.firstRunLatestOnly {
		return s.findMostRecentEntry(entries)
	}

	var newEntries []*entity.FeedEntry
	for _, entry := range entries {
		if s.shouldSkipEntry(ctx, entry, latestPublished, isFirstRun) {
			continue
		}
		newEntries = append(newEntries, entry)
	}
	return newEntries
}

func (s *RSSFeedService) findMostRecentEntry(entries []*entity.FeedEntry) []*entity.FeedEntry {
	if len(entries) == 0 {
		return nil
	}

	mostRecent := entries[0]
	for _, entry := range entries[1:] {
		if entry.Published.After(mostRecent.Published) {
			mostRecent = entry
		}
	}
	return []*entity.FeedEntry{mostRecent}
}

func (s *RSSFeedService) shouldSkipEntry(
	ctx context.Context,
	entry *entity.FeedEntry,
	latestPublished time.Time,
	isFirstRun bool,
) bool {
	processed, err := s.cacheRepo.IsProcessed(ctx, entry.GUID)
	if err != nil {
		log.Printf("Failed to check if processed [GUID: %s]: %v", entry.GUID, err)
		return true
	}
	if processed {
		return true
	}

	if !isFirstRun && !entry.IsNewerThan(latestPublished) {
		return true
	}

	return false
}

func (s *RSSFeedService) postEntries(ctx context.Context, entries []*entity.FeedEntry) time.Time {
	var latestTime time.Time

	for _, entry := range entries {
		summary := s.summarizeEntry(ctx, entry)

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

	return latestTime
}

func (s *RSSFeedService) summarizeEntry(ctx context.Context, entry *entity.FeedEntry) string {
	if s.summarizerRepo == nil || !s.summarizerRepo.IsEnabled() {
		return ""
	}

	summary, err := s.summarizerRepo.Summarize(ctx, entry.Link, entry.Title)
	if err != nil {
		log.Printf("Failed to summarize [%s]: %v", entry.Title, err)
		return ""
	}
	return summary
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
