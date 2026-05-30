package gsc

import (
	"context"
	"fmt"
)

// SiteEntry describes a Search Console property and permission level.
type SiteEntry struct {
	SiteURL        string `json:"siteUrl"`
	PermissionLevel string `json:"permissionLevel"`
}

// ListSites returns all Search Console properties accessible to the authenticated account.
func (client *Client) ListSites(ctx context.Context) ([]SiteEntry, error) {
	moduleLogger.Debug().Msg("listing search console sites")

	response, err := client.service.Sites.List().Context(ctx).Do()
	if err != nil {
		moduleLogger.Error().Err(err).Msg("list sites failed")
		return nil, fmt.Errorf("gsc: list sites: %w", err)
	}

	entries := make([]SiteEntry, 0, len(response.SiteEntry))
	for _, entry := range response.SiteEntry {
		if entry == nil {
			continue
		}
		entries = append(entries, SiteEntry{
			SiteURL:         entry.SiteUrl,
			PermissionLevel: entry.PermissionLevel,
		})
	}
	return entries, nil
}

// GetSite returns metadata for a single Search Console property.
func (client *Client) GetSite(ctx context.Context, siteURL string) (*SiteEntry, error) {
	resolvedURL, err := resolveSiteURL(client, siteURL)
	if err != nil {
		return nil, err
	}

	moduleLogger.Debug().Str("site", resolvedURL).Msg("getting site")

	entry, err := client.service.Sites.Get(resolvedURL).Context(ctx).Do()
	if err != nil {
		moduleLogger.Error().Err(err).Str("site", resolvedURL).Msg("get site failed")
		return nil, fmt.Errorf("gsc: get site: %w", err)
	}
	return &SiteEntry{
		SiteURL:         entry.SiteUrl,
		PermissionLevel: entry.PermissionLevel,
	}, nil
}

// AddSite adds a site to the user's Search Console account. Requires full (non-readonly) scope.
func (client *Client) AddSite(ctx context.Context, siteURL string) error {
	resolvedURL, err := resolveSiteURL(client, siteURL)
	if err != nil {
		return err
	}
	moduleLogger.Debug().Str("site", resolvedURL).Msg("adding site")
	if err := client.service.Sites.Add(resolvedURL).Context(ctx).Do(); err != nil {
		moduleLogger.Error().Err(err).Str("site", resolvedURL).Msg("add site failed")
		return fmt.Errorf("gsc: add site: %w", err)
	}
	return nil
}

// RemoveSite removes a site from the user's Search Console account. Requires full scope.
func (client *Client) RemoveSite(ctx context.Context, siteURL string) error {
	resolvedURL, err := resolveSiteURL(client, siteURL)
	if err != nil {
		return err
	}
	moduleLogger.Debug().Str("site", resolvedURL).Msg("removing site")
	if err := client.service.Sites.Delete(resolvedURL).Context(ctx).Do(); err != nil {
		moduleLogger.Error().Err(err).Str("site", resolvedURL).Msg("remove site failed")
		return fmt.Errorf("gsc: remove site: %w", err)
	}
	return nil
}
