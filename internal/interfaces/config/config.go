package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	MisskeyHost string   `envconfig:"MISSKEY_HOST" required:"true"`
	AuthToken   string   `envconfig:"AUTH_TOKEN" required:"true"`
	RSSURL      []string `envconfig:"RSS_URL"`

	FetchInterval int `envconfig:"FETCH_INTERVAL" default:"30"`

	MaxPermits int `envconfig:"MAX_PERMITS" default:"3"`

	RefillInterval int `envconfig:"REFILL_INTERVAL" default:"10"`
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}

	rssURLs := loadRSSURLs()
	if len(rssURLs) > 0 {
		cfg.RSSURL = rssURLs
	}

	if len(cfg.RSSURL) == 0 {
		return nil, fmt.Errorf("no RSS URLs configured. Please set RSS_URL or RSS_URL_1, RSS_URL_2, etc.")
	}

	return &cfg, nil
}

func loadRSSURLs() []string {
	var urls []string

	for i := 1; ; i++ {
		key := fmt.Sprintf("RSS_URL_%d", i)
		url := os.Getenv(key)
		if url == "" {
			break
		}
		urls = append(urls, url)
	}

	if len(urls) > 0 {
		return urls
	}

	return nil
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
