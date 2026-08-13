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
// Multiple sections — of the same or different providers — may be
// configured at once; every section becomes one entry in the list.
func All(cfg *config.Config) ([]ddns.Provider, error) {
	var list []ddns.Provider

	for i := range cfg.Cloudflare {
		p, err := cloudflare.New(&cfg.Cloudflare[i])
		if err != nil {
			return nil, fmt.Errorf("cloudflare[%d]: %w", i, err)
		}
		list = append(list, p)
	}

	if len(list) == 0 {
		return nil, errors.New("no provider configured")
	}

	return list, nil
}
