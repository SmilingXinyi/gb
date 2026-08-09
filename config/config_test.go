package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SmilingXinyi/gb/config"
)

type serverConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type appConfig struct {
	Name    string         `yaml:"name"`
	Debug   bool           `yaml:"debug"`
	Timeout config.Duration `yaml:"timeout"`
	Server  serverConfig   `yaml:"server"`
	Tags    []string       `yaml:"tags"`
	Nested  *serverConfig  `yaml:"nested"`
}

// TestLoad_BasicYAML loads a simple YAML file into a typed struct.
func TestLoad_BasicYAML(t *testing.T) {
	path := writeFile(t, "config.yaml", `
name: demo
debug: true
timeout: 5s
server:
  host: 127.0.0.1
  port: 8080
tags:
  - a
  - b
`)
	var settings appConfig
	require.NoError(t, config.Load(path, &settings))
	assert.Equal(t, "demo", settings.Name)
	assert.True(t, settings.Debug)
	assert.Equal(t, 5*time.Second, settings.Timeout.Value())
	assert.Equal(t, "127.0.0.1", settings.Server.Host)
	assert.Equal(t, 8080, settings.Server.Port)
	assert.Equal(t, []string{"a", "b"}, settings.Tags)
}

// TestLoad_EnvOverride applies _SECTION__FIELD overrides after YAML decode.
func TestLoad_EnvOverride(t *testing.T) {
	path := writeFile(t, "config.yaml", `
name: demo
server:
  host: 127.0.0.1
  port: 8080
`)
	t.Setenv("_NAME", "from-env")
	t.Setenv("_SERVER__PORT", "9090")
	t.Setenv("_SERVER__HOST", "0.0.0.0")

	var settings appConfig
	require.NoError(t, config.Load(path, &settings))
	assert.Equal(t, "from-env", settings.Name)
	assert.Equal(t, "0.0.0.0", settings.Server.Host)
	assert.Equal(t, 9090, settings.Server.Port)
}

// TestLoad_UnknownTopLevelEnvIsIgnored skips env keys that do not match YAML fields.
func TestLoad_UnknownTopLevelEnvIsIgnored(t *testing.T) {
	path := writeFile(t, "config.yaml", `
name: demo
`)
	t.Setenv("_DOES_NOT_EXIST", "x")
	var settings appConfig
	require.NoError(t, config.Load(path, &settings))
	assert.Equal(t, "demo", settings.Name)
}

// TestLoad_UnknownNestedEnvReturnsError fails when a nested path does not exist.
func TestLoad_UnknownNestedEnvReturnsError(t *testing.T) {
	path := writeFile(t, "config.yaml", `
name: demo
server:
  host: 127.0.0.1
  port: 8080
`)
	t.Setenv("_SERVER__MISSING", "1")
	var settings appConfig
	err := config.Load(path, &settings)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown configuration field")
}

// TestLoad_KnownFieldsRejectsUnknownYAMLKeys rejects undeclared YAML keys.
func TestLoad_KnownFieldsRejectsUnknownYAMLKeys(t *testing.T) {
	path := writeFile(t, "config.yaml", `
name: demo
extra: true
`)
	var settings appConfig
	err := config.Load(path, &settings)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode config file")
}

// TestLoad_MultipleDocumentsRejected rejects multi-document YAML files.
func TestLoad_MultipleDocumentsRejected(t *testing.T) {
	path := writeFile(t, "config.yaml", `
name: first
---
name: second
`)
	var settings appConfig
	err := config.Load(path, &settings)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple YAML documents")
}

// TestLoad_WithDotEnv loads required dotenv values into the process environment.
func TestLoad_WithDotEnv(t *testing.T) {
	directory := t.TempDir()
	configPath := filepath.Join(directory, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("name: demo\nserver:\n  host: h\n  port: 1\n"), 0o644))
	dotenvPath := filepath.Join(directory, ".env")
	require.NoError(t, os.WriteFile(dotenvPath, []byte("_NAME=from-dotenv\n_SERVER__PORT=7777\n"), 0o644))

	clearOverrideEnv(t)

	var settings appConfig
	require.NoError(t, config.Load(configPath, &settings, config.WithDotEnv(dotenvPath)))
	assert.Equal(t, "from-dotenv", settings.Name)
	assert.Equal(t, 7777, settings.Server.Port)
}

