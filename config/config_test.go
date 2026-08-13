package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSetDefaults(t *testing.T) {
	tests := []struct {
		name string
		in   Config
		want Config
	}{
		{
			name: "empty config",
			in:   Config{},
			want: Config{IP: "auto", UpdateInterval: new(defaultUpdateIntervalSeconds)},
		},
		{
			name: "explicit values preserved",
			in:   Config{IP: "192.0.2.1", UpdateInterval: new(0)},
			want: Config{IP: "192.0.2.1", UpdateInterval: new(0)},
		},
		{
			name: "provider section defaults",
			in:   Config{Cloudflare: []CloudflareConfig{{}}},
			want: Config{IP: "auto", UpdateInterval: new(defaultUpdateIntervalSeconds), Cloudflare: []CloudflareConfig{{Mode: "upsert", Type: "A", TTL: 1}}},
		},
		{
			name: "defaults apply to every section",
			in:   Config{Cloudflare: []CloudflareConfig{{}, {ZoneName: "second.example"}}},
			want: Config{IP: "auto", UpdateInterval: new(defaultUpdateIntervalSeconds), Cloudflare: []CloudflareConfig{{Mode: "upsert", Type: "A", TTL: 1}, {ZoneName: "second.example", Mode: "upsert", Type: "A", TTL: 1}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.in
			cfg.SetDefaults()
			if !reflect.DeepEqual(cfg, tt.want) {
				t.Errorf("SetDefaults() = %+v, want %+v", cfg, tt.want)
			}
		})
	}
}

func TestInterval(t *testing.T) {
	cfg := Config{}
	if got := cfg.Interval(); got != defaultUpdateIntervalSeconds {
		t.Errorf("Interval() with nil pointer = %d, want %d", got, defaultUpdateIntervalSeconds)
	}

	cfg = Config{UpdateInterval: new(0)}
	if got := cfg.Interval(); got != 0 {
		t.Errorf("Interval() with explicit 0 = %d, want 0", got)
	}
}

func TestValidate(t *testing.T) {
	// validCfg is a fully valid upsert configuration with one provider.
	validCfg := func() Config {
		return Config{
			Cloudflare: []CloudflareConfig{
				{
					APIToken:   "token",
					ZoneName:   "example.com",
					RecordName: "ddns.example.com",
					Mode:       "upsert",
					Type:       "A",
					TTL:        1,
				},
			},
		}
	}

	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{name: "valid api token", cfg: validCfg()},
		{
			name: "multiple sections are valid",
			cfg: Config{Cloudflare: []CloudflareConfig{
				{APIToken: "token", ZoneName: "example.com", RecordName: "ddns.example.com", Mode: "upsert", Type: "A", TTL: 1},
				{APIToken: "token", ZoneName: "example.net", RecordName: "ddns.example.net", Mode: "upsert", Type: "AAAA", TTL: 300},
			}},
		},
		{
			name:    "no provider configured",
			cfg:     Config{},
			wantErr: "no provider",
		},
		{
			name:    "missing authentication",
			cfg:     Config{Cloudflare: []CloudflareConfig{{ZoneName: "example.com", RecordName: "ddns.example.com", Mode: "upsert", Type: "A", TTL: 1}}},
			wantErr: "authentication",
		},
		{
			name:    "missing zone name",
			cfg:     Config{Cloudflare: []CloudflareConfig{{APIToken: "token", RecordName: "ddns.example.com", Mode: "upsert", Type: "A", TTL: 1}}},
			wantErr: "zone_name",
		},
		{
			name:    "missing record name",
			cfg:     Config{Cloudflare: []CloudflareConfig{{APIToken: "token", ZoneName: "example.com", Mode: "upsert", Type: "A", TTL: 1}}},
			wantErr: "record_name",
		},
		{
			name:    "invalid mode",
			cfg:     Config{Cloudflare: []CloudflareConfig{{APIToken: "token", ZoneName: "example.com", RecordName: "ddns.example.com", Mode: "bogus", Type: "A", TTL: 1}}},
			wantErr: "invalid mode",
		},
		{
			name:    "invalid type",
			cfg:     Config{Cloudflare: []CloudflareConfig{{APIToken: "token", ZoneName: "example.com", RecordName: "ddns.example.com", Mode: "upsert", Type: "CNAME", TTL: 1}}},
			wantErr: "invalid type",
		},
		{
			name:    "invalid TTL",
			cfg:     Config{Cloudflare: []CloudflareConfig{{APIToken: "token", ZoneName: "example.com", RecordName: "ddns.example.com", Mode: "upsert", Type: "A", TTL: 60}}},
			wantErr: "invalid TTL",
		},
		{
			name: "delete mode skips type and TTL checks",
			cfg:  Config{Cloudflare: []CloudflareConfig{{APIToken: "token", ZoneName: "example.com", RecordName: "ddns.example.com", Mode: "delete"}}},
		},
		{
			name:    "error carries the section index",
			cfg:     Config{Cloudflare: []CloudflareConfig{{APIToken: "token", ZoneName: "example.com", RecordName: "ddns.example.com", Mode: "upsert", Type: "A", TTL: 1}, {ZoneName: "example.net", RecordName: "ddns.example.net", Mode: "upsert", Type: "A", TTL: 1}}},
			wantErr: "cloudflare[1]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate() error = nil, want containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Validate() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	t.Run("valid file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		content := `{"cloudflare":[{"api_token":"token","zone_name":"example.com","record_name":"ddns.example.com","mode":"delete","type":"AAAA","ttl":300}]}`
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}

		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if len(cfg.Cloudflare) != 1 {
			t.Fatalf("Load() Cloudflare sections = %d, want 1", len(cfg.Cloudflare))
		}
		section := cfg.Cloudflare[0]
		if section.APIToken != "token" || section.ZoneName != "example.com" || section.RecordName != "ddns.example.com" {
			t.Errorf("Load() = %+v, fields do not match file contents", section)
		}
		if section.Mode != "delete" || section.Type != "AAAA" || section.TTL != 300 {
			t.Errorf("Load() section mode/type/ttl = %q/%q/%d, want delete/AAAA/300", section.Mode, section.Type, section.TTL)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		if _, err := Load(filepath.Join(t.TempDir(), "nope.json")); err == nil {
			t.Error("Load() error = nil, want error for missing file")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
			t.Fatal(err)
		}

		if _, err := Load(path); err == nil {
			t.Error("Load() error = nil, want error for invalid JSON")
		}
	})
}

func TestExample(t *testing.T) {
	data, err := Example()
	if err != nil {
		t.Fatalf("Example() error = %v", err)
	}

	var cfg Config
	if err := json.Unmarshal([]byte(data), &cfg); err != nil {
		t.Fatalf("Example() output is not valid JSON: %v", err)
	}

	want := Config{
		IP:             "auto",
		UpdateInterval: new(defaultUpdateIntervalSeconds),
		LogLevel:       "info",
		Cloudflare: []CloudflareConfig{
			{
				APIToken:   "your-cloudflare-api-token",
				ZoneName:   "example.com",
				RecordName: "ddns.example.com",
				Mode:       "upsert",
				Type:       "A_AAAA",
				TTL:        1,
			},
		},
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("Example() = %+v, want %+v", cfg, want)
	}
}
