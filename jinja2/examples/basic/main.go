package main

import (
	"fmt"
	"os"

	"github.com/SmilingXinyi/gb/jinja2"
)

func main() {
	config := jinja2.DefaultConfig()
	config.TrimBlocks = true
	engine := jinja2.New(config)

	source := "Hello {{ name | capitalize }}!\n{% for item in items %}- {{ item }}\n{% endfor %}"
	rendered, err := engine.Render(source, map[string]any{
		"name":  "bob",
		"items": []string{"alpha", "beta"},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "render failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(rendered)
}
