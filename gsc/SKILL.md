---
name: google-search-console
description: Google Search Console API integration for search analytics, URL inspection, sitemap management, and site listing. Use when working with search performance data, checking indexing status, managing sitemaps, or analyzing SEO metrics.
---

# Google Search Console (gsc module)

Go client in `github.com/SmilingXinyi/gb/gsc` wrapping the official [Google Search Console API](https://developers.google.com/webmaster-tools).

## When to Use

- User mentions "Google Search Console", "GSC", "search console"
- Search performance: clicks, impressions, CTR, position
- URL indexing status or inspection
- Sitemap management
- Listing verified properties

## Setup

1. Enable **Google Search Console API** in [Google Cloud Console](https://console.cloud.google.com/apis/library/searchconsole.googleapis.com).
2. Create a **service account** and download the JSON key.
3. Add the service account email as a user in [Search Console](https://search.google.com/search-console) → Settings → Users and permissions.

### Environment Variables

| Variable | Description |
|----------|-------------|
| `GSC_SERVICE_ACCOUNT_KEY_FILE` | Path to service account JSON key |
| `GSC_SERVICE_ACCOUNT_KEY_JSON` | Inline service account JSON (CI/CD) |
| `GSC_SITE_URL` | Default property URL (e.g. `https://example.com/` or `sc-domain:example.com`) |
| `GSC_READONLY` | Set to `true` for read-only scope |

Legacy names `GOOGLE_SERVICE_ACCOUNT_KEY_FILE`, `GOOGLE_SEARCH_CONSOLE_SITE_URL` are also supported.

## API Usage

```go
import (
    "context"
    "github.com/SmilingXinyi/gb/gsc"
)

ctx := context.Background()
client, err := gsc.NewClientFromEnv(ctx)

// List properties
sites, err := client.ListSites(ctx)

// Search analytics (last 7 days, by query)
result, err := client.QuerySearchAnalyticsLastDays(ctx, 7, []string{"query"}, 25)

// Custom date range
result, err := client.QuerySearchAnalytics(ctx, gsc.SearchAnalyticsRequest{
    StartDate:  "2026-01-01",
    EndDate:    "2026-01-31",
    Dimensions: []string{"page", "query"},
    RowLimit:   100,
})

// URL inspection
inspection, err := client.InspectURL(ctx, gsc.URLInspectionRequest{
    InspectionURL: "https://example.com/page",
})

// Sitemaps
sitemaps, err := client.ListSitemaps(ctx, "")
err = client.SubmitSitemap(ctx, "", "https://example.com/sitemap.xml")
```

## Capabilities

| Feature | Methods |
|---------|---------|
| Search analytics | `QuerySearchAnalytics`, `QuerySearchAnalyticsLastDays` |
| Sites | `ListSites`, `GetSite`, `AddSite`, `RemoveSite` |
| Sitemaps | `ListSitemaps`, `SubmitSitemap`, `DeleteSitemap` |
| URL inspection | `InspectURL` |

## Example

```bash
export GSC_SERVICE_ACCOUNT_KEY_FILE=/path/to/key.json
export GSC_SITE_URL=https://example.com/
go run ./gsc/examples/basic/
```

## References

- [Search Console API](https://developers.google.com/webmaster-tools)
- [Search Analytics query](https://developers.google.com/webmaster-tools/v1/searchanalytics/query)
- [URL Inspection API](https://developers.google.com/webmaster-tools/v1/urlInspection.index/inspect)
