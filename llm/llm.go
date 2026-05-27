// Package llm provides goai-based large language model helpers.
// It supports fluent session configuration, structured output via GenerateObject,
// and streaming via StreamText. Under the hood, goai/provider/compat connects to
// any OpenAI-compatible API without depending on go-openai.
package llm

import (
	"github.com/SmilingXinyi/gb/log"
)

var moduleLogger = log.Module("llm")
