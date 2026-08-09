// Command gb-oss is a small CLI for the unified oss Storage interface.
// It supports put, get, stat, list, and sign-url against registered providers.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SmilingXinyi/gb/oss"
	_ "github.com/SmilingXinyi/gb/oss/baidu"
	_ "github.com/SmilingXinyi/gb/oss/tencent"
)

const (
	defaultSignExpireSeconds = 3600
	defaultListMaxKeys       = 1000
)

// version is injected at build time via -ldflags "-X main.version=...".
var version = "dev"

// globalFlags holds shared credentials and bucket settings for all subcommands.
type globalFlags struct {
	provider  string
	accessKey string
	secretKey string
	region    string
	endpoint  string
	bucket    string
	token     string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// run parses arguments and dispatches a subcommand.
func run(arguments []string) error {
	if len(arguments) == 0 {
		printUsage(os.Stderr)
		return fmt.Errorf("missing command")
	}

	leadingFlags, command, commandArguments, err := splitLeadingFlagsAndCommand(arguments)
	if err != nil {
		return err
	}
	if command == "" || command == "-h" || command == "--help" || command == "help" {
		printUsage(os.Stdout)
		return nil
	}

	// Allow global flags before the subcommand:
	//   gb-oss -provider baidu put key file
	// by prepending them to the subcommand flag set.
	mergedArguments := append(append([]string{}, leadingFlags...), commandArguments...)

	switch command {
	case "put":
		return runPut(mergedArguments)
	case "get":
		return runGet(mergedArguments)
	case "stat":
		return runStat(mergedArguments)
	case "list":
		return runList(mergedArguments)
	case "sign-url":
		return runSignURL(mergedArguments)
	case "version":
		fmt.Println(version)
		return nil
	default:
		printUsage(os.Stderr)
		return fmt.Errorf("unknown command %q", command)
	}
}

// splitLeadingFlagsAndCommand separates root-level flags from the subcommand token.
func splitLeadingFlagsAndCommand(arguments []string) (leadingFlags []string, command string, commandArguments []string, err error) {
	flagsWithValue := map[string]struct{}{
		"-provider": {}, "--provider": {},
		"-access-key": {}, "--access-key": {},
		"-secret-key": {}, "--secret-key": {},
		"-region": {}, "--region": {},
		"-endpoint": {}, "--endpoint": {},
		"-bucket": {}, "--bucket": {},
		"-token": {}, "--token": {},
	}

	index := 0
	for index < len(arguments) {
		argument := arguments[index]
		if argument == "--" {
			index++
			break
		}
		if !strings.HasPrefix(argument, "-") {
			break
		}
		if strings.Contains(argument, "=") {
			leadingFlags = append(leadingFlags, argument)
			index++
			continue
		}
		if _, ok := flagsWithValue[argument]; ok {
			if index+1 >= len(arguments) {
				return nil, "", nil, fmt.Errorf("flag %s requires a value", argument)
			}
			leadingFlags = append(leadingFlags, argument, arguments[index+1])
			index += 2
			continue
		}
		// Unknown root flag (e.g. -h) — treat as the command token for help handling.
		break
	}

	if index >= len(arguments) {
		return leadingFlags, "", nil, nil
	}
	command = arguments[index]
	commandArguments = append([]string{}, arguments[index+1:]...)
	return leadingFlags, command, commandArguments, nil
}

// printUsage writes CLI help text.
func printUsage(writer io.Writer) {
	fmt.Fprintf(writer, `gb-oss %s — unified object storage CLI

Usage:
  gb-oss <command> [flags] [arguments]

Commands:
  put <key> <file>       Upload a local file to the object key
  get <key> [file]       Download an object (write to file or stdout)
  stat <key>             Print object metadata
  list [prefix]          List objects under an optional prefix
  sign-url <key>         Generate a pre-signed URL
  version                Print CLI version

Global flags (also via env OSS_PROVIDER, OSS_ACCESS_KEY, OSS_SECRET_KEY,
OSS_REGION, OSS_ENDPOINT, OSS_BUCKET, OSS_TOKEN):
  -provider string       Provider name: baidu | tencent
  -access-key string     Access key / SecretId
  -secret-key string     Secret key / SecretKey
  -region string         Region (e.g. bj, ap-guangzhou)
  -endpoint string       Optional endpoint override
  -bucket string         Default bucket name
  -token string          Optional STS session token

Command-specific flags:
  put:
    -content-type string MIME type for the uploaded object
  list:
    -delimiter string    Delimiter for directory-style listing (e.g. /)
    -max-keys int        Max keys per page (default 1000)
  sign-url:
    -method string       HTTP method for the signed URL (default GET)
    -expire int          Expiration in seconds (default 3600)

Examples:
  gb-oss -provider baidu -bucket my-bucket put docs/a.txt ./a.txt
  gb-oss -provider tencent -bucket my-bucket-1250000000 get docs/a.txt
  gb-oss -provider baidu list docs/ -delimiter /
  gb-oss -provider tencent sign-url docs/a.txt -expire 600
`, version)
}

// newFlagSet creates a FlagSet that already includes shared global flags.
func newFlagSet(name string) (*flag.FlagSet, *globalFlags) {
	flagSet := flag.NewFlagSet(name, flag.ContinueOnError)
	flagSet.SetOutput(os.Stderr)

	settings := &globalFlags{}
	flagSet.StringVar(&settings.provider, "provider", envOr("OSS_PROVIDER", ""), "provider name")
	flagSet.StringVar(&settings.accessKey, "access-key", envOr("OSS_ACCESS_KEY", ""), "access key")
	flagSet.StringVar(&settings.secretKey, "secret-key", envOr("OSS_SECRET_KEY", ""), "secret key")
	flagSet.StringVar(&settings.region, "region", envOr("OSS_REGION", ""), "region")
	flagSet.StringVar(&settings.endpoint, "endpoint", envOr("OSS_ENDPOINT", ""), "endpoint")
	flagSet.StringVar(&settings.bucket, "bucket", envOr("OSS_BUCKET", ""), "bucket")
	flagSet.StringVar(&settings.token, "token", envOr("OSS_TOKEN", ""), "STS token")
	return flagSet, settings
}

// envOr returns the environment value when set, otherwise fallback.
func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

// openClient builds an oss.Storage client from global flags.
func openClient(settings *globalFlags) (oss.Storage, error) {
	if settings.provider == "" {
		return nil, fmt.Errorf("provider is required (-provider or OSS_PROVIDER)")
	}
	if settings.accessKey == "" {
		return nil, fmt.Errorf("access-key is required (-access-key or OSS_ACCESS_KEY)")
	}
	if settings.secretKey == "" {
		return nil, fmt.Errorf("secret-key is required (-secret-key or OSS_SECRET_KEY)")
	}
	if settings.bucket == "" {
		return nil, fmt.Errorf("bucket is required (-bucket or OSS_BUCKET)")
	}

	config := oss.Config{
		AccessKey: settings.accessKey,
		SecretKey: settings.secretKey,
		Region:    settings.region,
		Endpoint:  settings.endpoint,
		Bucket:    settings.bucket,
		Token:     settings.token,
	}
	return oss.New(settings.provider, config)
}

// runPut uploads a local file to the given object key.
func runPut(arguments []string) error {
	flagSet, settings := newFlagSet("put")
	contentType := flagSet.String("content-type", "", "object Content-Type")
	if err := flagSet.Parse(arguments); err != nil {
		return err
	}
	positionals := flagSet.Args()
	if len(positionals) != 2 {
		return fmt.Errorf("usage: gb-oss put [flags] <key> <file>")
	}
	objectKey := positionals[0]
	localPath := positionals[1]

	client, err := openClient(settings)
	if err != nil {
		return err
	}

	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", localPath, err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat %s: %w", localPath, err)
	}

	options := &oss.PutOptions{ContentType: *contentType}
	if options.ContentType == "" {
		options.ContentType = detectContentType(localPath)
	}

	ctx := context.Background()
	if err := client.Put(ctx, settings.bucket, objectKey, file, fileInfo.Size(), options); err != nil {
		return err
	}
	fmt.Printf("put ok: bucket=%s key=%s size=%d\n", settings.bucket, objectKey, fileInfo.Size())
	return nil
}

