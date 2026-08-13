// Package providers registers and constructs DDNS provider implementations.
package providers

import (
	"errors"
	"fmt"
	"zjddns/config"
	"zjddns/ddns"
	"zjddns/providers/cloudflare"
)

// All constructs one Provider per configured section in cfg.
// Multiple providers may be configured at once — each updates its own
// records — and every configured section becomes one entry in the list.
func All(cfg *config.Config) ([]ddns.Provider, error) {
	var list []ddns.Provider

	if cfg.Cloudflare != nil {
		p, err := cloudflare.New(cfg)
		if err != nil {
			return nil, fmt.Errorf("cloudflare: %w", err)
		}
		list = append(list, p)
	}

	if len(list) == 0 {
		return nil, errors.New("no provider configured")
	}

	return list, nil
}
