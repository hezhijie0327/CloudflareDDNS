package providers

import (
	"strings"
	"testing"
	"zjddns/config"
)

func TestAllNoProviders(t *testing.T) {
	cfg := &config.Config{}

	if _, err := All(cfg); err == nil {
		t.Fatal("All() error = nil, want error when no provider section is configured")
	} else if !strings.Contains(err.Error(), "no provider") {
		t.Errorf("All() error = %v, want containing %q", err, "no provider")
	}
}
