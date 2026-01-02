# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

This is a Cloudflare DDNS tool written in Go that dynamically updates DNS records via Cloudflare API. The binary is compiled into Docker images for multi-platform deployment (linux/amd64, linux/arm64).

**Note:** The project is transitioning from a shell script implementation (CloudflareDDNS.sh, cloudflareddns.dockerfile) to a Go implementation (main.go, Dockerfile).

## Build & Test Commands

```bash
# Build the Go binary
go build -o cloudflareddns main.go

# Run linters (must pass before committing)
golangci-lint run
golangci-lint fmt

# Run the binary locally
./cloudflareddns
```

## Architecture

### Single-File Structure
The entire application is in `main.go` (~450 lines). Core components:

- **Config**: JSON-based configuration loaded from `config.json` with validation for:
  - Required fields: `x_auth_email`, `x_auth_key`, `zone_name`, `record_name`
  - Record types: `A`, `AAAA`, or `A_AAAA` (both)
  - Valid TTLs: 1 (auto), 120, 300, 600, 900, 1800, 3600, 7200, 18000, 43200, 86400
  - Modes: `upsert` (create/update), `delete`

- **HTTPClient**: Encapsulates all Cloudflare API interactions
  - Uses `X-Auth-Email` and `X-Auth-Key` headers (not API tokens)
  - Centralized `request()` method handles auth, marshaling, and error checking
  - Timeout: 5 seconds (configurable via `RequestTimeout` constant)

### Execution Flow

1. Load and validate `config.json`
2. Check network connectivity to Cloudflare API
3. Fetch account name and zone ID
4. Based on `mode`:
   - **upsert**: Get current WAN IP → Check if DNS record exists → Create or update
   - **delete**: Fetch DNS record IDs → Delete records

### IP Resolution

- **Auto mode**: Queries `https://api.cloudflare.com/cdn-cgi/trace` and parses `ip=` line
- **Static mode**: Parses `ip` config field (supports `ipv4,ipv6` format)
- **Validation**: `net.ParseIP()` with IPv4/IPv6 type checking based on record type

### Key Design Decisions

- **Dynamic response parsing**: Uses `map[string]interface{}` for API responses to avoid strict struct dependencies on Cloudflare's API structure
- **Non-fatal errors in loops**: In `handleUpsert()` and `handleDelete()`, errors for one record type don't stop processing of other types
- **Defer cleanup**: All `resp.Body.Close()` calls must check return values using closure pattern:
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
- Two Dockerfiles exist (transition period):
  - `Dockerfile` - Go implementation (new)
  - `cloudflareddns.dockerfile` - Shell script implementation (legacy)
