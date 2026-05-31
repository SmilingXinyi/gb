package gsc

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	t.Setenv(envServiceAccountKeyFile, "/tmp/key.json")
	t.Setenv(envSiteURL, "https://example.com/")
	t.Setenv(envReadOnly, "true")

	config := DefaultConfig()
	if config.ServiceAccountKeyFile != "/tmp/key.json" {
		t.Fatalf("ServiceAccountKeyFile = %q, want /tmp/key.json", config.ServiceAccountKeyFile)
	}
	if config.SiteURL != "https://example.com/" {
		t.Fatalf("SiteURL = %q, want https://example.com/", config.SiteURL)
	}
	if !config.ReadOnly {
		t.Fatal("expected ReadOnly to be true")
	}
}

func TestDefaultConfigLegacyEnv(t *testing.T) {
	t.Setenv(envServiceAccountKeyFile, "")
	t.Setenv(legacyEnvServiceAccountKeyFile, "/legacy/key.json")
	t.Setenv(legacyEnvSiteURL, "sc-domain:example.com")

	config := DefaultConfig()
	if config.ServiceAccountKeyFile != "/legacy/key.json" {
		t.Fatalf("legacy key file = %q", config.ServiceAccountKeyFile)
	}
	if config.SiteURL != "sc-domain:example.com" {
		t.Fatalf("legacy site URL = %q", config.SiteURL)
	}
}

func TestDateRange(t *testing.T) {
	startDate, endDate := DateRange(7)
	endParsed, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		t.Fatalf("parse end date: %v", err)
	}
	startParsed, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		t.Fatalf("parse start date: %v", err)
	}
	diff := endParsed.Sub(startParsed)
	if diff < 6*24*time.Hour || diff > 7*24*time.Hour {
		t.Fatalf("expected ~6 day span, got %v", diff)
	}
}

func TestResolveSiteURL(t *testing.T) {
	client := &Client{config: Config{SiteURL: "https://default.com/"}}

	siteURL, err := resolveSiteURL(client, "https://override.com/")
	if err != nil || siteURL != "https://override.com/" {
		t.Fatalf("override: got %q, err %v", siteURL, err)
	}

	siteURL, err = resolveSiteURL(client, "")
	if err != nil || siteURL != "https://default.com/" {
		t.Fatalf("default: got %q, err %v", siteURL, err)
	}

	emptyClient := &Client{}
	_, err = resolveSiteURL(emptyClient, "")
	if err == nil {
		t.Fatal("expected error when site URL missing")
	}
}

func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "a", "b"); got != "a" {
		t.Fatalf("firstNonEmpty = %q", got)
	}
	if got := firstNonEmpty("", "", ""); got != "" {
		t.Fatalf("firstNonEmpty empty = %q", got)
	}
}

func TestDefaultConfigFromEnvironment(t *testing.T) {
	_ = os.Getenv
	config := DefaultConfig()
	if config.ServiceAccountKeyFile != "" && config.ServiceAccountKeyJSON != "" {
		t.Fatal("both key file and JSON should not be set from empty env in isolated test")
	}
	_ = config
}
