# Cloudflare DDNS Tool

[English](#english) | [中文](#中文)

---

## 中文

一个轻量级、高效的 Cloudflare DDNS 更新工具，使用 Go 语言编写。当您的 WAN IP 变化时，自动更新 DNS 记录。

### 特性

- 🚀 **多种 DNS 记录类型**：支持 A（IPv4）、AAAA（IPv6）以及同时更新两种记录
- 🔄 **自动 IP 检测**：通过 Cloudflare trace API 自动检测 WAN IP
- 🎯 **双操作模式**：创建/更新 DNS 记录或删除记录
- 🐳 **Docker 支持**：多架构 Docker 镜像（linux/amd64、linux/arm64）
- ⚡ **快速轻量**：由 Go 编译的单个二进制文件，依赖最少
- 🔒 **安全**：使用 Cloudflare 的 X-Auth-Email 和 X-Auth-Key 认证

### 快速开始

#### 使用 Docker

```bash
# 构建镜像
docker build -t cloudflareddns .

# 运行容器
docker run -v $(pwd)/config.json:/app/config.json cloudflareddns
```

#### 使用二进制文件

```bash
# 编译二进制文件
go build -o cloudflareddns main.go

# 运行
./cloudflareddns
```

### 配置说明

在二进制文件所在目录创建 `config.json` 文件：

```json
{
  "x_auth_email": "your_email@example.com",
  "x_auth_key": "your_cloudflare_api_key",
  "zone_name": "example.com",
  "record_name": "ddns.example.com",
  "type": "A",
  "ttl": 1,
  "ip": "auto",
  "proxy_status": false,
  "mode": "upsert"
}
```

#### 配置选项

| 字段 | 类型 | 必填 | 说明 |
|-------|------|------|------|
| `x_auth_email` | string | ✅ | 您的 Cloudflare 账户邮箱 |
| `x_auth_key` | string | ✅ | 您的 Cloudflare API 密钥（全局 API 密钥或源 CA 密钥） |
| `zone_name` | string | ✅ | 您的域名（如 `example.com`） |
| `record_name` | string | ✅ | 完整的 DNS 记录名称（如 `ddns.example.com`） |
| `type` | string | ❌ | 记录类型：`A`、`AAAA` 或 `A_AAAA`（默认：`A`） |
| `ttl` | int | ❌ | TTL 值：`1`（自动）或 `120`-`86400` 秒（默认：`1`） |
| `ip` | string | ❌ | IP 地址：`auto`（自动检测）、静态 IP 或 `ipv4,ipv6`（默认：`auto`） |
| `proxy_status` | bool | ❌ | 启用 Cloudflare 代理：`true` 或 `false`（默认：`false`） |
| `mode` | string | ❌ | 操作模式：`upsert`（创建/更新）或 `delete`（默认：`upsert`） |

#### 有效的 TTL 值

- `1` - 自动（Cloudflare 自动优化）
- `120` - 2 分钟
- `300` - 5 分钟
- `600` - 10 分钟
- `900` - 15 分钟
- `1800` - 30 分钟
- `3600` - 1 小时
- `7200` - 2 小时
- `18000` - 5 小时
- `43200` - 12 小时
- `86400` - 24 小时

#### 记录类型

- **A** - IPv4 地址记录
- **AAAA** - IPv6 地址记录
- **A_AAAA** - 同时创建 A 和 AAAA 记录

#### 操作模式

- **upsert** - 如果 DNS 记录不存在则创建，如果存在则更新
- **delete** - 删除 DNS 记录

#### IP 配置

- **auto** - 通过 Cloudflare trace API 自动检测您的 WAN IP（推荐）
- **static** - 使用指定的 IP 地址（如 `"192.168.1.1"`）
- **dual** - 同时指定 IPv4 和 IPv6（如 `"192.168.1.1,2001:db8::1"`）

### 使用示例

#### 更新 IPv4 A 记录

```json
{
  "x_auth_email": "user@example.com",
  "x_auth_key": "123456789abcdef",
  "zone_name": "example.com",
  "record_name": "home.example.com",
  "type": "A",
  "ttl": 1,
  "ip": "auto",
  "proxy_status": false,
  "mode": "upsert"
}
```

#### 同时更新 IPv4 和 IPv6

```json
{
  "x_auth_email": "user@example.com",
  "x_auth_key": "123456789abcdef",
  "zone_name": "example.com",
  "record_name": "home.example.com",
  "type": "A_AAAA",
  "ttl": 300,
  "ip": "auto",
  "proxy_status": true,
  "mode": "upsert"
}
```

#### 使用静态 IP 地址

```json
{
  "x_auth_email": "user@example.com",
  "x_auth_key": "123456789abcdef",
  "zone_name": "example.com",
  "record_name": "server.example.com",
  "type": "A",
  "ttl": 600,
  "ip": "192.168.1.100",
  "proxy_status": false,
  "mode": "upsert"
}
```

#### 删除 DNS 记录

```json
{
  "x_auth_email": "user@example.com",
  "x_auth_key": "123456789abcdef",
  "zone_name": "example.com",
  "record_name": "old.example.com",
  "mode": "delete"
}
```

### 带版本信息编译

将构建信息嵌入到二进制文件中：

```bash
go build \
  -ldflags="-X main.CommitHash=$(git rev-parse --short HEAD) \
            -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%S) \
            -X main.Version=$(git describe --tags --always)" \
  -o cloudflareddns main.go
```

### 开发

#### 代码检查

```bash
# 运行 linter
golangci-lint run

# 格式化代码
golangci-lint fmt
```

#### 编译

```bash
# 为当前平台编译
go build -o cloudflareddns main.go

# 为多平台编译
GOOS=linux GOARCH=amd64 go build -o cloudflareddns-linux-amd64 main.go
GOOS=linux GOARCH=arm64 go build -o cloudflareddns-linux-arm64 main.go
```

### Docker

本项目包含多阶段 Dockerfile 用于构建最小化镜像：

```bash
# 为 linux/amd64 构建
docker buildx build --platform linux/amd64 -t cloudflareddns:amd64 .

# 为 linux/arm64 构建
docker buildx build --platform linux/arm64 -t cloudflareddns:arm64 .

# 为两种架构同时构建
docker buildx build --platform linux/amd64,linux/arm64 -t cloudflareddns:latest .
```

### 获取 Cloudflare API 凭证

1. 登录您的 [Cloudflare 控制台](https://dash.cloudflare.com/)
2. 前往 **我的个人资料** → **API 令牌**或**全局 API 密钥**
3. **邮箱**：使用您的账户邮箱
4. **API 密钥**：您可以使用以下任一选项：
   - **全局 API 密钥**（在"全局 API 密钥"部分找到）
   - **源 CA 密钥**（用于创建证书）

⚠️ **注意**：此工具使用 `X-Auth-Email` 和 `X-Auth-Key` 请求头，而非 API 令牌。

### 输出示例

```
🚀 Cloudflare DDNS Tool v1.5.0

👤 Account: My Account
🌐 Zone ID: abc123def456

🔍 Checking A record...
🌍 WAN IP: 203.0.113.1
📝 Record does not exist, creating...
✅ Successfully created A record
```

### 许可证

MIT License

### 贡献

欢迎贡献！请随时提交 Pull Request。

---

## English

A lightweight, efficient Cloudflare DDNS updater written in Go. Automatically updates your DNS records when your WAN IP changes.

### Features

- 🚀 **Multiple DNS Record Types**: Support for A (IPv4), AAAA (IPv6), and both simultaneously
- 🔄 **Auto IP Detection**: Automatically detects WAN IP via Cloudflare trace API
- 🎯 **Dual Operation Modes**: Create/update DNS records or delete them
- 🐳 **Docker Support**: Multi-architecture Docker images (linux/amd64, linux/arm64)
- ⚡ **Fast & Lightweight**: Single binary compiled from Go with minimal dependencies
- 🔒 **Secure**: Uses Cloudflare's X-Auth-Email and X-Auth-Key authentication

### Quick Start

#### Using Docker

```bash
# Build the image
docker build -t cloudflareddns .

# Run the container
docker run -v $(pwd)/config.json:/app/config.json cloudflareddns
```

#### Using Binary

```bash
# Build the binary
go build -o cloudflareddns main.go

# Run
./cloudflareddns
```

### Configuration

Create a `config.json` file in the same directory as the binary:

```json
{
  "x_auth_email": "your_email@example.com",
  "x_auth_key": "your_cloudflare_api_key",
  "zone_name": "example.com",
  "record_name": "ddns.example.com",
  "type": "A",
  "ttl": 1,
  "ip": "auto",
  "proxy_status": false,
  "mode": "upsert"
}
```

#### Configuration Options

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `x_auth_email` | string | ✅ | Your Cloudflare account email |
| `x_auth_key` | string | ✅ | Your Cloudflare API key (Global API Key or Origin CA Key) |
| `zone_name` | string | ✅ | Your domain name (e.g., `example.com`) |
| `record_name` | string | ✅ | Full DNS record name (e.g., `ddns.example.com`) |
| `type` | string | ❌ | Record type: `A`, `AAAA`, or `A_AAAA` (default: `A`) |
| `ttl` | int | ❌ | TTL value: `1` (auto) or `120`-`86400` in seconds (default: `1`) |
| `ip` | string | ❌ | IP address: `auto` (detect), static IP, or `ipv4,ipv6` (default: `auto`) |
| `proxy_status` | bool | ❌ | Enable Cloudflare proxy: `true` or `false` (default: `false`) |
| `mode` | string | ❌ | Operation mode: `upsert` (create/update) or `delete` (default: `upsert`) |

#### Valid TTL Values

- `1` - Auto (Cloudflare automatically optimizes)
- `120` - 2 minutes
- `300` - 5 minutes
- `600` - 10 minutes
- `900` - 15 minutes
- `1800` - 30 minutes
- `3600` - 1 hour
- `7200` - 2 hours
- `18000` - 5 hours
- `43200` - 12 hours
- `86400` - 24 hours

#### Record Types

- **A** - IPv4 address record
- **AAAA** - IPv6 address record
- **A_AAAA** - Both A and AAAA records simultaneously

#### Operation Modes

- **upsert** - Create DNS record if it doesn't exist, or update if it does
- **delete** - Delete the DNS record(s)

#### IP Configuration

- **auto** - Automatically detect your WAN IP via Cloudflare's trace API (recommended)
- **static** - Use a specific IP address (e.g., `"192.168.1.1"`)
- **dual** - Specify both IPv4 and IPv6 (e.g., `"192.168.1.1,2001:db8::1"`)

### Examples

#### Update IPv4 A Record

```json
{
  "x_auth_email": "user@example.com",
  "x_auth_key": "123456789abcdef",
  "zone_name": "example.com",
  "record_name": "home.example.com",
  "type": "A",
  "ttl": 1,
  "ip": "auto",
  "proxy_status": false,
  "mode": "upsert"
}
```

#### Update Both IPv4 and IPv6

```json
{
  "x_auth_email": "user@example.com",
  "x_auth_key": "123456789abcdef",
  "zone_name": "example.com",
  "record_name": "home.example.com",
  "type": "A_AAAA",
  "ttl": 300,
  "ip": "auto",
  "proxy_status": true,
  "mode": "upsert"
}
```

#### Use Static IP Addresses

```json
{
  "x_auth_email": "user@example.com",
  "x_auth_key": "123456789abcdef",
  "zone_name": "example.com",
  "record_name": "server.example.com",
  "type": "A",
  "ttl": 600,
  "ip": "192.168.1.100",
  "proxy_status": false,
  "mode": "upsert"
}
```

#### Delete DNS Record

```json
{
  "x_auth_email": "user@example.com",
  "x_auth_key": "123456789abcdef",
  "zone_name": "example.com",
  "record_name": "old.example.com",
  "mode": "delete"
}
```

### Building with Version Info

To embed build information into the binary:

```bash
go build \
  -ldflags="-X main.CommitHash=$(git rev-parse --short HEAD) \
            -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%S) \
            -X main.Version=$(git describe --tags --always)" \
  -o cloudflareddns main.go
```

### Development

#### Linting

```bash
# Run linters
golangci-lint run

# Format code
golangci-lint fmt
```

#### Build

```bash
# Build for current platform
go build -o cloudflareddns main.go

# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o cloudflareddns-linux-amd64 main.go
GOOS=linux GOARCH=arm64 go build -o cloudflareddns-linux-arm64 main.go
```

### Docker

The project includes a multi-stage Dockerfile for building minimal images:

```bash
# Build for linux/amd64
docker buildx build --platform linux/amd64 -t cloudflareddns:amd64 .

# Build for linux/arm64
docker buildx build --platform linux/arm64 -t cloudflareddns:arm64 .

# Build for both architectures
docker buildx build --platform linux/amd64,linux/arm64 -t cloudflareddns:latest .
```

### Getting Cloudflare API Credentials

1. Log in to your [Cloudflare Dashboard](https://dash.cloudflare.com/)
2. Go to **My Profile** → **API Tokens** or **Global API Key**
3. For **Email**: Use your account email
4. For **API Key**: You can use either:
   - **Global API Key** (found under "Global API Key" section)
   - **Origin CA Key** (for creating certificates)

⚠️ **Note**: This tool uses the `X-Auth-Email` and `X-Auth-Key` headers, not API tokens.

### Output Example

```
🚀 Cloudflare DDNS Tool v1.5.0

👤 Account: My Account
🌐 Zone ID: abc123def456

🔍 Checking A record...
🌍 WAN IP: 203.0.113.1
📝 Record does not exist, creating...
✅ Successfully created A record
```

### License

MIT License

### Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
