package log

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseLevelFilter(t *testing.T) {
	tests := []struct {
		name         string
		in           string
		wantLevel    Level
		wantComps    []string
		wantCompsNil bool
	}{
		{name: "empty uses default", in: "", wantLevel: Info, wantCompsNil: true},
		{name: "plain level", in: "info", wantLevel: Info, wantCompsNil: true},
		{name: "level with components", in: "debug:CLOUDFLARE,IPDETECT", wantLevel: Debug, wantComps: []string{"CLOUDFLARE", "IPDETECT"}},
		{name: "level with empty component list", in: "debug:", wantLevel: Debug, wantCompsNil: true},
		{name: "bogus level falls back", in: "bogus", wantLevel: Info, wantCompsNil: true},
		{name: "component casing normalized", in: "warn:cloudflare", wantLevel: Warn, wantComps: []string{"cloudflare"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lvl, comps := ParseLevelFilter(tt.in, Info)
			if lvl != tt.wantLevel {
				t.Errorf("ParseLevelFilter(%q) level = %v, want %v", tt.in, lvl, tt.wantLevel)
			}
			if tt.wantCompsNil {
				if comps != nil {
					t.Errorf("ParseLevelFilter(%q) components = %v, want nil", tt.in, comps)
				}
				return
			}
			if strings.Join(comps, ",") != strings.Join(tt.wantComps, ",") {
				t.Errorf("ParseLevelFilter(%q) components = %v, want %v", tt.in, comps, tt.wantComps)
			}
		})
	}
}

func TestExtractPrefix(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "prefixed message", in: "CLOUDFLARE: updating record", want: "CLOUDFLARE"},
		{name: "lowercase normalized", in: "ddns: checking", want: "DDNS"},
		{name: "no prefix", in: "plain message", want: ""},
		{name: "colon without space", in: "a:b", want: ""},
		{name: "colon at start", in: ": message", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractPrefix(tt.in); got != tt.want {
				t.Errorf("extractPrefix(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestComponentFilter(t *testing.T) {
	var buf bytes.Buffer
	m := NewLogger()
	m.writer = &buf
	m.SetComponentFilter([]string{"CLOUDFLARE"})
	m.SetLevel(Debug)

	m.Info("CLOUDFLARE: kept")
	m.Info("DDNS: filtered out")
	m.Error("DDNS: errors always pass")

	out := buf.String()
	if !strings.Contains(out, "CLOUDFLARE: kept") {
		t.Errorf("output = %q, want filtered-in Info message", out)
	}
	if strings.Contains(out, "DDNS: filtered out") {
		t.Errorf("output = %q, contains filtered-out Info message", out)
	}
	if !strings.Contains(out, "DDNS: errors always pass") {
		t.Errorf("output = %q, want Error message passing the filter", out)
	}
}
