package sseq

import "testing"

func TestValidateConfigRequiresProviderFields(t *testing.T) {
	testCases := []struct {
		name   string
		config Config
	}{
		{
			name:   "empty config",
			config: Config{},
		},
		{
			name: "http without endpoint",
			config: Config{
				Provider: ProviderHTTP,
			},
		},
		{
			name: "file without filename",
			config: Config{
				Provider: ProviderFile,
			},
		},
		{
			name: "axiom without token",
			config: Config{
				Provider: ProviderAxiom,
				Axiom: AxiomConfig{
					Dataset: "traces",
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if err := validateConfig(testCase.config); err == nil {
				t.Fatal("expected validateConfig() error")
			}
		})
	}
}

func TestValidateConfigRejectsAmbiguousAutoDetect(t *testing.T) {
	config := Config{
		Endpoint: "http://localhost:5342/ingest/clef",
		File: FileConfig{
			Filename: "spans.clef",
		},
	}

	if err := validateConfig(config); err == nil {
		t.Fatal("expected ambiguous provider error")
	}
}

func TestSetupReturnsErrorForInvalidConfig(t *testing.T) {
	if err := Setup(Config{}); err == nil {
		t.Fatal("expected Setup() error for empty config")
	}
}

func TestSetupSucceedsForHTTPConfig(t *testing.T) {
	if err := Setup(Config{
		Provider:      ProviderHTTP,
		Endpoint:      "http://localhost:5342/ingest/clef",
		BatchSize:     1,
		FlushInterval: 0,
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	t.Cleanup(Shutdown)
}
