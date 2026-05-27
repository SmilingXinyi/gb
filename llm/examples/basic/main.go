package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/SmilingXinyi/gb/llm"
)

func main() {
	config := llm.DefaultConfig()
	if config.APIKey == "" {
		fmt.Println("Set LLM_API_KEY before running this example.")
		os.Exit(1)
	}

	client, err := llm.NewClient(config)
	if err != nil {
		panic(err)
	}

	reply, err := client.Chat(context.Background(), []llm.Message{
		llm.SystemMessage("You are a concise assistant."),
		llm.UserMessage("Reply with one short greeting."),
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("Chat reply:", reply)

	structureDescription := `{
		"topic": "string",
		"confidence": "number"
	}`

	extracted, err := client.Extract(
		context.Background(),
		"Classify the greeting above.",
		structureDescription,
	)
	if err != nil {
		panic(err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(extracted), &parsed); err != nil {
		panic(err)
	}
	fmt.Println("Extracted JSON:", parsed)
}
