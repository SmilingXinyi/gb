package sseq

import (
	"fmt"
	"os"
	"strings"
)

// validateConfig checks that the config selects exactly one provider with required fields.
func validateConfig(config Config) error {
	provider := resolveProvider(config)
	if provider == "" {
		return fmt.Errorf("sseq: no provider configured")
	}

	if config.Provider == "" && countConfiguredProviders(config) > 1 {
		return fmt.Errorf("sseq: ambiguous provider configuration; set Provider explicitly")
	}

	switch provider {
	case ProviderHTTP:
		if resolveHTTPEndpoint(config) == "" {
			return fmt.Errorf("sseq: http provider requires endpoint")
		}
	case ProviderFile:
		if strings.TrimSpace(config.File.Filename) == "" {
			return fmt.Errorf("sseq: file provider requires filename")
		}
	case ProviderAxiom:
		if strings.TrimSpace(config.Axiom.Token) == "" || strings.TrimSpace(config.Axiom.Dataset) == "" {
			return fmt.Errorf("sseq: axiom provider requires token and dataset")
		}
	default:
		return fmt.Errorf("sseq: unknown provider %q", provider)
	}

	return nil
}

// warnIgnoredProviderConfigs logs when non-selected provider settings are present.
func warnIgnoredProviderConfigs(config Config) {
	if config.Provider == "" {
		return
	}

	selected := config.Provider
	if selected != ProviderHTTP && isHTTPConfigured(config) {
		fmt.Fprintf(os.Stderr, "sseq: Provider=%s; http config will be ignored\n", selected)
	}
	if selected != ProviderFile && isFileConfigured(config) {
		fmt.Fprintf(os.Stderr, "sseq: Provider=%s; file config will be ignored\n", selected)
	}
	if selected != ProviderAxiom && isAxiomConfigured(config) {
		fmt.Fprintf(os.Stderr, "sseq: Provider=%s; axiom config will be ignored\n", selected)
	}
}

func countConfiguredProviders(config Config) int {
	count := 0
	if isHTTPConfigured(config) {
		count++
	}
	if isFileConfigured(config) {
		count++
	}
	if isAxiomConfigured(config) {
		count++
	}
	return count
}

func isHTTPConfigured(config Config) bool {
	return resolveHTTPEndpoint(config) != ""
}

func isFileConfigured(config Config) bool {
	return strings.TrimSpace(config.File.Filename) != ""
}

func isAxiomConfigured(config Config) bool {
	return strings.TrimSpace(config.Axiom.Token) != "" && strings.TrimSpace(config.Axiom.Dataset) != ""
}

func resolveHTTPEndpoint(config Config) string {
	if config.HTTP.Endpoint != "" {
		return config.HTTP.Endpoint
	}
	return config.Endpoint
}
