package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRun_Help prints usage and returns nil for help flags.
func TestRun_Help(t *testing.T) {
	for _, arguments := range [][]string{{"help"}, {"-h"}, {"--help"}} {
		err := run(arguments)
		require.NoError(t, err, "arguments=%v", arguments)
	}
}

// TestRun_Version prints the embedded version string.
func TestRun_Version(t *testing.T) {
	require.NoError(t, run([]string{"version"}))
}

// TestRun_MissingCommand returns an error when no command is provided.
func TestRun_MissingCommand(t *testing.T) {
	err := run(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing command")
}

// TestRun_UnknownCommand returns an error for unsupported commands.
func TestRun_UnknownCommand(t *testing.T) {
	err := run([]string{"delete-all"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}

// TestDetectContentType maps common extensions to MIME types.
func TestDetectContentType(t *testing.T) {
	assert.Equal(t, "text/plain", detectContentType("notes.txt"))
	assert.Equal(t, "application/json", detectContentType("data.JSON"))
	assert.Equal(t, "image/png", detectContentType("/tmp/a.png"))
	assert.Equal(t, "application/octet-stream", detectContentType("blob.bin"))
}

// TestOpenClient_RequiresProvider validates required client settings.
func TestOpenClient_RequiresProvider(t *testing.T) {
	_, err := openClient(&globalFlags{
		accessKey: "ak",
		secretKey: "sk",
		bucket:    "bucket",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider")
}

// TestSplitLeadingFlagsAndCommand keeps root flags before the subcommand.
func TestSplitLeadingFlagsAndCommand(t *testing.T) {
	leadingFlags, command, commandArguments, err := splitLeadingFlagsAndCommand([]string{
		"-provider", "baidu",
		"-bucket", "demo",
		"put",
		"key",
		"file.txt",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"-provider", "baidu", "-bucket", "demo"}, leadingFlags)
	assert.Equal(t, "put", command)
	assert.Equal(t, []string{"key", "file.txt"}, commandArguments)
}

// TestRunPut_UsageError fails fast when positional args are incomplete.
func TestRunPut_UsageError(t *testing.T) {
	err := runPut([]string{"-provider", "baidu", "-access-key", "ak", "-secret-key", "sk", "-bucket", "b", "only-key"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "usage: gb-oss put")
}

// TestPrintUsage_WritesHelp ensures usage text mentions core commands.
func TestPrintUsage_WritesHelp(t *testing.T) {
	buffer := &bytes.Buffer{}
	printUsage(buffer)
	output := buffer.String()
	assert.Contains(t, output, "put <key> <file>")
	assert.Contains(t, output, "sign-url <key>")
	assert.Contains(t, output, "-provider")
	assert.Contains(t, output, "cloudflare")
	assert.Contains(t, output, "-account-id")
	assert.Contains(t, output, "-no-progress")
}

// TestEnvOr_UsesFallback returns the fallback when the env var is unset.
func TestEnvOr_UsesFallback(t *testing.T) {
	name := "OSS_CLI_TEST_ENV_OR"
	require.NoError(t, os.Unsetenv(name))
	assert.Equal(t, "fallback", envOr(name, "fallback"))

	t.Setenv(name, "from-env")
	assert.Equal(t, "from-env", envOr(name, "fallback"))
}

// TestDetectContentType_FromTempFile keeps extension detection stable for real paths.
func TestDetectContentType_FromTempFile(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "readme.md")
	require.NoError(t, os.WriteFile(path, []byte("# hi"), 0o644))
	assert.Equal(t, "application/octet-stream", detectContentType(path))
}
