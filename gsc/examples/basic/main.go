// Basic example: list Search Console sites and query last-7-day search analytics.
// Set GSC_SERVICE_ACCOUNT_KEY_FILE and GSC_SITE_URL before running.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/SmilingXinyi/gb/gsc"
	"github.com/SmilingXinyi/gb/log"
)

func main() {
	log.Setup(log.DefaultConfig())

	ctx := context.Background()
	client, err := gsc.NewClientFromEnv(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create client: %v\n", err)
		os.Exit(1)
	}

	sites, err := client.ListSites(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "list sites: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Sites:")
	printJSON(sites)

	analytics, err := client.QuerySearchAnalyticsLastDays(ctx, 7, []string{"query"}, 10)
	if err != nil {
		fmt.Fprintf(os.Stderr, "search analytics: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Search analytics (last 7 days, top queries):")
	printJSON(analytics)
}

func printJSON(value any) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "encode json: %v\n", err)
		return
	}
	fmt.Println(string(encoded))
}
