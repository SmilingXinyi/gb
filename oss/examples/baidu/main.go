// Package main 演示如何使用 oss 统一接口操作百度云 BOS 对象存储。
// 运行：go run ./oss/examples/baidu/
//
// 需设置环境变量或在当前目录放置 .env 文件：
//
//	BAIDU_OSS_AK=your-ak
//	BAIDU_OSS_SK=your-sk
//	BAIDU_OSS_REGION=bj
//	BAIDU_OSS_BUCKET=your-bucket
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
	_ "github.com/SmilingXinyi/gb/oss/baidu" // 注册百度云 provider
)

func main() {
	_ = godotenv.Load(".env")

	cfg := oss.Config{
		AccessKey: os.Getenv("BAIDU_OSS_AK"),
		SecretKey: os.Getenv("BAIDU_OSS_SK"),
		Region:    os.Getenv("BAIDU_OSS_REGION"),
		Bucket:    os.Getenv("BAIDU_OSS_BUCKET"),
	}
	if cfg.AccessKey == "" || cfg.Bucket == "" {
		log.Fatal("请设置 BAIDU_OSS_AK / BAIDU_OSS_SK / BAIDU_OSS_BUCKET 环境变量")
	}

	// 1. 创建客户端（通过 provider 工厂，不直接依赖 baidu 包）
	client, err := oss.New(oss.ProviderBaidu, cfg)
	if err != nil {
		log.Fatal("init client:", err)
	}

	ctx := context.Background()
	bucket := cfg.Bucket
	key := "examples/hello.txt"
	content := []byte("hello, baidu oss!")

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
	rc, err := client.Get(ctx, bucket, key)
	mustNil("get", err)
	defer rc.Close()
	data, _ := io.ReadAll(rc)
	fmt.Printf("✓ get: content=%q\n", data)

	// 5. 列举对象
	result, err := client.List(ctx, bucket, "examples/", &oss.ListOptions{
		Delimiter: "/",
		MaxKeys:   100,
	})
	mustNil("list", err)
	fmt.Printf("✓ list: %d objects, isTruncated=%v\n", len(result.Objects), result.IsTruncated)
	for _, obj := range result.Objects {
		fmt.Printf("    - %s  %d bytes\n", obj.Key, obj.Size)
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
	for _, k := range []string{key, copyKey} {
		mustNil("delete "+k, client.Delete(ctx, bucket, k))
		fmt.Println("✓ delete:", k)
	}
}

func mustNil(op string, err error) {
	if err != nil {
		log.Fatalf("%s failed: %v", op, err)
	}
}
