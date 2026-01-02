# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

This is a Cloudflare DDNS tool written in Go that dynamically updates DNS records via Cloudflare API. The binary is compiled into Docker images for multi-platform deployment (linux/amd64, linux/arm64).

## Build & Test Commands

```bash
# Build the Go binary
go build -o cloudflareddns main.go

# Run linters (must pass before committing)
golangci-lint run
golangci-lint fmt

# Run the binary locally
./cloudflareddns

# Generate example config
./cloudflareddns -generate-config > config.json
```

## Architecture

### Single-File Structure
The entire application is in `main.go` (~600 lines). Core components:

- **Config**: JSON-based configuration loaded from `config.json` with validation for:
  - **Authentication**: `api_token` (recommended) OR deprecated `x_auth_email` + `x_auth_key`
  - Required fields: `zone_name`, `record_name`
  - Record types: `A`, `AAAA`, or `A_AAAA` (both)
  - Valid TTLs: 1 (auto), 120, 300, 600, 900, 1800, 3600, 7200, 18000, 43200, 86400
  - Modes: `upsert` (create/update), `delete`
  - `update_interval`: Seconds between updates (default: 300, 0: run once and exit)

- **HTTPClient**: Encapsulates all Cloudflare API interactions
  - Supports Bearer token auth (`api_token`) or X-Auth-Email/X-Auth-Key headers (deprecated)
  - Centralized `request()` method handles auth, marshaling, and error checking
  - Timeout: 5 seconds (configurable via `RequestTimeout` constant)

### Execution Flow

1. Load and validate `config.json`
2. Fetch account name and zone ID (auth validation happens here)
3. Based on `mode`:
   - **upsert**: Get current WAN IP → Check if DNS record exists → Create or update
   - **delete**: Fetch DNS record IDs → Delete records
4. If `update_interval` > 0, run periodically using ticker; otherwise exit after one run

### IP Resolution

- **Auto mode**: Queries `https://api.cloudflare.com/cdn-cgi/trace` and parses `ip=` line
  - For A records: forces IPv4 via `tcp4` dialer
  - For AAAA records: forces IPv6 via `tcp6` dialer
- **Static mode**: Parses `ip` config field (supports `ipv4,ipv6` format for dual-stack)
- **Validation**: `net.ParseIP()` with IPv4/IPv6 type checking based on record type

### Key Design Decisions

- **Single-file architecture**: All functionality in `main.go` for simplicity and easy deployment
- **Dynamic response parsing**: Uses `map[string]interface{}` for API responses to avoid strict struct dependencies on Cloudflare's API structure
- **Non-fatal errors in loops**: In `handleUpsert()` and `handleDelete()`, errors for one record type don't stop processing of other types
- **No network connectivity check**: Removed redundant `checkConnectivity()` - auth validation happens naturally during first API call
- **Defer cleanup pattern**: All `resp.Body.Close()` calls must check return values using closure pattern:
  ```go
  defer func() {
      _ = resp.Body.Close()
  }()
  ```

## Linting Rules

The project uses `golangci-lint` with strict error checking:
- `errcheck`: All error return values must be checked (including `defer resp.Body.Close()`)
- Format issues are caught via `golangci-lint fmt`

## Docker Deployment

- Build workflow: `.github/workflows/main.yml`
- Multi-arch builds: linux/amd64, linux/arm64
- Multi-registry: Pushes to both Docker Hub (hezhijie0327/cloudflareddns) and GHCR
- Automated builds: Scheduled at 8:00 and 20:00 UTC+8 daily
- Base image: scratch (minimal final image size with CA certs bundle)
