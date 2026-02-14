package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)
type RSSSettings struct {
    URL    string
    Filter bool
}
type Config struct {
	MisskeyHost string   `envconfig:"MISSKEY_HOST" required:"true"`
	AuthToken   string   `envconfig:"AUTH_TOKEN" required:"true"`
	RSSURL      []RSSSettings

	FetchInterval int `envconfig:"FETCH_INTERVAL" default:"30"`

	MaxPermits int `envconfig:"MAX_PERMITS" default:"3"`

	RefillInterval int `envconfig:"REFILL_INTERVAL" default:"10"`

	LocalOnly bool `envconfig:"LOCAL_ONLY" default:"false"`

	LLMProvider          string `envconfig:"LLM_PROVIDER" default:""`
	LLMAPIKey            string `envconfig:"LLM_API_KEY"`
	LLMModel             string `envconfig:"LLM_MODEL"`
	LLMMaxTokens         int    `envconfig:"LLM_MAX_TOKENS" default:"0"`
	LLMTimeout           int    `envconfig:"LLM_TIMEOUT" default:"30"`
	LLMSystemInstruction string `envconfig:"LLM_SYSTEM_INSTRUCTION"`

	CacheDBPath string `envconfig:"CACHE_DB_PATH" default:""`

	CacheCleanupInterval int `envconfig:"CACHE_CLEANUP_INTERVAL" default:"24"`

	CacheRetentionDays int `envconfig:"CACHE_RETENTION_DAYS" default:"7"`

	FirstRunLatestOnly bool `envconfig:"FIRST_RUN_LATEST_ONLY" default:"true"`
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}

	rssSettings := loadRSSURLs()
    if len(rssSettings) > 0 {
        cfg.RSSURL = rssSettings
    }

    if len(cfg.RSSURL) == 0 {
        return nil, fmt.Errorf("no RSS URLs configured")
    }

	return &cfg, nil
}

func loadRSSURLs() []RSSSettings {
    var settings []RSSSettings

    for i := 1; ; i++ {
        key := fmt.Sprintf("RSS_URL_%d", i)
        url := os.Getenv(key)
        if url == "" {
            break
        }
        
        // フィルター設定を読み込む
        filterKey := fmt.Sprintf("RSS_URL_%d_FILTER", i)
        filter := os.Getenv(filterKey) == "true"
        
        settings = append(settings, RSSSettings{
            URL:    url,
            Filter: filter,
        })
    }
    return settings
}

func GetNumberedEnvInt(prefix string, index int, defaultValue int) int {
	key := fmt.Sprintf("%s_%d", prefix, index)
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return intVal
}

func (c *Config) GetFetchInterval() time.Duration {
	return time.Duration(c.FetchInterval) * time.Second
}

func (c *Config) GetRefillInterval() time.Duration {
	return time.Duration(c.RefillInterval) * time.Second
}

type LLMConfig struct {
	Provider          string
	APIKey            string
	Model             string
	MaxTokens         int
	Timeout           time.Duration
	SystemInstruction string
}

func (c *Config) GetLLMConfig() LLMConfig {
	return LLMConfig{
		Provider:          c.LLMProvider,
		APIKey:            c.LLMAPIKey,
		Model:             c.LLMModel,
		MaxTokens:         c.LLMMaxTokens,
		Timeout:           time.Duration(c.LLMTimeout) * time.Second,
		SystemInstruction: c.LLMSystemInstruction,
	}
}

func (c *Config) IsPersistentCache() bool {
	return c.CacheDBPath != ""
}

func (c *Config) GetCacheCleanupInterval() time.Duration {
	return time.Duration(c.CacheCleanupInterval) * time.Hour
}

func (c *Config) GetCacheRetentionPeriod() time.Duration {
	return time.Duration(c.CacheRetentionDays) * 24 * time.Hour
}
