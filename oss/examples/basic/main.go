package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/SmilingXinyi/gb/oss"
	_ "github.com/SmilingXinyi/gb/oss/baidu"
)

func main() {
	// Initialize OSS client for Baidu BOS
	// In real usage, replace with your actual credentials
	config := oss.Config{
		Region:    "bj",
		AccessKey: "your-access-key",
		SecretKey: "your-secret-key",
		Bucket:    "your-bucket",
	}

	client, err := oss.New(oss.ProviderBaidu, config)
	if err != nil {
		log.Fatalf("Failed to create OSS client: %v", err)
	}

	ctx := context.Background()
	bucket := "your-bucket"
	key := "test/hello.txt"
	content := "Hello, OSS!"

	// Upload
	err = client.Put(ctx, bucket, key, strings.NewReader(content), int64(len(content)), nil)
	if err != nil {
		fmt.Printf("Upload failed (expected if credentials are invalid): %v\n", err)
	} else {
		fmt.Println("Upload successful")
	}

	// Stat
	meta, err := client.Stat(ctx, bucket, key)
	if err != nil {
		fmt.Printf("Stat failed: %v\n", err)
	} else {
		fmt.Printf("Object size: %d, ContentType: %s\n", meta.Size, meta.ContentType)
	}

	// List
	result, err := client.List(ctx, bucket, "test/", &oss.ListOptions{MaxKeys: 10})
	if err != nil {
		fmt.Printf("List failed: %v\n", err)
	} else {
		fmt.Printf("Found %d objects\n", len(result.Objects))
		for _, obj := range result.Objects {
			fmt.Printf(" - %s (%d bytes)\n", obj.Key, obj.Size)
		}
	}
}
