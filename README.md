# ZJDDNS

```
███████╗     ██╗██████╗ ██████╗ ███╗   ██╗███████╗
╚══███╔╝     ██║██╔══██╗██╔══██╗████╗  ██║██╔════╝
  ███╔╝      ██║██║  ██║██║  ██║██╔██╗ ██║███████╗
 ███╔╝  ██   ██║██║  ██║██║  ██║██║╚██╗██║╚════██║
███████╗╚█████╔╝██████╔╝██████╔╝██║ ╚████║███████║
╚══════╝ ╚════╝ ╚═════╝ ╚═════╝ ╚═╝  ╚═══╝╚══════╝

```

[![Version](https://img.shields.io/badge/Version-2.0.0-informational)](https://github.com/hezhijie0327/ZJDDNS/releases)
[![License](https://img.shields.io/badge/License-Apache%202.0--Commons%20Clause-blue)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev/)
[![Lint](https://img.shields.io/badge/golangci--lint-0%20issues-success)](https://golangci-lint.run/)

轻量级、零第三方依赖的 DDNS 更新工具。WAN IP 变化时自动更新 DNS 记录。

## 快速开始

### 使用 Docker

```bash
# 运行容器（默认使用 config.json）
docker run -v $(pwd)/config.json:/config.json hezhijie0327/zjddns:latest

# 指定配置文件路径
docker run -v $(pwd)/myconfig.json:/myconfig.json hezhijie0327/zjddns:latest -config myconfig.json
```

### 使用二进制

```bash
# 构建
go build -o zjddns ./cmd/zjddns

# 运行（默认使用 config.json）
./zjddns

# 指定配置文件路径
./zjddns -config /path/to/config.json

# 生成示例配置文件
./zjddns -generate-config > config.json
```

## 核心特性

- 🚀 **多种 DNS 记录类型**：支持 A（IPv4）、AAAA（IPv6）以及同时更新两种记录
- 🔄 **自动 IP 检测**：DNS 查询优先，失败自动回退 HTTP trace
- 🎯 **双操作模式**：创建/更新 DNS 记录或删除记录
- 🧱 **零第三方依赖**：纯标准库构建，DNS 报文查询/解析为手写实现
- 🐳 **Docker 支持**：多架构 Docker 镜像（linux/amd64、linux/arm64）
- 🔌 **Provider 机制**：每个提供商一个配置段，**同一提供商可配多个域名、多个提供商可并存**；新提供商以插件包形式接入（`providers/`）

## 命令行参数

| 参数               | 说明                       | 默认值        |
| ------------------ | -------------------------- | ------------- |
| `-config`          | 指定配置文件路径           | `config.json` |
| `-generate-config` | 生成示例配置文件到标准输出 | -             |
| `-version`         | 显示版本信息               | -             |
| `-h` / `-help`     | 显示帮助信息               | -             |

## 配置

在二进制文件所在目录创建 `config.json` 文件，或使用 `-generate-config` 生成示例：

```json
{
  "ip": "auto",
  "update_interval": 300,
  "log_level": "info",
  "cloudflare": [
    {
      "api_token": "your_cloudflare_api_token",
      "zone_name": "example.com",
      "record_name": "ddns.example.com",
      "mode": "upsert",
      "type": "A_AAAA",
      "ttl": 1,
      "proxy_status": false
    }
  ]
}
```

### 公用配置选项（顶层）

| 字段             | 类型   | 必填 | 说明                                                                  |
| ---------------- | ------ | ---- | --------------------------------------------------------------------- |
| `ip`             | string | ❌   | IP 地址：`auto`（自动检测）、静态 IP 或 `ipv4,ipv6`（默认：`auto`）   |
| `update_interval`| int    | ❌   | 更新间隔秒数（默认：`300`，`0` 表示只运行一次）                       |
| `log_level`      | string | ❌   | 日志级别：`error`/`warn`/`info`/`debug`，支持 `debug:CLOUDFLARE,IPDETECT` 组件过滤（默认：`info`） |

### 提供商配置

- **Cloudflare**：配置字段、TTL 规则、使用示例与 API 凭证申请见 [docs/providers/cloudflare.md](docs/providers/cloudflare.md)

### 记录类型

- **A** - IPv4 地址记录
- **AAAA** - IPv6 地址记录
- **A_AAAA** - 同时创建 A 和 AAAA 记录

### 操作模式

- **upsert** - 如果 DNS 记录不存在则创建，如果存在则更新
- **delete** - 删除 DNS 记录

### IP 配置

- **auto** - 自动检测您的 WAN IP（推荐）
- **static** - 使用指定的 IP 地址（如 `"192.168.1.1"`）
- **dual** - 同时指定 IPv4 和 IPv6（如 `"192.168.1.1,2001:db8::1"`）

## 项目结构

```
zjddns/
├── cmd/zjddns/             ← 二进制入口（CLI 参数、banner、更新循环、版本信息）
├── config/                 ← 配置加载、默认值、校验、示例生成
├── providers/              ← DDNS Provider 构造（每个配置段一个实例，可并存）
│   └── cloudflare/         ← Cloudflare API v4 客户端（zone、DNS 记录 CRUD）
├── ddns/                   ← Provider 接口 + upsert/delete 编排（IP 检测接线）
├── internal/ipdetect/      ← WAN IP 检测（手写 DNS 查询 + HTTP trace 回退）
└── scripts/                ← pre-commit hook 安装、版本 bump
```

## 开发

### 代码检查

```bash
# 运行 linter（提交前必须零警告）
go fix ./... && golangci-lint run

# 格式化代码（gofumpt）
golangci-lint fmt
```

### 测试

```bash
go test ./...
```

### 编译

```bash
go build \
  -ldflags="-X main.CommitHash=$(git rev-parse --short HEAD) \
            -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%S)" \
  -o zjddns ./cmd/zjddns
```

### 开发辅助脚本

```bash
# 安装 pre-commit hook（提交前自动格式化 + lint）
sh scripts/install-hook.sh        # Linux / macOS
pwsh scripts/install-hook.ps1     # Windows

# 版本号 bump（patch/minor/major，语义见 CLAUDE.md）
sh scripts/bump-version.sh patch
```

## License

[Apache License 2.0 with Commons Clause v1.0](LICENSE)
