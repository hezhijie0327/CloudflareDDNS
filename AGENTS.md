# AGENTS.md

This file contains guidelines for agentic coding agents working on this Cloudflare DDNS repository.

## Project Overview

This is a lightweight, efficient Cloudflare DDNS updater written in Go that automatically updates DNS records when WAN IP changes. The project is structured as a single-file application with all logic in `main.go`.

## Build Commands

```bash
# Build the application
go build \
  -ldflags="-X main.CommitHash=$(git rev-parse --short HEAD) \
            -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%S) \
  -o cloudflareddns main.go

# Build with Docker
docker build -t cloudflareddns .

# Run the application
./cloudflareddns

# Run with config file
./cloudflareddns -config /path/to/config.json

# Generate example config
./cloudflareddns -generate-config > config.json
```

## Lint/Formatting Commands

```bash
# Run linters
golangci-lint run

# Format code
golangci-lint fmt

# Alternative formatting
go fmt ./...
```

## Testing

This project doesn't have automated tests currently. When testing changes:

1. Verify the application builds without errors
2. Run with example configurations to ensure functionality
3. Test all command line arguments (-config, -generate-config, -version)
4. Verify both API token and legacy authentication methods

## Code Style Guidelines

### Imports

- Group imports by standard library, third-party, and local packages
- Use explicit imports (no dot imports)
- Order imports alphabetically within each group

```go
import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "io"
    "net"
    "net/http"
    "os"
    "runtime"
    "strings"
    "time"
)
```

### Naming Conventions

- Use CamelCase for exported types, functions, and variables
- Use camelCase for unexported (private) types, functions, and variables
- Use ALL_CAPS for constants
- Use descriptive names that clearly indicate purpose

Example:
```go
type Config struct {
    APIToken string `json:"api_token"`
    XAuthEmail string `json:"x_auth_email,omitempty"`
}

func (h *HTTPClient) request(method, path string, payload interface{}) (*CloudflareResponse, error) {
    // Implementation
}
```

### Error Handling

- Always handle errors explicitly
- Return errors from functions that can fail
- Use fmt.Errorf for error messages with context
- Use %w verb for wrapping errors when relevant

Example:
```go
data, err := os.ReadFile(path)
if err != nil {
    return nil, fmt.Errorf("read config file: %w", err)
}
```

### JSON Tags

- Use json tags for all struct fields that are serialized
- Use omitempty for optional fields
- Use snake_case for JSON field names

Example:
```go
type Config struct {
    APIToken string `json:"api_token"`
    XAuthEmail string `json:"x_auth_email,omitempty"`
    UpdateInterval *int `json:"update_interval,omitempty"`
}
```

### Constants

- Define constants for magic numbers and strings
- Group related constants together

Example:
```go
const (
    CloudflareAPI  = "https://api.cloudflare.com"
    RequestTimeout = 5 * time.Second
)
```

### Comments

- Use godoc format for exported functions and types
- Add comments for complex logic
- Use inline comments sparingly, only for non-obvious code

Example:
```go
// Config 配置结构
type Config struct {
    // APIToken is the Cloudflare API token for authentication
    APIToken string `json:"api_token"`
}
```

## Project Structure

- `main.go`: All application code in a single file
- `go.mod`: Go module definition
- `Dockerfile`: Multi-stage build for containerization
- `README.md`: Project documentation
- `config.json`: Configuration file (not tracked in git)

## Key Design Patterns

1. **Single File Architecture**: All code is contained in main.go for simplicity
2. **Configuration via JSON**: All settings are loaded from a JSON config file
3. **Default Values**: Set sensible defaults in the `setDefaults()` method
4. **Error Wrapping**: Use fmt.Errorf with %w to preserve error context
5. **HTTP Client Wrapper**: The HTTPClient struct encapsulates API interactions

## When Making Changes

1. Ensure backward compatibility for configuration options
2. Test both API token and legacy authentication methods
3. Verify all command line arguments still work
4. Test Docker build process
5. Update version number if needed (Version variable in main.go)

## Security Considerations

- Never log sensitive information like API tokens
- Validate all configuration values
- Use proper HTTP timeouts
- Handle all potential errors to avoid crashes