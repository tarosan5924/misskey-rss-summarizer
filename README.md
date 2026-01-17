# Misskey RSS Bot with AI Summarization

A bot that fetches RSS feeds and automatically posts them to Misskey.

## Features

- Fetch RSS feeds at regular intervals
- Automatic posting to Misskey with rate limiting
- **Optional AI-powered article summarization** (using LLM providers like Google Gemini)

## Setup

Create a `.env` file based on `.env.example`.

### LLM Summarization (Optional)

To enable AI-powered article summarization, add the following to your `.env` file:

```bash
LLM_PROVIDER=gemini
LLM_API_KEY=your_api_key_here
LLM_MODEL=<check_provider_documentation_for_available_models>
```

The bot will automatically fetch article content from URLs and generate summaries using the specified LLM provider.

**Supported Providers:**
- `gemini` - Google Gemini API

**Important:** Check the LLM provider's documentation for currently available models, as model names may change over time.

**Note:** LLM summarization is opt-in. If `LLM_PROVIDER` is not set or empty, the bot will post articles without summaries.

### Build and Run

```bash
go build
./misskeyRSSbot
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
WorkingDirectory=/path/to/misskeyRSSbot
ExecStart=/path/to/misskeyRSSbot/misskeyRSSbot
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```