// runGet downloads an object to a local file or stdout.
func runGet(arguments []string) error {
	flagSet, settings := newFlagSet("get")
	if err := flagSet.Parse(arguments); err != nil {
		return err
	}
	positionals := flagSet.Args()
	if len(positionals) < 1 || len(positionals) > 2 {
		return fmt.Errorf("usage: gb-oss get [flags] <key> [file]")
	}
	objectKey := positionals[0]

	client, err := openClient(settings)
	if err != nil {
		return err
	}

	ctx := context.Background()
	body, err := client.Get(ctx, settings.bucket, objectKey)
	if err != nil {
		return err
	}
	defer body.Close()

	if len(positionals) == 1 {
		_, copyErr := io.Copy(os.Stdout, body)
		return copyErr
	}

	outputPath := positionals[1]
	parentDir := filepath.Dir(outputPath)
	if parentDir != "." && parentDir != "" {
		if err := os.MkdirAll(parentDir, 0o755); err != nil {
			return fmt.Errorf("create parent dir: %w", err)
		}
	}
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create %s: %w", outputPath, err)
	}
	defer outputFile.Close()

	written, err := io.Copy(outputFile, body)
	if err != nil {
		return err
	}
	fmt.Printf("get ok: bucket=%s key=%s -> %s (%d bytes)\n", settings.bucket, objectKey, outputPath, written)
	return nil
}

