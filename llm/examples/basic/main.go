package main

import (
	"context"
	"fmt"
	"os"

	"github.com/SmilingXinyi/gb/llm"
	"github.com/SmilingXinyi/gb/log"
)

type TopicResult struct {
	Topic      string  `json:"topic"`
	Confidence float64 `json:"confidence"`
}

func main() {
	log.Setup(log.DefaultConfig())

	config := llm.DefaultConfig()
	if config.APIKey == "" {
		fmt.Println("Set LLM_API_KEY before running this example.")
		os.Exit(1)
	}

	session := llm.NewSessionFromConfig(config).
		WithSystem("You are a concise assistant.").
		AddUser("Reply with one short greeting.")

	reply, err := session.Execute(context.Background())
	if err != nil {
		panic(err)
	}
	fmt.Println("Chat reply:", reply)

	structuredSession := llm.NewSessionFromConfig(config).
		WithSystem("Classify the user message.").
		AddUser(reply)

	result, err := llm.ExecuteTo[TopicResult](context.Background(), structuredSession)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Structured: topic=%s confidence=%.2f\n", result.Topic, result.Confidence)
}
