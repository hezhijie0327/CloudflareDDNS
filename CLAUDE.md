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
internal/ipdetect       ← zero deps (foundation): WAN IP detection
config                  ← zero deps: load/defaults/validate/example
providers/cloudflare    ← zero deps: Cloudflare API v4 client; implements ddns.Provider
ddns                    ← config + internal/ipdetect: Provider interface + upsert/delete orchestration
providers               ← ddns + providers/cloudflare + config: one instance per configured section
cmd/zjddns              ← wires everything: flags, banner, provider selection, update loop
```

Key flows (need reading across packages to see):

- **Provider mechanism** (`ddns.Provider`, defined consumer-side in ddns): `Upsert(recordType, ip)` / `Delete(recordType)`. **No provider selector field** — every configured provider section (e.g. `"cloudflare": {...}`) becomes one Provider instance; multiple sections run simultaneously, each updating its own records. `providers.All(cfg)` builds the list. Adding a provider = new `providers/<name>/` package + its config section struct in `config/` + a case in `providers.All` — do not leak provider specifics into `ddns` or `cmd`.
- **Provider construction** (`providers/cloudflare.New`): prints the deprecated-auth warning (provider-specific), resolves the Zone ID once at startup (prints `🌐 Zone ID`), stores it on the Client; errors bubble up as provider-init failure in cmd.
- **Upsert** (`ddns.Runner.Upsert`): per record type — `ipdetect.Detector.WANIP` once → push the same IP to **every** provider. Cloudflare implementation: `RecordID` ("" = missing) → create, or `RecordContent` compare → update if changed. Record-level console output (record IDs, create/update details, failures) is owned by the provider implementation; Runner prints only the generic lines (🔍/🌍/❌ WAN IP).
- **WAN IP detection** (`internal/ipdetect`): DNS CHAOS TXT query to `whoami.cloudflare` via family-forced server (1.1.1.1 / 2606:4700:4700::1111) with HTTP trace (`/cdn-cgi/trace`) fallback; static IP config supports `"ipv4,ipv6"` dual form. DNS packet build/parse is hand-rolled in `dnsquery.go` (UDP with TCP retry on TC flag) — this is why there are no deps.
- **Auth** (`providers/cloudflare.Client.request`): Bearer `api_token` preferred; legacy `X-Auth-Email`/`X-Auth-Key` headers kept for backward compatibility — both paths must keep working.
- **Config semantics** (`config`): top level holds provider-agnostic settings (`type`/`ttl`/`ip`/`mode`/`update_interval`); each provider gets its own nested section struct (`CloudflareConfig`, pointer field — presence = enabled). `SetDefaults()` runs before `Validate()` in cmd; `Validate()` requires at least one provider section, then validates each section; `update_interval` is a pointer (nil → 300s, 0 → run once); type/TTL checks are skipped in `delete` mode. The deprecated auth fields intentionally trigger staticcheck deprecation notices — expected.

## Conventions

- **Zero-warning lint.** All suppressions are inline `//nolint:NAME // reason`; no global excludes. Declaration order per file (`decorder`): `type → const → var → func`. Formatter: `gofumpt` (single alphabetized import group — no blank-line groups, no manual grouping).
- **Naming**: PascalCase/camelCase, acronyms all-caps (`WANIP`, `APIBase`), no `Get` prefix (`client.ZoneID()`), `any` not `interface{}`, `errors.New` for static error strings (perfsprint), `time.DateTime` over layout strings.
- **Version bumping** (`sh scripts/bump-version.sh`): patch = bug fixes, perf, refactors, lint; minor = new features/config options; major = breaking config changes. Default to **patch**. Version lives in `cmd/zjddns/version.go` (`ProjectName`/`Version`/`CommitHash`/`BuildTime`, ZJDNS pattern — ldflags fields surfaced only when set).
- **Behavior preservation**: keep all emoji console output, config defaults, and the deprecated-auth warning path unchanged. Config JSON schema is backward compatible.
- **Testing style**: table-driven with `t.Run`, `TestXxx` names; pure-function tests only (no network mocks) — `ipdetect` tests craft DNS response bytes by hand.
