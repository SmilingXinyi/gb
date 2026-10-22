// Command jaeger sends a parent span and a failing child span to a local Jaeger.
package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/SmilingXinyi/gb/sseq"
)

// main exports one trace to Jaeger's OTLP/HTTP endpoint.
func main() {
	err := sseq.SetupOTLP("http://localhost:4318/v1/traces", "demo",
		sseq.WithResourceAttributes(map[string]any{
			"deployment.environment": "dev",
		}),
	)
	if err != nil {
		panic(err)
	}
	defer sseq.Shutdown()

	err = sseq.Trace(context.Background(), "HTTP POST /checkout", "server", func(ctx context.Context) error {
		sseq.Set(ctx, "http.route", "/checkout")
		sseq.Event(ctx, "cart.loaded", "item.count", 2)
		return sseq.Trace(ctx, "Charge card", "client", func(ctx context.Context) error {
			sseq.Set(ctx, "payment.provider", "example")
			time.Sleep(20 * time.Millisecond)
			return errors.New("card declined")
		})
	})
	if err != nil {
		fmt.Println("trace sent to Jaeger:", err)
		return
	}
	fmt.Println("trace sent to Jaeger")
}
