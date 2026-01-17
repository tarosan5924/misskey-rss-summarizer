# Misskey RSS Summarizer

A bot that fetches RSS feeds and automatically posts them to Misskey.

## Setup

Create a `.env` file based on `.env.example`.

### Build and Run

```bash
go build
./misskey-rss-summarizer
```

### Running as a systemd Service

Example systemd service configuration:

```ini
[Unit]
Description=Misskey RSS Bot
After=network.target

[Service]
Type=simple
User=youruser
WorkingDirectory=/path/to/misskey-rss-summarizer
ExecStart=/path/to/misskey-rss-summarizer/misskey-rss-summarizer
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```
