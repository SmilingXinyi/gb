// Package gsc provides a Go client for the Google Search Console API.
// It supports search analytics, URL inspection, sitemap management, and site listing.
// Configure credentials via service account JSON (file or inline) or Application Default Credentials.
package gsc

import (
	"context"
	"fmt"

	"github.com/SmilingXinyi/gb/log"
	"google.golang.org/api/option"
	searchconsole "google.golang.org/api/searchconsole/v1"
)

var moduleLogger = log.Module("gsc")

// Client wraps the official Search Console API service with a simplified API.
type Client struct {
	config  Config
	service *searchconsole.Service
}

// NewClient creates a Client from Config using service account or ADC authentication.
func NewClient(ctx context.Context, config Config) (*Client, error) {
	serviceOptions, err := buildClientOptions(config)
	if err != nil {
		return nil, err
	}
	if config.ReadOnly {
		serviceOptions = append(serviceOptions, option.WithScopes(searchconsole.WebmastersReadonlyScope))
	}

	service, err := searchconsole.NewService(ctx, serviceOptions...)
	if err != nil {
		moduleLogger.Error().Err(err).Msg("failed to create search console service")
		return nil, fmt.Errorf("gsc: create service: %w", err)
	}
	moduleLogger.Debug().Bool("readonly", config.ReadOnly).Msg("search console client initialized")
	return &Client{config: config, service: service}, nil
}

// NewClientFromEnv creates a Client using DefaultConfig().
func NewClientFromEnv(ctx context.Context) (*Client, error) {
	return NewClient(ctx, DefaultConfig())
}

// Config returns a copy of the client configuration.
func (client *Client) Config() Config {
	return client.config
}

// DefaultSiteURL returns the configured default site URL, or empty if unset.
func (client *Client) DefaultSiteURL() string {
	return client.config.SiteURL
}

func buildClientOptions(config Config) ([]option.ClientOption, error) {
	switch {
	case config.ServiceAccountKeyFile != "":
		return []option.ClientOption{
			option.WithAuthCredentialsFile(option.ServiceAccount, config.ServiceAccountKeyFile),
		}, nil
	case config.ServiceAccountKeyJSON != "":
		return []option.ClientOption{
			option.WithAuthCredentialsJSON(option.ServiceAccount, []byte(config.ServiceAccountKeyJSON)),
		}, nil
	default:
		return nil, nil
	}
}

func resolveSiteURL(client *Client, siteURL string) (string, error) {
	if siteURL != "" {
		return siteURL, nil
	}
	if client.config.SiteURL != "" {
		return client.config.SiteURL, nil
	}
	return "", fmt.Errorf("gsc: site URL is required (set GSC_SITE_URL or pass siteURL)")
}
