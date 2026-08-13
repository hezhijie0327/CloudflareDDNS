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
			want: Config{Mode: "upsert", Type: "A", TTL: 1, IP: "auto", UpdateInterval: new(defaultUpdateIntervalSeconds)},
		},
		{
			name: "explicit values preserved",
			in:   Config{Mode: "delete", Type: "AAAA", TTL: 300, IP: "192.0.2.1", UpdateInterval: new(0)},
			want: Config{Mode: "delete", Type: "AAAA", TTL: 300, IP: "192.0.2.1", UpdateInterval: new(0)},
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
			Mode: "upsert",
			Type: "A",
			TTL:  1,
			Cloudflare: &CloudflareConfig{
				APIToken:   "token",
				ZoneName:   "example.com",
				RecordName: "ddns.example.com",
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
			name: "valid legacy auth",
			cfg:  Config{Mode: "upsert", Type: "A", TTL: 1, Cloudflare: &CloudflareConfig{XAuthEmail: "a@b.c", XAuthKey: "key", ZoneName: "example.com", RecordName: "ddns.example.com"}},
		},
		{
			name:    "no provider configured",
			cfg:     Config{Mode: "upsert", Type: "A", TTL: 1},
			wantErr: "no provider",
		},
		{
			name:    "missing authentication",
			cfg:     Config{Mode: "upsert", Type: "A", TTL: 1, Cloudflare: &CloudflareConfig{ZoneName: "example.com", RecordName: "ddns.example.com"}},
			wantErr: "authentication",
		},
		{
			name:    "missing zone name",
			cfg:     Config{Mode: "upsert", Type: "A", TTL: 1, Cloudflare: &CloudflareConfig{APIToken: "token", RecordName: "ddns.example.com"}},
			wantErr: "zone_name",
		},
		{
			name:    "missing record name",
			cfg:     Config{Mode: "upsert", Type: "A", TTL: 1, Cloudflare: &CloudflareConfig{APIToken: "token", ZoneName: "example.com"}},
			wantErr: "record_name",
		},
		{
			name:    "invalid mode",
			cfg:     Config{Mode: "bogus", Type: "A", TTL: 1, Cloudflare: &CloudflareConfig{APIToken: "token", ZoneName: "example.com", RecordName: "ddns.example.com"}},
			wantErr: "invalid mode",
		},
		{
			name:    "invalid type",
			cfg:     Config{Mode: "upsert", Type: "CNAME", TTL: 1, Cloudflare: &CloudflareConfig{APIToken: "token", ZoneName: "example.com", RecordName: "ddns.example.com"}},
			wantErr: "invalid type",
		},
		{
			name:    "invalid TTL",
			cfg:     Config{Mode: "upsert", Type: "A", TTL: 60, Cloudflare: &CloudflareConfig{APIToken: "token", ZoneName: "example.com", RecordName: "ddns.example.com"}},
			wantErr: "invalid TTL",
		},
		{
			name: "delete mode skips type and TTL checks",
			cfg:  Config{Mode: "delete", Cloudflare: &CloudflareConfig{APIToken: "token", ZoneName: "example.com", RecordName: "ddns.example.com"}},
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
		content := `{"type":"A","ttl":300,"cloudflare":{"api_token":"token","zone_name":"example.com","record_name":"ddns.example.com"}}`
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}

		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg.Cloudflare == nil {
			t.Fatal("Load() Cloudflare section = nil, want parsed section")
		}
		if cfg.Cloudflare.APIToken != "token" || cfg.Cloudflare.ZoneName != "example.com" || cfg.Cloudflare.RecordName != "ddns.example.com" || cfg.TTL != 300 {
			t.Errorf("Load() = %+v, fields do not match file contents", cfg)
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
		Type:           "A_AAAA",
		TTL:            1,
		IP:             "auto",
		Mode:           "upsert",
		UpdateInterval: new(defaultUpdateIntervalSeconds),
		Cloudflare: &CloudflareConfig{
			APIToken:   "your-cloudflare-api-token",
			ZoneName:   "example.com",
			RecordName: "ddns.example.com",
		},
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("Example() = %+v, want %+v", cfg, want)
	}
}
