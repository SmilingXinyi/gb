package gsc

import (
	"context"
	"fmt"

	searchconsole "google.golang.org/api/searchconsole/v1"
)

// SearchAnalyticsRow is a single row from a search analytics query.
type SearchAnalyticsRow struct {
	Keys        []string  `json:"keys"`
	Clicks      float64   `json:"clicks"`
	Impressions float64   `json:"impressions"`
	CTR         float64   `json:"ctr"`
	Position    float64   `json:"position"`
}

// SearchAnalyticsResult holds aggregated search performance data.
type SearchAnalyticsResult struct {
	ResponseAggregationType string               `json:"responseAggregationType,omitempty"`
	Rows                    []SearchAnalyticsRow `json:"rows,omitempty"`
}

// QuerySearchAnalytics runs a Search Analytics query for the given site and date range.
func (client *Client) QuerySearchAnalytics(ctx context.Context, request SearchAnalyticsRequest) (*SearchAnalyticsResult, error) {
	siteURL, err := resolveSiteURL(client, request.SiteURL)
	if err != nil {
		return nil, err
	}
	if request.StartDate == "" || request.EndDate == "" {
		return nil, fmt.Errorf("gsc: start and end dates are required (YYYY-MM-DD)")
	}

	apiRequest := &searchconsole.SearchAnalyticsQueryRequest{
		StartDate:  request.StartDate,
		EndDate:    request.EndDate,
		Dimensions: request.Dimensions,
		RowLimit:   request.RowLimit,
		StartRow:   request.StartRow,
	}
	if request.SearchType != "" {
		apiRequest.SearchType = request.SearchType
	}
	if request.Aggregation != "" {
		apiRequest.AggregationType = request.Aggregation
	}
	if apiRequest.RowLimit == 0 {
		apiRequest.RowLimit = 100
	}

	moduleLogger.Debug().
		Str("site", siteURL).
		Str("start", request.StartDate).
		Str("end", request.EndDate).
		Msg("querying search analytics")

	response, err := client.service.Searchanalytics.Query(siteURL, apiRequest).Context(ctx).Do()
	if err != nil {
		moduleLogger.Error().Err(err).Str("site", siteURL).Msg("search analytics query failed")
		return nil, fmt.Errorf("gsc: query search analytics: %w", err)
	}

	result := &SearchAnalyticsResult{ResponseAggregationType: response.ResponseAggregationType}
	for _, row := range response.Rows {
		if row == nil {
			continue
		}
		result.Rows = append(result.Rows, SearchAnalyticsRow{
			Keys:        row.Keys,
			Clicks:      row.Clicks,
			Impressions: row.Impressions,
			CTR:         row.Ctr,
			Position:    row.Position,
		})
	}
	return result, nil
}

// QuerySearchAnalyticsLastDays queries search analytics for the last N days using DefaultSiteURL when set.
func (client *Client) QuerySearchAnalyticsLastDays(ctx context.Context, days int, dimensions []string, rowLimit int64) (*SearchAnalyticsResult, error) {
	startDate, endDate := DateRange(days)
	return client.QuerySearchAnalytics(ctx, SearchAnalyticsRequest{
		StartDate:  startDate,
		EndDate:    endDate,
		Dimensions: dimensions,
		RowLimit:   rowLimit,
	})
}
