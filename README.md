# miniflux-cli

Small Go CLI for the [Miniflux API](https://miniflux.app/docs/api.html).

## Features

- Reads configuration from environment variables or flags
- Supports Basic Auth or API token auth
- Full feed management (CRUD): list/get/create/update/delete
- Feed refresh support: refresh all feeds or one feed
- Commands focused on article viewing: `entries` and `entry`
- Includes a multi-architecture Docker build
- Includes a `compose.yaml` for `nerdctl.lima compose up`

## Environment variables

Create your local env file:

```bash
cp .env.example .env
```

Then edit `.env` with your own values:

```bash
MINIFLUX_BASE_URL="http://127.0.0.1:8080/v1/"
MINIFLUX_USERNAME="your-username"
MINIFLUX_PASSWORD="your-password"
# MINIFLUX_API_TOKEN="your-token"
```

`.env` is ignored by git (`.gitignore`) to prevent leaking personal credentials.

Use API token auth (without username/password) by setting:

```bash
MINIFLUX_BASE_URL="http://127.0.0.1:8080/v1/"
MINIFLUX_API_TOKEN="your-token"
```

When `MINIFLUX_API_TOKEN` is set, the CLI sends it in `X-Auth-Token`.

## Local build

```bash
set -a; source .env; set +a
go build -o bin/miniflux-cli .
./bin/miniflux-cli feeds
./bin/miniflux-cli feed get 115
./bin/miniflux-cli feed create --feed-url "https://example.com/feed.xml" --category-id 1
./bin/miniflux-cli feed update --title "New Title" 115
./bin/miniflux-cli feed delete 115
./bin/miniflux-cli feeds refresh
./bin/miniflux-cli feed refresh 115
./bin/miniflux-cli entries
./bin/miniflux-cli entries --status read --limit 10
./bin/miniflux-cli entry 12345
./bin/miniflux-cli health
```

## Docker build

Build the local image:

```bash
nerdctl.lima build -t miniflux-cli:local .
```

Build multi-arch images with BuildKit:

```bash
nerdctl.lima build \
  --platform=linux/amd64,linux/arm64 \
  -t miniflux-cli:multiarch .
```

## Run with compose

```bash
set -a; source .env; set +a
nerdctl.lima compose up --build
```

## Feature usage

All examples below assume environment variables are loaded:

```bash
set -a; source .env; set +a
```

Check server health (`GET /healthcheck`):

```bash
./bin/miniflux-cli health
```

List all feeds (`GET /v1/feeds`):

```bash
./bin/miniflux-cli feeds
./bin/miniflux-cli feed list
```

Get one feed (`GET /v1/feeds/{id}`):

```bash
./bin/miniflux-cli feed get 115
```

Create a feed (`POST /v1/feeds`):

```bash
./bin/miniflux-cli feed create \
  --feed-url "https://example.com/feed.xml" \
  --category-id 1
```

Update a feed (`PUT /v1/feeds/{id}`):

```bash
./bin/miniflux-cli feed update --title "Updated feed title" 115
./bin/miniflux-cli feed update --feed-url "https://example.com/new.xml" --category-id 2 115
```

Delete a feed (`DELETE /v1/feeds/{id}`):

```bash
./bin/miniflux-cli feed delete 115
```

Refresh all feeds (`PUT /v1/feeds/refresh`):

```bash
./bin/miniflux-cli feeds refresh
```

Refresh one feed (`PUT /v1/feeds/{id}/refresh`):

```bash
./bin/miniflux-cli feed refresh 115
```

List entries (`GET /v1/entries`) with optional filters:

Default behavior: `entries` without `--status` uses `unread`.

```bash
./bin/miniflux-cli entries
./bin/miniflux-cli entries --status unread --limit 20 --offset 0
./bin/miniflux-cli entries --status read --limit 50
./bin/miniflux-cli entries --feed-id 115 --limit 20
./bin/miniflux-cli entries --category-id 11 --status unread --limit 20
```

Show a single entry with full content (`GET /v1/entries/{id}`):

```bash
./bin/miniflux-cli entry 12345
```

Override configuration with flags (takes priority over env):

```bash
./bin/miniflux-cli \
  --debug \
  --base-url "http://127.0.0.1:8080/v1/" \
  --token "your-token" \
  entries --status unread --limit 10
```
