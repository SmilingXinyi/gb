// Package main shows how to use the oss interface with Cloudflare R2.
// Run: go run ./examples/cloudflare/
//
// Set environment variables or place a .env file in the working directory:
//
//	OSS_CLOUDFLARE_ACCESS_KEY=your-access-key-id
//	OSS_CLOUDFLARE_SECRET_KEY=your-secret-access-key
//	OSS_CLOUDFLARE_ACCOUNT_ID=your-account-id
//	OSS_CLOUDFLARE_BUCKET=your-bucket
package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/SmilingXinyi/gb/oss"
	_ "github.com/SmilingXinyi/gb/oss/cloudflare"
)

func main() {
	_ = godotenv.Load(".env")

	config := oss.Config{
		AccessKey: os.Getenv("OSS_CLOUDFLARE_ACCESS_KEY"),
		SecretKey: os.Getenv("OSS_CLOUDFLARE_SECRET_KEY"),
		AccountID: os.Getenv("OSS_CLOUDFLARE_ACCOUNT_ID"),
		Region:    os.Getenv("OSS_CLOUDFLARE_REGION"),
		Endpoint:  os.Getenv("OSS_CLOUDFLARE_ENDPOINT"),
		Bucket:    os.Getenv("OSS_CLOUDFLARE_BUCKET"),
	}
	if config.AccessKey == "" || config.SecretKey == "" || config.Bucket == "" {
		log.Fatal("set OSS_CLOUDFLARE_ACCESS_KEY, OSS_CLOUDFLARE_SECRET_KEY, and OSS_CLOUDFLARE_BUCKET")
	}
	if config.AccountID == "" && config.Endpoint == "" {
		log.Fatal("set OSS_CLOUDFLARE_ACCOUNT_ID or OSS_CLOUDFLARE_ENDPOINT")
	}

	client, err := oss.New(oss.ProviderCloudflare, config)
	if err != nil {
		log.Fatal("init client:", err)
	}

	ctx := context.Background()
	bucket := config.Bucket
	objectKey := "examples/hello.txt"
	content := []byte("hello, cloudflare r2!")

	err = client.Put(ctx, bucket, objectKey, bytes.NewReader(content), int64(len(content)), &oss.PutOptions{
		ContentType: "text/plain",
		Metadata:    map[string]string{"demo": "true"},
	})
	mustNil("put", err)
	fmt.Println("put:", objectKey)

	meta, err := client.Stat(ctx, bucket, objectKey)
	mustNil("stat", err)
	fmt.Printf("stat: size=%d contentType=%s etag=%s lastModified=%s\n",
		meta.Size, meta.ContentType, meta.ETag, meta.LastModified.Format("2006-01-02 15:04:05"))

	readCloser, err := client.Get(ctx, bucket, objectKey)
	mustNil("get", err)
	defer readCloser.Close()
	data, _ := io.ReadAll(readCloser)
	fmt.Printf("get: content=%q\n", data)

	listResult, err := client.List(ctx, bucket, "examples/", &oss.ListOptions{
		Delimiter: "/",
		MaxKeys:   100,
	})
	mustNil("list", err)
	fmt.Printf("list: %d objects, isTruncated=%v\n", len(listResult.Objects), listResult.IsTruncated)
	for _, object := range listResult.Objects {
		fmt.Printf("    - %s  %d bytes\n", object.Key, object.Size)
	}

	signedURL, err := client.SignURL(ctx, bucket, objectKey, "GET", 3600)
	mustNil("sign url", err)
	fmt.Println("signed url:", signedURL)

	copyKey := "examples/hello_copy.txt"
	err = client.Copy(ctx, bucket, objectKey, bucket, copyKey)
	mustNil("copy", err)
	fmt.Println("copy:", copyKey)

	for _, deleteKey := range []string{objectKey, copyKey} {
		mustNil("delete "+deleteKey, client.Delete(ctx, bucket, deleteKey))
		fmt.Println("delete:", deleteKey)
	}
}

// mustNil stops the example when an operation returns an error.
func mustNil(operation string, err error) {
	if err != nil {
		log.Fatalf("%s failed: %v", operation, err)
	}
}
