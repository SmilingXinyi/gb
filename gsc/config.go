package gsc

import (
	"os"
	"time"
)

const (
	envServiceAccountKeyFile = "GSC_SERVICE_ACCOUNT_KEY_FILE"
	envServiceAccountKeyJSON = "GSC_SERVICE_ACCOUNT_KEY_JSON"
	envSiteURL               = "GSC_SITE_URL"
	envReadOnly              = "GSC_READONLY"

	legacyEnvServiceAccountKeyFile = "GOOGLE_SERVICE_ACCOUNT_KEY_FILE"
	legacyEnvServiceAccountKeyJSON = "GOOGLE_SERVICE_ACCOUNT_KEY_JSON"
	legacyEnvSiteURL               = "GOOGLE_SEARCH_CONSOLE_SITE_URL"
)

// Config holds Google Search Console client settings.
type Config struct {
	// ServiceAccountKeyFile is the path to a service account JSON key file.
	ServiceAccountKeyFile string
	// ServiceAccountKeyJSON is inline service account JSON (for CI/CD).
	ServiceAccountKeyJSON string
	// SiteURL is the default Search Console property (e.g. https://example.com/ or sc-domain:example.com).
	SiteURL string
	// ReadOnly restricts OAuth scopes to read-only when true.
	ReadOnly bool
}

// DefaultConfig loads settings from environment variables.
// Supports GSC_* and legacy GOOGLE_* variable names.
func DefaultConfig() Config {
	config := Config{
		ServiceAccountKeyFile: firstNonEmpty(
			os.Getenv(envServiceAccountKeyFile),
			os.Getenv(legacyEnvServiceAccountKeyFile),
		),
		ServiceAccountKeyJSON: firstNonEmpty(
			os.Getenv(envServiceAccountKeyJSON),
			os.Getenv(legacyEnvServiceAccountKeyJSON),
		),
		SiteURL: firstNonEmpty(
			os.Getenv(envSiteURL),
			os.Getenv(legacyEnvSiteURL),
			os.Getenv("GSC_SITE_URL"),
			os.Getenv("SEARCH_CONSOLE_SITE"),
		),
	}
	if readOnlyValue := os.Getenv(envReadOnly); readOnlyValue == "1" || readOnlyValue == "true" {
		config.ReadOnly = true
	}
	return config
}

// SearchAnalyticsRequest defines parameters for a search analytics query.
type SearchAnalyticsRequest struct {
	SiteURL      string
	StartDate    string
	EndDate      string
	Dimensions   []string
	SearchType   string
	RowLimit     int64
	StartRow     int64
	Aggregation  string
}

// DateRange returns start and end dates for the last N days (inclusive of today in UTC).
func DateRange(days int) (startDate string, endDate string) {
	if days < 1 {
		days = 1
	}
	end := time.Now().UTC()
	start := end.AddDate(0, 0, -(days - 1))
	return start.Format("2006-01-02"), end.Format("2006-01-02")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
