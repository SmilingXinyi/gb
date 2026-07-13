package main

import (
	"context"
	"fmt"
	"time"

	"github.com/SmilingXinyi/gb/sseq"
)

func main() {
	if err := sseq.SetupSeq("http://localhost:5342/ingest/clef", "", "demo"); err != nil {
		panic(err)
	}
	defer sseq.Shutdown()

	err := sseq.Trace(context.Background(), "HTTP GET /api/users", "server", func(ctx context.Context) error {
		sseq.Set(ctx, "http.method", "GET")
		sseq.Set(ctx, "http.route", "/api/users")

		if err := sseq.Trace(ctx, "Authenticate user", "internal", func(context.Context) error {
			time.Sleep(20 * time.Millisecond)
			return nil
		}); err != nil {
			return err
		}

		return sseq.Trace(ctx, "Query users table", "", func(ctx context.Context) error {
			sseq.Set(ctx, "db.system", "postgres")
			sseq.Set(ctx, "db.operation", "SELECT")
			time.Sleep(30 * time.Millisecond)
			return nil
		})
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("span tree sent to Seq")
}
