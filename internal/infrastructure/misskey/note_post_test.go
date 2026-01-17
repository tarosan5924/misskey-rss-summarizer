package misskey

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"misskeyRSSbot/internal/domain/entity"
)

func TestNoteRepository_Post_Success(t *testing.T) {
	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/notes/create" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("unexpected method: %s", r.Method)
		}

		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("unexpected content-type: %s", ct)
		}

		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedPayload)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"createdNote": {"id": "note123"}}`))
	}))
	defer server.Close()

	repo := &noteRepository{
		host:        server.URL,
		authToken:   "test-token",
		client:      &http.Client{Timeout: 30 * time.Second},
		rateLimiter: newRateLimiter(3, 10*time.Second),
		localOnly:   false,
	}

	note := entity.NewNote("Test note content", entity.VisibilityHome)
	ctx := context.Background()

	err := repo.Post(ctx, note)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedPayload["i"] != "test-token" {
		t.Errorf("expected auth token 'test-token', got '%v'", receivedPayload["i"])
	}
	if receivedPayload["text"] != "Test note content" {
		t.Errorf("expected text 'Test note content', got '%v'", receivedPayload["text"])
	}
	if receivedPayload["visibility"] != "home" {
		t.Errorf("expected visibility 'home', got '%v'", receivedPayload["visibility"])
	}
	if receivedPayload["localOnly"] != false {
		t.Errorf("expected localOnly to be false, got '%v'", receivedPayload["localOnly"])
	}
}

func TestNoteRepository_Post_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal server error"}`))
	}))
	defer server.Close()

	repo := &noteRepository{
		host:        server.URL,
		authToken:   "test-token",
		client:      &http.Client{Timeout: 30 * time.Second},
		rateLimiter: newRateLimiter(3, 10*time.Second),
		localOnly:   false,
	}

	note := entity.NewNote("Test note", entity.VisibilityPublic)
	ctx := context.Background()

	err := repo.Post(ctx, note)
	if err == nil {
		t.Error("expected error for server error response, got nil")
	}
}

func TestNoteRepository_Post_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Unauthorized"}`))
	}))
	defer server.Close()

	repo := &noteRepository{
		host:        server.URL,
		authToken:   "invalid-token",
		client:      &http.Client{Timeout: 30 * time.Second},
		rateLimiter: newRateLimiter(3, 10*time.Second),
		localOnly:   false,
	}

	note := entity.NewNote("Test note", entity.VisibilityPublic)
	ctx := context.Background()

	err := repo.Post(ctx, note)
	if err == nil {
		t.Error("expected error for unauthorized response, got nil")
	}
}

func TestNoteRepository_Post_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	repo := &noteRepository{
		host:        server.URL,
		authToken:   "test-token",
		client:      &http.Client{Timeout: 30 * time.Second},
		rateLimiter: newRateLimiter(3, 10*time.Second),
		localOnly:   false,
	}

	note := entity.NewNote("Test note", entity.VisibilityPublic)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := repo.Post(ctx, note)
	if err == nil {
		t.Error("expected error for cancelled context, got nil")
	}
}

func TestNoteRepository_Post_DifferentVisibilities(t *testing.T) {
	visibilities := []entity.NoteVisibility{
		entity.VisibilityPublic,
		entity.VisibilityHome,
		entity.VisibilityFollowers,
	}

	for _, vis := range visibilities {
		t.Run(string(vis), func(t *testing.T) {
			var receivedVis string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var payload map[string]interface{}
				body, _ := io.ReadAll(r.Body)
				json.Unmarshal(body, &payload)
				receivedVis = payload["visibility"].(string)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{}`))
			}))
			defer server.Close()

			repo := &noteRepository{
				host:        server.URL,
				authToken:   "test-token",
				client:      &http.Client{Timeout: 30 * time.Second},
				rateLimiter: newRateLimiter(3, 10*time.Second),
				localOnly:   false,
			}

			note := entity.NewNote("Test", vis)
			ctx := context.Background()

			if err := repo.Post(ctx, note); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if receivedVis != string(vis) {
				t.Errorf("expected visibility '%s', got '%s'", vis, receivedVis)
			}
		})
	}
}

func TestNoteRepository_Post_LocalOnlyTrue(t *testing.T) {
	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedPayload)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"createdNote": {"id": "note123"}}`))
	}))
	defer server.Close()

	repo := &noteRepository{
		host:        server.URL,
		authToken:   "test-token",
		client:      &http.Client{Timeout: 30 * time.Second},
		rateLimiter: newRateLimiter(3, 10*time.Second),
		localOnly:   true,
	}

	note := entity.NewNote("Test note", entity.VisibilityPublic)
	ctx := context.Background()

	err := repo.Post(ctx, note)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedPayload["localOnly"] != true {
		t.Errorf("expected localOnly to be true, got '%v'", receivedPayload["localOnly"])
	}
}
