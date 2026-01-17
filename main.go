package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"misskeyRSSbot/internal/application"
	"misskeyRSSbot/internal/infrastructure/llm"
	"misskeyRSSbot/internal/infrastructure/misskey"
	"misskeyRSSbot/internal/infrastructure/rss"
	"misskeyRSSbot/internal/infrastructure/storage"
	"misskeyRSSbot/internal/interfaces/config"
)

func main() {
	fmt.Println("Starting Misskey RSS Bot...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	feedRepo := rss.NewFeedRepository()
	noteRepo := misskey.NewNoteRepository(misskey.Config{
		Host:           cfg.MisskeyHost,
		AuthToken:      cfg.AuthToken,
		MaxPermits:     cfg.MaxPermits,
		RefillInterval: cfg.GetRefillInterval(),
		LocalOnly:      cfg.LocalOnly,
	})
	cacheRepo := storage.NewMemoryCacheRepository()

	// LLM要約機能のセットアップ
	llmCfg := cfg.GetLLMConfig()
	summarizerRepo, err := llm.NewSummarizerRepository(ctx, llm.Config{
		Provider:          llmCfg.Provider,
		APIKey:            llmCfg.APIKey,
		Model:             llmCfg.Model,
		MaxTokens:         llmCfg.MaxTokens,
		Timeout:           llmCfg.Timeout,
		SystemInstruction: llmCfg.SystemInstruction,
	})
	if err != nil {
		log.Printf("Warning: LLM summarizer initialization failed: %v", err)
		log.Println("Continuing without summarization feature...")
		summarizerRepo, _ = llm.NewSummarizerRepository(ctx, llm.Config{Provider: "noop"})
	}

	service := application.NewRSSFeedService(
		feedRepo,
		noteRepo,
		cacheRepo,
		summarizerRepo,
	)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("Shutdown signal received")
		cancel()
	}()

	interval := cfg.GetFetchInterval()
	log.Printf("RSS fetch interval: %v", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Println("Fetching RSS feeds...")
	if err := service.ProcessAllFeeds(ctx, cfg.RSSURL); err != nil {
		log.Printf("RSS processing error: %v", err)
	}
	log.Println("RSS feeds fetched")

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down...")
			return
		case <-ticker.C:
			log.Println("Fetching RSS feeds...")
			if err := service.ProcessAllFeeds(ctx, cfg.RSSURL); err != nil {
				log.Printf("RSS processing error: %v", err)
			}
			log.Println("RSS feeds fetched")
		}
	}
}