// runStat prints object metadata.
func runStat(arguments []string) error {
	flagSet, settings := newFlagSet("stat")
	if err := flagSet.Parse(arguments); err != nil {
		return err
	}
	positionals := flagSet.Args()
	if len(positionals) != 1 {
		return fmt.Errorf("usage: gb-oss stat [flags] <key>")
	}
	objectKey := positionals[0]

	client, err := openClient(settings)
	if err != nil {
		return err
	}

	ctx := context.Background()
	meta, err := client.Stat(ctx, settings.bucket, objectKey)
	if err != nil {
		return err
	}

	fmt.Printf("key=%s\n", meta.Key)
	fmt.Printf("size=%d\n", meta.Size)
	fmt.Printf("content-type=%s\n", meta.ContentType)
	fmt.Printf("etag=%s\n", meta.ETag)
	fmt.Printf("last-modified=%s\n", meta.LastModified.UTC().Format(time.RFC3339))
	fmt.Printf("storage-class=%s\n", meta.StorageClass)
	for metaKey, metaValue := range meta.Metadata {
		fmt.Printf("meta.%s=%s\n", metaKey, metaValue)
	}
	return nil
}

// runList lists objects under an optional prefix.
func runList(arguments []string) error {
	flagSet, settings := newFlagSet("list")
	delimiter := flagSet.String("delimiter", "", "list delimiter")
	maxKeys := flagSet.Int("max-keys", defaultListMaxKeys, "max keys per page")
	if err := flagSet.Parse(arguments); err != nil {
		return err
	}
	positionals := flagSet.Args()
	prefix := ""
	if len(positionals) > 1 {
		return fmt.Errorf("usage: gb-oss list [flags] [prefix]")
	}
	if len(positionals) == 1 {
		prefix = positionals[0]
	}

	client, err := openClient(settings)
	if err != nil {
		return err
	}

	ctx := context.Background()
	result, err := client.List(ctx, settings.bucket, prefix, &oss.ListOptions{
		Delimiter: *delimiter,
		MaxKeys:   *maxKeys,
	})
	if err != nil {
		return err
	}

	fmt.Printf("prefix=%q truncated=%v next-token=%q\n", prefix, result.IsTruncated, result.NextToken)
	for _, object := range result.Objects {
		fmt.Printf("object\t%s\t%d\t%s\t%s\n",
			object.Key,
			object.Size,
			object.ETag,
			object.LastModified.UTC().Format(time.RFC3339),
		)
	}
	for _, commonPrefix := range result.CommonPrefixes {
		fmt.Printf("prefix\t%s\n", commonPrefix)
	}
	return nil
}

// runSignURL generates a pre-signed object URL.
func runSignURL(arguments []string) error {
	flagSet, settings := newFlagSet("sign-url")
	method := flagSet.String("method", "GET", "HTTP method for the signed URL")
	expireSeconds := flagSet.Int64("expire", defaultSignExpireSeconds, "expiration in seconds")
	if err := flagSet.Parse(arguments); err != nil {
		return err
	}
	positionals := flagSet.Args()
	if len(positionals) != 1 {
		return fmt.Errorf("usage: gb-oss sign-url [flags] <key>")
	}
	objectKey := positionals[0]

	client, err := openClient(settings)
	if err != nil {
		return err
	}

	ctx := context.Background()
	signedURL, err := client.SignURL(ctx, settings.bucket, objectKey, strings.ToUpper(*method), *expireSeconds)
	if err != nil {
		return err
	}
	fmt.Println(signedURL)
	return nil
}

// detectContentType guesses a MIME type from the file extension.
func detectContentType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".txt":
		return "text/plain"
	case ".json":
		return "application/json"
	case ".html", ".htm":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".pdf":
		return "application/pdf"
	case ".zip":
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}
