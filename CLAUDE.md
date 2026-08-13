# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

ZJDDNS — a lightweight, zero-dependency DDNS updater that keeps DNS records in sync with the WAN IP. Pure Go, module `zjddns`, **zero third-party dependencies** (a deliberate feature — do not add external packages without explicit approval; DNS queries are hand-rolled for this reason). Cloudflare is the first DDNS provider behind a Provider abstraction; more providers are expected. User docs: [README.md](README.md).

## Commands

```bash
# Build
go build -o zjddns ./cmd/zjddns

# Build with version metadata (ldflags target package main in cmd/zjddns)
go build \
  -ldflags="-X main.CommitHash=$(git rev-parse --short HEAD) \
            -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%S)" \
  -o zjddns ./cmd/zjddns

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
internal/log           ← zero deps (foundation): leveled logging, component filter
internal/ipdetect       ← zero deps: WAN IP detection
config                  ← zero deps: load/defaults/validate/example
providers/cloudflare    ← config: Cloudflare API v4 client; implements ddns.Provider
ddns                    ← config + internal/ipdetect: Provider interface + upsert/delete orchestration
providers               ← ddns + providers/cloudflare + config: one instance per configured section
cmd/zjddns              ← wires everything: flags, banner, log level, update loop
```

Key flows (need reading across packages to see):

- **Provider mechanism** (`ddns.Provider`, defined consumer-side in ddns): `Upsert(recordType, ip)` / `Delete(recordType)`. **No provider selector field** — every configured provider section (e.g. `"cloudflare": {...}`) becomes one Provider instance; multiple sections run simultaneously, each updating its own records. `providers.All(cfg)` builds the list. Adding a provider = new `providers/<name>/` package + its config section struct in `config/` + a case in `providers.All` — do not leak provider specifics into `ddns` or `cmd`.
- **Provider construction** (`providers/cloudflare.New`): prints the deprecated-auth warning (provider-specific), resolves the Zone ID once at startup (prints `🌐 Zone ID`), stores it on the Client; errors bubble up as provider-init failure in cmd.
- **Run dispatch** (`ddns.Runner.Run`): providers split by `Provider.Mode()` — upserters run the union-type flow (per record type `ipdetect.Detector.WANIP` once → push the same IP to every upserter handling that type), deleters each delete their own types. Mixed modes in one config are supported. Cloudflare upsert implementation: `RecordID` ("" = missing) → create, or `RecordContent` compare → update if changed. Record-level logging is owned by the provider implementation; Runner logs only the generic lines.
- **WAN IP detection** (`internal/ipdetect`): pure detection — `Detector.IPv4()` / `IPv6()`, no record-type or config knowledge. DNS CHAOS TXT query to `whoami.cloudflare` via family-forced server (1.1.1.1 / 2606:4700:4700::1111) with HTTP trace (`/cdn-cgi/trace`) fallback. DNS packet build/parse is hand-rolled in `dnsquery.go` (UDP with TCP retry on TC flag) — this is why there are no deps. Static IP config resolution (`"ipv4,ipv6"` dual form) lives in `ddns` (record types are its domain).
- **Auth** (`providers/cloudflare.Client.request`): Bearer `api_token` only — the legacy X-Auth-* method was dropped in 2.0.0.
- **Cloudflare API client** (`providers/cloudflare`): typed response decoding (`zone`, `dnsRecord`) over `json.RawMessage` envelopes — no `interface{}` digging; sentinel errors (`ErrZoneNotFound`, `ErrRecordNotFound`) for checkable states; `Client.baseURL` is injectable so tests run against `httptest.Server` (see `client_test.go`).
- **Config semantics** (`config`): top level holds provider-agnostic settings (`ip`/`update_interval`/`log_level`); each provider owns a nested **section array** (e.g. `"cloudflare": [...]`) — the same provider may be configured multiple times for different domains, every entry runs concurrently. All value constants (`DefaultMode`, `ModeUpsert`, `TypeAAndAAAA`, `DefaultIP`, …) live in `config/defaults.go` — no magic strings in other packages. `SetDefaults()` runs before `Validate()` in cmd and applies per-section defaults; `Validate()` requires at least one section, then validates each entry (errors carry the section index, `cloudflare[1]: ...`); type/TTL checks are skipped in `delete` mode. Shared validators (`validModes`, `validTypes`) are ZJDDNS-wide abstractions; provider-specific rules (e.g. Cloudflare's `validTTLs` with 1 = auto) live next to their section struct.
- **Provider construction** (`providers.All`): iterates every section and calls the provider constructor with **just its own section** (`cloudflare.New(section)`) — provider packages never hold the whole `Config`; `providers/cloudflare.Client` binds a single `*config.CloudflareConfig`.

## Logging

All logs use `zjddns/internal/log` (package-level `Default` logger, ZJDNS-style). Default level: `info`, configurable via config `log_level` including component filtering (`"debug:CLOUDFLARE,IPDETECT"`).

- **No emoji in log output** — prefix + plain text only.
- **Canonical prefixes**: `DDNS` (runner), `CLOUDFLARE` (cloudflare provider), `IPDETECT` (WAN IP detection), `CONFIG` (config & startup).
- **Level semantics**: Error = component failure; Warn = fallback path (e.g. DNS detection fallback); Info = lifecycle + **state changes only** — every Info line carries the record name and, for updates, the IP change and how the IP was obtained (`Updated A record ddns.example.com: 1.2.3.4 -> 5.6.7.8 (DNS)`); Debug = routine per-cycle detail (checking, record lookups, no-op "unchanged", detection source). Default `info` is quiet when nothing changes — this is intentional; use `log_level: "debug"` for troubleshooting.
- **IP provenance**: `ipdetect.IP{Value, Source}` flows through the whole pipeline (`Source` = `DNS`/`HTTP`/`static`) so state-change messages show how the IP was obtained.
- **Filter rule**: component filter gates only Info/Debug; Error/Warn always pass so a filtered component cannot swallow operational failures.

## Conventions

- **Zero-warning lint.** All suppressions are inline `//nolint:NAME // reason`; no global excludes. Declaration order per file (`decorder`): `type → const → var → func`. Formatter: `gofumpt` (single alphabetized import group — no blank-line groups, no manual grouping).
- **Comments**: godoc style, **English only** (matching ZJDNS); exported identifiers start with their own name; full sentences ending with a period. Package docs describe the role and key invariants.
- **Naming**: PascalCase/camelCase, acronyms all-caps (`WANIP`, `APIBase`), no `Get` prefix (`client.ZoneID()`), `any` not `interface{}`, `errors.New` for static error strings (perfsprint), `time.DateTime` over layout strings.
- **Version bumping** (`sh scripts/bump-version.sh`): patch = bug fixes, perf, refactors, lint; minor = new features/config options; major = breaking config changes. Default to **patch**. Version lives in `cmd/zjddns/version.go` (`ProjectName`/`Version`/`CommitHash`/`BuildTime`, ZJDNS pattern — ldflags fields surfaced only when set).
- **Behavior preservation**: keep all emoji console output, config defaults, and the deprecated-auth warning path unchanged. Config JSON schema is backward compatible.
- **Testing style**: table-driven with `t.Run`, `TestXxx` names; pure-function tests only (no network mocks) — `ipdetect` tests craft DNS response bytes by hand.
