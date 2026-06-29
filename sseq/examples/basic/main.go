package main

import (
	"context"
	"fmt"
	"time"

	"github.com/SmilingXinyi/gb/sseq"
)

func main() {
	sseq.Setup(sseq.DefaultConfig())
	defer sseq.Shutdown()

	err := sseq.Do(context.Background(), "HTTP GET /api/users", func(ctx context.Context) error {
		if err := sseq.Do(ctx, "Authenticate user", func(ctx context.Context) error {
			time.Sleep(20 * time.Millisecond)
			return nil
		}); err != nil {
			return err
		}

		return sseq.Do(ctx, "Query users table", func(ctx context.Context) error {
			time.Sleep(30 * time.Millisecond)
			return nil
		})
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("span tree sent to Seq")
}
