# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

CloudflareDDNS — a lightweight DDNS updater that keeps Cloudflare DNS records in sync with the WAN IP. Pure Go, module `cloudflareddns`, **zero third-party dependencies** (a deliberate feature — do not add external packages without explicit approval; DNS queries are hand-rolled for this reason). User docs: [README.md](README.md).

## Commands

```bash
# Build
go build -o cloudflareddns ./cmd/cloudflareddns

# Build with version metadata (ldflags target package main in cmd/cloudflareddns)
go build \
  -ldflags="-X main.CommitHash=$(git rev-parse --short HEAD) \
            -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%S)" \
  -o cloudflareddns ./cmd/cloudflareddns

# Tests
go test ./...                          # all
go test ./config -run TestValidate -v  # single package/test

# Lint — REQUIRED before committing, zero warnings (see .golangci.yml)
go fix ./... && golangci-lint run && golangci-lint fmt

# Scripts
sh scripts/install-hook.sh     # install pre-commit hook (auto fmt+lint; already installed locally)
sh scripts/bump-version.sh patch   # patch|minor|major — bumps version.go + README badge
```

## Architecture

Layered DAG mirroring ZJDNS — imports only flow upward; no cycles:

```
internal/ipdetect   ← zero deps (foundation): WAN IP detection
cloudflare          ← zero deps: minimal Cloudflare API v4 client
config              ← zero deps: load/defaults/validate/example
ddns                ← cloudflare + internal/ipdetect: upsert/delete orchestration (the "server" layer)
cmd/cloudflareddns  ← wires everything: flags, zone lookup, update loop
```

Key flows (need reading across packages to see):

- **Upsert** (`ddns.Runner.Upsert`): per record type — `ipdetect.Detector.WANIP` → `cloudflare.Client.RecordID` ("" = missing) → create, or `RecordContent` compare → update if changed.
- **WAN IP detection** (`internal/ipdetect`): DNS CHAOS TXT query to `whoami.cloudflare` via family-forced server (1.1.1.1 / 2606:4700:4700::1111) with HTTP trace (`/cdn-cgi/trace`) fallback; static IP config supports `"ipv4,ipv6"` dual form. DNS packet build/parse is hand-rolled in `dnsquery.go` (UDP with TCP retry on TC flag) — this is why there are no deps.
- **Auth** (`cloudflare.Client.request`): Bearer `api_token` preferred; legacy `X-Auth-Email`/`X-Auth-Key` headers kept for backward compatibility — both paths must keep working.
- **Config semantics** (`config`): `SetDefaults()` runs before `Validate()` in cmd; `update_interval` is a pointer (nil → 300s, 0 → run once); `Validate()` skips type/TTL checks in `delete` mode; the deprecated auth fields intentionally trigger staticcheck deprecation notices — expected.

## Conventions

- **Zero-warning lint.** All suppressions are inline `//nolint:NAME // reason`; no global excludes. Declaration order per file (`decorder`): `type → const → var → func`. Formatter: `gofumpt` (single alphabetized import group — no blank-line groups, no manual grouping).
- **Naming**: PascalCase/camelCase, acronyms all-caps (`WANIP`, `APIBase`), no `Get` prefix (`client.ZoneID()`), `any` not `interface{}`, `errors.New` for static error strings (perfsprint), `time.DateTime` over layout strings.
- **Version bumping** (`sh scripts/bump-version.sh`): patch = bug fixes, perf, refactors, lint; minor = new features/config options; major = breaking config changes. Default to **patch**. Version lives in `cmd/cloudflareddns/version.go` (`ProjectName`/`Version`/`CommitHash`/`BuildTime`, ZJDNS pattern — ldflags fields surfaced only when set).
- **Behavior preservation**: keep all emoji console output, config defaults, and the deprecated-auth warning path unchanged. Config JSON schema is backward compatible.
- **Testing style**: table-driven with `t.Run`, `TestXxx` names; pure-function tests only (no network mocks) — `ipdetect` tests craft DNS response bytes by hand.