// TestLoad_WithOptionalDotEnvIgnoresMissingFile skips missing optional dotenv files.
func TestLoad_WithOptionalDotEnvIgnoresMissingFile(t *testing.T) {
	clearOverrideEnv(t)
	path := writeFile(t, "config.yaml", "name: demo\n")
	var settings appConfig
	require.NoError(t, config.Load(path, &settings, config.WithOptionalDotEnv(filepath.Join(t.TempDir(), "missing.env"))))
	assert.Equal(t, "demo", settings.Name)
}

// TestLoad_WithDotEnvMissingFileErrors fails when a required dotenv file is absent.
func TestLoad_WithDotEnvMissingFileErrors(t *testing.T) {
	path := writeFile(t, "config.yaml", "name: demo\n")
	var settings appConfig
	err := config.Load(path, &settings, config.WithDotEnv(filepath.Join(t.TempDir(), "missing.env")))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load dotenv file")
}

// TestLoad_NilDestinationErrors validates destination pointer requirements.
func TestLoad_NilDestinationErrors(t *testing.T) {
	path := writeFile(t, "config.yaml", "name: demo\n")
	err := config.Load(path, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "destination is required")
}

// TestApplyEnv_AllocatesNilPointerNestedFields creates missing nested pointer objects.
func TestApplyEnv_AllocatesNilPointerNestedFields(t *testing.T) {
	settings := &appConfig{}
	err := config.ApplyEnv(settings, []string{
		"_NESTED__HOST=allocated",
		"_NESTED__PORT=1234",
	})
	require.NoError(t, err)
	require.NotNil(t, settings.Nested)
	assert.Equal(t, "allocated", settings.Nested.Host)
	assert.Equal(t, 1234, settings.Nested.Port)
}

// TestApplyEnv_SliceViaYAMLLiteral decodes complex values through YAML unmarshal.
func TestApplyEnv_SliceViaYAMLLiteral(t *testing.T) {
	settings := &appConfig{Name: "demo"}
	err := config.ApplyEnv(settings, []string{`_TAGS=["x","y"]`})
	require.NoError(t, err)
	assert.Equal(t, []string{"x", "y"}, settings.Tags)
}

// TestApplyEnv_BoolAndNumbers parses scalar overrides.
func TestApplyEnv_BoolAndNumbers(t *testing.T) {
	settings := &appConfig{}
	err := config.ApplyEnv(settings, []string{
		"_DEBUG=true",
		"_SERVER__PORT=42",
	})
	require.NoError(t, err)
	assert.True(t, settings.Debug)
	assert.Equal(t, 42, settings.Server.Port)
}

// TestApplyEnv_DurationUsesTextUnmarshaler parses duration overrides.
func TestApplyEnv_DurationUsesTextUnmarshaler(t *testing.T) {
	settings := &appConfig{}
	err := config.ApplyEnv(settings, []string{"_TIMEOUT=1500ms"})
	require.NoError(t, err)
	assert.Equal(t, 1500*time.Millisecond, settings.Timeout.Value())
}

// TestEnvironmentPathHelpers covers invalid override names via ApplyEnv no-ops.
func TestEnvironmentPathHelpers(t *testing.T) {
	settings := &appConfig{Name: "keep"}
	err := config.ApplyEnv(settings, []string{
		"NAME=ignored",     // missing leading underscore
		"__NAME=ignored",   // leading double underscore
		"_SER-VER=ignored", // invalid segment character; also no top-level match path
		"NOEQUALS",         // malformed
	})
	require.NoError(t, err)
	assert.Equal(t, "keep", settings.Name)
}

// writeFile creates a temporary file with the given contents and returns its path.
func writeFile(t *testing.T, name, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o644))
	return path
}

// clearOverrideEnv removes process env keys used by dotenv/env override tests.
func clearOverrideEnv(t *testing.T) {
	t.Helper()
	keys := []string{"_NAME", "_SERVER__PORT", "_SERVER__HOST", "_DEBUG", "_TIMEOUT", "_TAGS", "_NESTED__HOST", "_NESTED__PORT", "_SERVER__MISSING", "_DOES_NOT_EXIST"}
	for _, key := range keys {
		require.NoError(t, os.Unsetenv(key))
	}
	t.Cleanup(func() {
		for _, key := range keys {
			_ = os.Unsetenv(key)
		}
	})
}
