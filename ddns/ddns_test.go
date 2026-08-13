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
	mode    string
	types   []string
	upserts []upsertCall
	deletes []string
}

func (p *stubProvider) Mode() string { return p.mode }

func (p *stubProvider) Types() []string { return p.types }

func (p *stubProvider) Upsert(recordType, ip string) error {
	p.upserts = append(p.upserts, upsertCall{recordType: recordType, ip: ip})
	return nil
}

func (p *stubProvider) Delete(recordType string) error {
	p.deletes = append(p.deletes, recordType)
	return nil
}

func TestRunnerRunUpsert(t *testing.T) {
	tests := []struct {
		name     string
		cfg      config.Config
		provider *stubProvider
		want     []upsertCall
	}{
		{
			name:     "A record with static IP",
			cfg:      config.Config{IP: "203.0.113.1"},
			provider: &stubProvider{mode: "upsert", types: []string{"A"}},
			want:     []upsertCall{{recordType: "A", ip: "203.0.113.1"}},
		},
		{
			name:     "A_AAAA with dual static IPs",
			cfg:      config.Config{IP: "203.0.113.1,2001:db8::1"},
			provider: &stubProvider{mode: "upsert", types: []string{"A", "AAAA"}},
			want: []upsertCall{
				{recordType: "A", ip: "203.0.113.1"},
				{recordType: "AAAA", ip: "2001:db8::1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := New([]Provider{tt.provider}, &tt.cfg)

			runner.Run()

			if !reflect.DeepEqual(tt.provider.upserts, tt.want) {
				t.Errorf("Run() upsert calls = %+v, want %+v", tt.provider.upserts, tt.want)
			}
		})
	}
}

func TestRunnerRunMixedModes(t *testing.T) {
	// upsert Provider 处理 A，delete Provider 处理 AAAA；
	// 一次 Run() 内按各自 mode 分派，WAN IP 每种类型只检测一次
	cfg := config.Config{IP: "203.0.113.1,2001:db8::1"}
	upserter := &stubProvider{mode: "upsert", types: []string{"A"}}
	deleter := &stubProvider{mode: "delete", types: []string{"AAAA"}}
	runner := New([]Provider{upserter, deleter}, &cfg)

	runner.Run()

	wantUpserts := []upsertCall{{recordType: "A", ip: "203.0.113.1"}}
	if !reflect.DeepEqual(upserter.upserts, wantUpserts) {
		t.Errorf("upsert provider calls = %+v, want %+v", upserter.upserts, wantUpserts)
	}
	if !reflect.DeepEqual(deleter.deletes, []string{"AAAA"}) {
		t.Errorf("delete provider calls = %+v, want %+v", deleter.deletes, []string{"AAAA"})
	}
	if len(upserter.deletes) != 0 || len(deleter.upserts) != 0 {
		t.Errorf("mode dispatch leaked: upsert provider deletes = %+v, delete provider upserts = %+v",
			upserter.deletes, deleter.upserts)
	}
}

func TestRunnerRunDelete(t *testing.T) {
	tests := []struct {
		name     string
		cfg      config.Config
		provider *stubProvider
		want     []string
	}{
		{
			name:     "A record",
			cfg:      config.Config{},
			provider: &stubProvider{mode: "delete", types: []string{"A"}},
			want:     []string{"A"},
		},
		{
			name:     "A_AAAA deletes both types",
			cfg:      config.Config{},
			provider: &stubProvider{mode: "delete", types: []string{"A", "AAAA"}},
			want:     []string{"A", "AAAA"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := New([]Provider{tt.provider}, &tt.cfg)

			runner.Run()

			if !reflect.DeepEqual(tt.provider.deletes, tt.want) {
				t.Errorf("Run() delete calls = %+v, want %+v", tt.provider.deletes, tt.want)
			}
		})
	}
}
