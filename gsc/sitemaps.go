package gsc

import (
	"context"
	"fmt"
)

// SitemapEntry describes a sitemap registered in Search Console.
type SitemapEntry struct {
	Path         string `json:"path"`
	LastSubmitted string `json:"lastSubmitted,omitempty"`
	IsPending    bool   `json:"isPending,omitempty"`
	IsSitemapsIndex bool `json:"isSitemapsIndex,omitempty"`
	Type         string `json:"type,omitempty"`
	LastDownloaded string `json:"lastDownloaded,omitempty"`
	Warnings     int64  `json:"warnings,omitempty"`
	Errors       int64  `json:"errors,omitempty"`
}

// ListSitemaps returns all sitemaps submitted for a site.
func (client *Client) ListSitemaps(ctx context.Context, siteURL string) ([]SitemapEntry, error) {
	resolvedURL, err := resolveSiteURL(client, siteURL)
	if err != nil {
		return nil, err
	}

	moduleLogger.Debug().Str("site", resolvedURL).Msg("listing sitemaps")

	response, err := client.service.Sitemaps.List(resolvedURL).Context(ctx).Do()
	if err != nil {
		moduleLogger.Error().Err(err).Str("site", resolvedURL).Msg("list sitemaps failed")
		return nil, fmt.Errorf("gsc: list sitemaps: %w", err)
	}

	entries := make([]SitemapEntry, 0, len(response.Sitemap))
	for _, sitemap := range response.Sitemap {
		if sitemap == nil {
			continue
		}
		entries = append(entries, SitemapEntry{
			Path:            sitemap.Path,
			LastSubmitted:   sitemap.LastSubmitted,
			IsPending:       sitemap.IsPending,
			IsSitemapsIndex: sitemap.IsSitemapsIndex,
			Type:            sitemap.Type,
			LastDownloaded:  sitemap.LastDownloaded,
			Warnings:        sitemap.Warnings,
			Errors:          sitemap.Errors,
		})
	}
	return entries, nil
}

// SubmitSitemap registers a sitemap URL for a site. Requires full (non-readonly) scope.
func (client *Client) SubmitSitemap(ctx context.Context, siteURL, feedPath string) error {
	resolvedURL, err := resolveSiteURL(client, siteURL)
	if err != nil {
		return err
	}
	if feedPath == "" {
		return fmt.Errorf("gsc: sitemap feed path is required")
	}

	moduleLogger.Debug().Str("site", resolvedURL).Str("feed", feedPath).Msg("submitting sitemap")
	if err := client.service.Sitemaps.Submit(resolvedURL, feedPath).Context(ctx).Do(); err != nil {
		moduleLogger.Error().Err(err).Str("site", resolvedURL).Str("feed", feedPath).Msg("submit sitemap failed")
		return fmt.Errorf("gsc: submit sitemap: %w", err)
	}
	return nil
}

// DeleteSitemap removes a sitemap from Search Console. Requires full scope.
func (client *Client) DeleteSitemap(ctx context.Context, siteURL, feedPath string) error {
	resolvedURL, err := resolveSiteURL(client, siteURL)
	if err != nil {
		return err
	}
	if feedPath == "" {
		return fmt.Errorf("gsc: sitemap feed path is required")
	}

	moduleLogger.Debug().Str("site", resolvedURL).Str("feed", feedPath).Msg("deleting sitemap")
	if err := client.service.Sitemaps.Delete(resolvedURL, feedPath).Context(ctx).Do(); err != nil {
		moduleLogger.Error().Err(err).Str("site", resolvedURL).Str("feed", feedPath).Msg("delete sitemap failed")
		return fmt.Errorf("gsc: delete sitemap: %w", err)
	}
	return nil
}
