// Package main 演示如何使用 oss 统一接口操作腾讯云 COS 对象存储。
// 运行：go run ./oss/examples/tencent/
//
// 需设置环境变量或在当前目录放置 .env 文件：
//
//	OSS_TENCENT_ACCESS_KEY=your-secret-id
//	OSS_TENCENT_SECRET_KEY=your-secret-key
//	OSS_TENCENT_REGION=ap-guangzhou
//	OSS_TENCENT_BUCKET=your-bucket-name-1234567890
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
	_ "github.com/SmilingXinyi/gb/oss/tencent" // 注册腾讯云 provider
)

func main() {
	_ = godotenv.Load(".env")

	config := oss.Config{
		AccessKey: os.Getenv("OSS_TENCENT_ACCESS_KEY"),
		SecretKey: os.Getenv("OSS_TENCENT_SECRET_KEY"),
		Region:    os.Getenv("OSS_TENCENT_REGION"),
		Bucket:    os.Getenv("OSS_TENCENT_BUCKET"),
	}
	if config.AccessKey == "" || config.Bucket == "" {
		log.Fatal("请设置 OSS_TENCENT_ACCESS_KEY / OSS_TENCENT_SECRET_KEY / OSS_TENCENT_BUCKET 环境变量")
	}

	// 1. 创建客户端（通过 provider 工厂，不直接依赖 tencent 包）
	client, err := oss.New(oss.ProviderTencent, config)
	if err != nil {
		log.Fatal("init client:", err)
	}

	ctx := context.Background()
	bucket := config.Bucket
	key := "examples/hello.txt"
	content := []byte("hello, tencent cos!")

	// 2. 上传对象
	err = client.Put(ctx, bucket, key, bytes.NewReader(content), int64(len(content)), &oss.PutOptions{
		ContentType: "text/plain",
		Metadata:    map[string]string{"demo": "true"},
	})
	mustNil("put", err)
	fmt.Println("✓ put:", key)

	// 3. 获取元信息
	meta, err := client.Stat(ctx, bucket, key)
	mustNil("stat", err)
	fmt.Printf("✓ stat: size=%d contentType=%s etag=%s lastModified=%s\n",
		meta.Size, meta.ContentType, meta.ETag, meta.LastModified.Format("2006-01-02 15:04:05"))

	// 4. 下载对象
	readCloser, err := client.Get(ctx, bucket, key)
	mustNil("get", err)
	defer readCloser.Close()
	data, _ := io.ReadAll(readCloser)
	fmt.Printf("✓ get: content=%q\n", data)

	// 5. 列举对象
	listResult, err := client.List(ctx, bucket, "examples/", &oss.ListOptions{
		Delimiter: "/",
		MaxKeys:   100,
	})
	mustNil("list", err)
	fmt.Printf("✓ list: %d objects, isTruncated=%v\n", len(listResult.Objects), listResult.IsTruncated)
	for _, object := range listResult.Objects {
		fmt.Printf("    - %s  %d bytes\n", object.Key, object.Size)
	}

	// 6. 生成预签名下载 URL（有效期 3600 秒）
	signedURL, err := client.SignURL(ctx, bucket, key, "GET", 3600)
	mustNil("sign url", err)
	fmt.Println("✓ signed url:", signedURL)

	// 7. 服务端复制
	copyKey := "examples/hello_copy.txt"
	err = client.Copy(ctx, bucket, key, bucket, copyKey)
	mustNil("copy", err)
	fmt.Println("✓ copy →", copyKey)

	// 8. 删除对象（清理）
	for _, deleteKey := range []string{key, copyKey} {
		mustNil("delete "+deleteKey, client.Delete(ctx, bucket, deleteKey))
		fmt.Println("✓ delete:", deleteKey)
	}
}

func mustNil(op string, err error) {
	if err != nil {
		log.Fatalf("%s failed: %v", op, err)
	}
}
