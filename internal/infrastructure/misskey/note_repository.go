package misskey

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"misskeyRSSbot/internal/domain/entity"
	"misskeyRSSbot/internal/domain/repository"
)

type rateLimiter struct {
	mu         sync.Mutex
	permits    int
	maxPermits int
	refillRate time.Duration
	lastRefill time.Time
}

func newRateLimiter(maxPermits int, refillRate time.Duration) *rateLimiter {
	return &rateLimiter{
		permits:    maxPermits,
		maxPermits: maxPermits,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (rl *rateLimiter) Wait(ctx context.Context) error {
	rl.mu.Lock()

	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	permitsToAdd := int(elapsed / rl.refillRate)
	if permitsToAdd > 0 {
		rl.permits = min(rl.permits+permitsToAdd, rl.maxPermits)
		rl.lastRefill = now
	}

	if rl.permits <= 0 {
		waitTime := rl.refillRate - (now.Sub(rl.lastRefill) % rl.refillRate)
		rl.mu.Unlock()

		timer := time.NewTimer(waitTime)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			rl.mu.Lock()
			rl.permits = 1
			rl.lastRefill = time.Now()
			rl.permits--
			rl.mu.Unlock()
			return nil
		}
	}

	rl.permits--
	rl.mu.Unlock()
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type noteRepository struct {
	host        string
	authToken   string
	client      *http.Client
	rateLimiter *rateLimiter
}

type Config struct {
	Host           string
	AuthToken      string
	MaxPermits     int
	RefillInterval time.Duration
}

func NewNoteRepository(cfg Config) repository.NoteRepository {
	maxPermits := cfg.MaxPermits
	if maxPermits == 0 {
		maxPermits = 3
	}
	refillInterval := cfg.RefillInterval
	if refillInterval == 0 {
		refillInterval = 10 * time.Second
	}

	return &noteRepository{
		host:        cfg.Host,
		authToken:   cfg.AuthToken,
		client:      &http.Client{Timeout: 30 * time.Second},
		rateLimiter: newRateLimiter(maxPermits, refillInterval),
	}
}

func (r *noteRepository) Post(ctx context.Context, note *entity.Note) error {
	if err := r.rateLimiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limiter error: %w", err)
	}

	notePayload := map[string]interface{}{
		"i":          r.authToken,
		"text":       note.Text,
		"visibility": string(note.Visibility),
	}

	payload, err := json.Marshal(notePayload)
	if err != nil {
		return fmt.Errorf("failed to serialize note: %w", err)
	}

	url := r.host
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}
	url = url + "/api/notes/create"

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to Misskey API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Misskey API returned non-OK status: %d", resp.StatusCode)
	}

	return nil
}
