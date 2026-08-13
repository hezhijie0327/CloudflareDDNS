# Cloudflare Provider

ZJDDNS 内置的 Cloudflare DDNS 提供商，通过 Cloudflare API v4 管理 zone 下的 DNS 记录。配置段为 `"cloudflare": [...]` 数组——同一提供商可配置多段，每段独立更新一个域名记录。

## 配置字段

| 字段           | 类型   | 必填 | 说明                                                                 |
| -------------- | ------ | ---- | -------------------------------------------------------------------- |
| `api_token`    | string | ✅   | 您的 Cloudflare API Token                                            |
| `zone_name`    | string | ✅   | 您的域名（如 `example.com`）                                         |
| `record_name`  | string | ✅   | 完整的 DNS 记录名称（如 `ddns.example.com`）                         |
| `mode`         | string | ❌   | 操作模式：`upsert`（创建/更新）或 `delete`（默认：`upsert`）         |
| `type`         | string | ❌   | 记录类型：`A`、`AAAA` 或 `A_AAAA`（默认：`A`）                      |
| `ttl`          | int    | ❌   | TTL 值（默认：`1`，Cloudflare 自动优化）                             |
| `proxy_status` | bool   | ❌   | 启用 Cloudflare 代理（橙色云）：`true` 或 `false`（默认：`false`）   |

## 有效的 TTL 值

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

## 使用示例

### 更新 IPv4 A 记录

```json
{
  "ip": "auto",
  "cloudflare": [
    {
      "api_token": "your_cloudflare_api_token",
      "zone_name": "example.com",
      "record_name": "home.example.com",
      "mode": "upsert",
      "type": "A",
      "ttl": 1,
      "proxy_status": false
    }
  ]
}
```

### 同时更新 IPv4 和 IPv6

```json
{
  "ip": "auto",
  "cloudflare": [
    {
      "api_token": "your_cloudflare_api_token",
      "zone_name": "example.com",
      "record_name": "home.example.com",
      "mode": "upsert",
      "type": "A_AAAA",
      "ttl": 300,
      "proxy_status": true
    }
  ]
}
```

### 使用静态 IP 地址

```json
{
  "ip": "192.168.1.100",
  "cloudflare": [
    {
      "api_token": "your_cloudflare_api_token",
      "zone_name": "example.com",
      "record_name": "server.example.com",
      "mode": "upsert",
      "type": "A",
      "ttl": 600,
      "proxy_status": false
    }
  ]
}
```

### 删除 DNS 记录

```json
{
  "cloudflare": [
    {
      "api_token": "your_cloudflare_api_token",
      "zone_name": "example.com",
      "record_name": "old.example.com",
      "mode": "delete"
    }
  ]
}
```

### 同一提供商的多个域名

```json
{
  "ip": "auto",
  "cloudflare": [
    {
      "api_token": "your_cloudflare_api_token",
      "zone_name": "example.com",
      "record_name": "ddns.example.com",
      "mode": "upsert",
      "type": "A"
    },
    {
      "api_token": "your_cloudflare_api_token",
      "zone_name": "example.net",
      "record_name": "ddns.example.net",
      "mode": "upsert",
      "type": "A_AAAA"
    }
  ]
}
```

## 获取 API 凭证

1. 登录您的 [Cloudflare 控制台](https://dash.cloudflare.com/)
2. 前往 **我的个人资料** → **API 令牌**
3. 点击 **创建令牌**
4. 在创建 API Token 时，需要配置以下权限：
   - **区域** → **区域设置** → **编辑**
   - **区域** → **区域** → **编辑**
   - **区域** → **DNS** → **编辑**
5. 可以选择 **区域资源** 来限制 Token 只能访问特定域名
6. 创建后，复制 Token 并填写到配置段的 `api_token` 字段
