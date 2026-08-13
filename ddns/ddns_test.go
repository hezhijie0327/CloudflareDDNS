package ddns

import (
	"reflect"
	"testing"
	"zjddns/config"
)

// upsertCall 记录一次 Upsert 调用的参数
type upsertCall struct {
	recordType string
	ip         string
}

// stubProvider 记录调用参数，不触网
type stubProvider struct {
	upserts []upsertCall
	deletes []string
}

func (p *stubProvider) Upsert(recordType, ip string) error {
	p.upserts = append(p.upserts, upsertCall{recordType: recordType, ip: ip})
	return nil
}

func (p *stubProvider) Delete(recordType string) error {
	p.deletes = append(p.deletes, recordType)
	return nil
}

func TestRunnerUpsert(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
		want []upsertCall
	}{
		{
			name: "A record with static IP",
			cfg:  config.Config{Type: "A", IP: "203.0.113.1"},
			want: []upsertCall{{recordType: "A", ip: "203.0.113.1"}},
		},
		{
			name: "A_AAAA with dual static IPs",
			cfg:  config.Config{Type: "A_AAAA", IP: "203.0.113.1,2001:db8::1"},
			want: []upsertCall{
				{recordType: "A", ip: "203.0.113.1"},
				{recordType: "AAAA", ip: "2001:db8::1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &stubProvider{}
			runner := New([]Provider{provider}, &tt.cfg)

			runner.Upsert()

			if !reflect.DeepEqual(provider.upserts, tt.want) {
				t.Errorf("Upsert() calls = %+v, want %+v", provider.upserts, tt.want)
			}
		})
	}
}

func TestRunnerUpsertMultipleProviders(t *testing.T) {
	cfg := config.Config{Type: "A", IP: "203.0.113.1"}
	first := &stubProvider{}
	second := &stubProvider{}
	runner := New([]Provider{first, second}, &cfg)

	runner.Upsert()

	want := []upsertCall{{recordType: "A", ip: "203.0.113.1"}}
	if !reflect.DeepEqual(first.upserts, want) {
		t.Errorf("first provider Upsert() calls = %+v, want %+v", first.upserts, want)
	}
	if !reflect.DeepEqual(second.upserts, want) {
		t.Errorf("second provider Upsert() calls = %+v, want %+v", second.upserts, want)
	}
}

func TestRunnerDelete(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
		want []string
	}{
		{
			name: "A record",
			cfg:  config.Config{Type: "A"},
			want: []string{"A"},
		},
		{
			name: "A_AAAA deletes both types",
			cfg:  config.Config{Type: "A_AAAA"},
			want: []string{"A", "AAAA"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &stubProvider{}
			runner := New([]Provider{provider}, &tt.cfg)

			runner.Delete()

			if !reflect.DeepEqual(provider.deletes, tt.want) {
				t.Errorf("Delete() calls = %+v, want %+v", provider.deletes, tt.want)
			}
		})
	}
}
