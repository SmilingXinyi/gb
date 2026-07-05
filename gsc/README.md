# gsc

`gsc` is a Go client for the [Google Search Console API](https://developers.google.com/webmaster-tools). It wraps the official API with a simplified interface for search analytics, URL inspection, site management, and sitemap operations.

## Features

- **Search analytics**: Query performance data by date range, dimensions, and row limits.
- **URL inspection**: Check indexing status and crawl details for individual URLs.
- **Site management**: List, get, add, and remove Search Console properties.
- **Sitemap management**: List, submit, and delete sitemaps.
- **Flexible auth**: Service account JSON (file or inline), or Application Default Credentials (ADC).
- **Environment-based config**: Load settings from `GSC_*` or legacy `GOOGLE_*` variables.

## Installation

```bash
go get github.com/SmilingXinyi/gb/gsc@latest
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"

	"github.com/SmilingXinyi/gb/gsc"
	"github.com/SmilingXinyi/gb/log"
)

func main() {
	log.Setup(log.DefaultConfig())

	ctx := context.Background()
	client, err := gsc.NewClientFromEnv(ctx)
	if err != nil {
		panic(err)
	}

	// List all sites in the account
	sites, err := client.ListSites(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Sites: %+v\n", sites)

	// Query top queries for the last 7 days
	analytics, err := client.QuerySearchAnalyticsLastDays(ctx, 7, []string{"query"}, 10)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Analytics: %+v\n", analytics)
}
```

## Configuration

Set the following environment variables before running:

| Variable | Description |
| :--- | :--- |
| `GSC_SERVICE_ACCOUNT_KEY_FILE` | Path to a service account JSON key file |
| `GSC_SERVICE_ACCOUNT_KEY_JSON` | Inline service account JSON (useful in CI/CD) |
| `GSC_SITE_URL` | Default property URL (e.g. `https://example.com/` or `sc-domain:example.com`) |
| `GSC_READONLY` | Set to `1` or `true` to restrict OAuth scopes to read-only |

Legacy `GOOGLE_*` variable names are also supported. See [config.go](./config.go) for the full `Config` struct.

## API Overview

| Method | Description |
| :--- | :--- |
| `NewClient(ctx, config)` | Create a client from explicit configuration |
| `NewClientFromEnv(ctx)` | Create a client using `DefaultConfig()` |
| `QuerySearchAnalytics(ctx, request)` | Run a custom search analytics query |
| `QuerySearchAnalyticsLastDays(ctx, days, dimensions, rowLimit)` | Convenience query for recent data |
| `InspectURL(ctx, request)` | Inspect a URL's indexing status |
| `ListSites(ctx)` / `GetSite(ctx, siteURL)` | List or get site properties |
| `AddSite(ctx, siteURL)` / `RemoveSite(ctx, siteURL)` | Manage site properties |
| `ListSitemaps(ctx, siteURL)` | List sitemaps for a property |
| `SubmitSitemap(ctx, siteURL, feedPath)` / `DeleteSitemap(ctx, siteURL, feedPath)` | Manage sitemaps |

## Example

```bash
export GSC_SERVICE_ACCOUNT_KEY_FILE=/path/to/key.json
export GSC_SITE_URL=https://example.com/

go run ./examples/basic/
```

## Testing

```bash
cd gsc
go test ./...
```
