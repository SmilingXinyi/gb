package gsc

import (
	"context"
	"fmt"

	searchconsole "google.golang.org/api/searchconsole/v1"
)

// URLInspectionRequest defines parameters for URL index inspection.
type URLInspectionRequest struct {
	SiteURL       string
	InspectionURL string
	LanguageCode  string
}

// URLInspectionResult holds the outcome of a URL inspection request.
type URLInspectionResult struct {
	InspectionResult *searchconsole.UrlInspectionResult `json:"inspectionResult,omitempty"`
}

// InspectURL checks the Google index status of a URL via the URL Inspection API.
func (client *Client) InspectURL(ctx context.Context, request URLInspectionRequest) (*URLInspectionResult, error) {
	siteURL, err := resolveSiteURL(client, request.SiteURL)
	if err != nil {
		return nil, err
	}
	if request.InspectionURL == "" {
		return nil, fmt.Errorf("gsc: inspection URL is required")
	}

	apiRequest := &searchconsole.InspectUrlIndexRequest{
		InspectionUrl: request.InspectionURL,
		SiteUrl:       siteURL,
		LanguageCode:  request.LanguageCode,
	}

	moduleLogger.Debug().
		Str("site", siteURL).
		Str("url", request.InspectionURL).
		Msg("inspecting URL")

	response, err := client.service.UrlInspection.Index.Inspect(apiRequest).Context(ctx).Do()
	if err != nil {
		moduleLogger.Error().Err(err).Str("url", request.InspectionURL).Msg("URL inspection failed")
		return nil, fmt.Errorf("gsc: inspect URL: %w", err)
	}

	return &URLInspectionResult{InspectionResult: response.InspectionResult}, nil
}
