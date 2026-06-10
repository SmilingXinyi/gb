# oss

统一对象存储抽象层，通过 `Storage` 接口屏蔽各云存储 SDK 差异，支持按 provider 切换而不修改业务代码。

## 目录结构

```
oss/
├── go.mod              # 模块定义
├── oss.go              # Provider 枚举、Storage 接口
├── config.go           # Config 配置结构体
├── object.go           # 通用数据结构（PutOptions、ObjectMeta、ListResult 等）
├── errors.go           # 通用错误类型（ErrObjectNotFound、ErrInvalidConfig 等）
├── client.go           # New() 工厂函数 + Register() 注册机制
├── baidu/              # 百度云 BOS 适配器（已实现）
├── aliyun/             # 阿里云 OSS 适配器（骨架）
├── tencent/            # 腾讯云 COS 适配器（已实现）
├── s3/                 # AWS S3 / S3 兼容存储适配器（骨架）
├── examples/
│   ├── basic/          # 基础用法示例
│   ├── baidu/          # 百度云可执行示例
│   └── tencent/        # 腾讯云可执行示例
```

## 快速上手

```go
import (
    "github.com/SmilingXinyi/gb/oss"
    _ "github.com/SmilingXinyi/gb/oss/baidu" // 注册百度云 provider
)

// 方式 1：使用 oss.Provider 类型（推荐，类型安全）
client, err := oss.New(oss.ProviderBaidu, oss.Config{
    AccessKey: "your-ak",
    SecretKey: "your-sk",
    Region:    "bj",
    Bucket:    "my-bucket",
})

// 方式 2：使用 string 类型（更灵活）
// client, err := oss.New("baidu", oss.Config{...})

// 上传
err = client.Put(ctx, "", "path/to/file.txt", reader, size, nil)

// 下载
rc, err := client.Get(ctx, "", "path/to/file.txt")
defer rc.Close()

// 元信息
meta, err := client.Stat(ctx, "", "path/to/file.txt")

// 列举
result, err := client.List(ctx, "", "prefix/", &oss.ListOptions{Delimiter: "/"})

// 预签名 URL
url, err := client.SignURL(ctx, "", "path/to/file.txt", "GET", 3600)

// 服务端复制
err = client.Copy(ctx, "src-bucket", "src/key", "dst-bucket", "dst/key")

// 删除
err = client.Delete(ctx, "", "path/to/file.txt")
```

切换 provider 只需替换 import 和 `oss.New` 的第一个参数，其余调用代码不变。

## 测试

### 百度云 BOS

**单元测试（无需凭证）**

```bash
go test ./oss/baidu/ -run "TestNewClient"
```

**集成测试**

```bash
# 1. 复制模板并填入真实凭证
cp oss/baidu/.env.example oss/baidu/.env

# 2. 运行全部集成测试
go test ./oss/baidu/ -v -run "TestIntegration"

# 3. 运行指定测试项
go test ./oss/baidu/ -v -run "TestIntegration_PutAndGet"

# 4. 单元测试 + 集成测试一并运行
go test ./oss/baidu/ -v
```

`.env` 格式（参考 `oss/baidu/.env.example`）：

```dotenv
BAIDU_OSS_AK=your-access-key
BAIDU_OSS_SK=your-secret-key
BAIDU_OSS_REGION=bj
BAIDU_OSS_BUCKET=your-bucket-name
```

**示例程序**

```bash
go run ./oss/examples/baidu/
```

---

### 阿里云 OSS

> 适配器尚未实现，测试命令待补充。

---

### 腾讯云 COS

**单元测试（无需凭证）**

```bash
go test ./oss/tencent/ -v -run "TestIsNotFound"
```

**集成测试**

```bash
# 1. 复制模板并填入真实凭证
cp oss/tencent/.env.example oss/tencent/.env

# 2. 运行全部集成测试
go test ./oss/tencent/ -v -run "TestIntegration"
```

**示例程序**

```bash
go run ./oss/examples/tencent/
```

---

### AWS S3 / S3 兼容存储

> 适配器尚未实现，测试命令待补充。

---

> **注意**：各 provider 的 `.env` 文件需放置在其对应的适配器目录下：
> - 百度云 BOS：`oss/baidu/.env`
> - 阿里云 OSS：`oss/aliyun/.env`（待实现）
> - 腾讯云 COS：`oss/tencent/.env`
> - AWS S3：`oss/s3/.env`（待实现）
>
> 未配置 `.env` 时，集成测试自动 skip，单元测试正常运行。

## Provider 配置说明

### 百度云 BOS

| Config 字段 | 说明 | 示例 |
|---|---|---|
| `AccessKey` | BOS AK | `AKXXXXXXXXXX` |
| `SecretKey` | BOS SK | `SKXXXXXXXXXX` |
| `Region` | 地域前缀，推导 Endpoint | `bj`、`gz`、`su` |
| `Endpoint` | 直接指定接入点（优先于 Region） | `bj.bcebos.com` |
| `Bucket` | 默认 bucket（可选） | `my-bucket` |
| `Token` | STS 临时 Token（可选） | — |

Region 与 Endpoint 映射关系（留空默认 `bj`）：

| Region | Endpoint |
|---|---|
| `bj` | `bj.bcebos.com`（北京） |
| `gz` | `gz.bcebos.com`（广州） |
| `su` | `su.bcebos.com`（苏州） |
| `hkg` | `hkg.bcebos.com`（香港） |
| `fwh` | `fwh.bcebos.com`（武汉） |

### 腾讯云 COS

| Config 字段 | 说明 | 示例 |
|---|---|---|
| `AccessKey` | SecretId | `AKIDXXXXXXXXXX` |
| `SecretKey` | SecretKey | `XXXXXXXXXX` |
| `Region` | 地域 | `ap-guangzhou`、`ap-shanghai` |
| `Endpoint` | 直接指定 BucketURL（格式 `https://{bucket}.cos.{region}.myqcloud.com`） | — |
| `Bucket` | 默认 bucket（格式 `{name}-{appid}`） | `my-bucket-1250000000` |
| `Token` | SessionToken（可选） | — |
