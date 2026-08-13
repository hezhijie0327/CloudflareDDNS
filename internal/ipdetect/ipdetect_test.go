package ipdetect

import (
	"encoding/binary"
	"reflect"
	"strings"
	"testing"
)

// answerResponse builds a DNS response whose single answer record carries
// the given type and RDATA, in reply to a whoami.cloudflare CH TXT question.
func answerResponse(id, flags, recordType uint16, rdata []byte) []byte {
	resp := make([]byte, 0, 64+len(rdata))
	resp = binary.BigEndian.AppendUint16(resp, id)
	resp = binary.BigEndian.AppendUint16(resp, flags)
	resp = binary.BigEndian.AppendUint16(resp, 1) // QDCOUNT
	resp = binary.BigEndian.AppendUint16(resp, 1) // ANCOUNT
	resp = binary.BigEndian.AppendUint16(resp, 0) // NSCOUNT
	resp = binary.BigEndian.AppendUint16(resp, 0) // ARCOUNT

	// 问题段：whoami.cloudflare + TXT(16) + CH(3)
	resp = append(resp, 0x06)
	resp = append(resp, "whoami"...)
	resp = append(resp, 0x0A)
	resp = append(resp, "cloudflare"...)
	resp = append(resp, 0x00)
	resp = binary.BigEndian.AppendUint16(resp, 16)
	resp = binary.BigEndian.AppendUint16(resp, 3)

	// 答案段：压缩指针指向问题段名称 + Type + Class IN + TTL + RDATA
	resp = binary.BigEndian.AppendUint16(resp, 0xC00C)
	resp = binary.BigEndian.AppendUint16(resp, recordType)
	resp = binary.BigEndian.AppendUint16(resp, 1)
	resp = binary.BigEndian.AppendUint32(resp, 0)
	resp = binary.BigEndian.AppendUint16(resp, uint16(len(rdata))) //nolint:gosec // crafted test payload, tiny fixed sizes
	resp = append(resp, rdata...)

	return resp
}

// txtResponse builds a DNS response whose answer carries the given TXT strings.
func txtResponse(id, flags uint16, txts []string) []byte {
	var rdata []byte
	for _, txt := range txts {
		rdata = append(rdata, byte(len(txt))) //nolint:gosec // crafted test payload, tiny fixed sizes
		rdata = append(rdata, txt...)
	}
	return answerResponse(id, flags, 16, rdata) // TXT
}

func TestBuildDNSQuery(t *testing.T) {
	const id = uint16(0x1234)
	query := buildDNSQuery(id, "whoami.cloudflare")

	if len(query) < 12 {
		t.Fatalf("query length = %d, want >= 12", len(query))
	}

	if got := binary.BigEndian.Uint16(query[0:2]); got != id {
		t.Errorf("ID = %#x, want %#x", got, id)
	}
	if got := binary.BigEndian.Uint16(query[2:4]); got != 0x0100 {
		t.Errorf("flags = %#x, want RD (0x0100)", got)
	}
	if got := binary.BigEndian.Uint16(query[4:6]); got != 1 {
		t.Errorf("QDCOUNT = %d, want 1", got)
	}

	// 问题段：whoami.cloudflare + TXT(16) + CH(3)
	question := query[12:]
	want := []byte{0x06}
	want = append(want, "whoami"...)
	want = append(want, 0x0A)
	want = append(want, "cloudflare"...)
	want = append(want, 0x00, 0x00, 0x10, 0x00, 0x03)
	if !reflect.DeepEqual(question, want) {
		t.Errorf("question = %v, want %v", question, want)
	}
}

func TestSkipDNSName(t *testing.T) {
	tests := []struct {
		name string
		msg  []byte
		want int
	}{
		{
			name: "simple name",
			msg:  []byte{3, 'w', 'w', 'w', 3, 'c', 'o', 'm', 0, 0xAA, 0xBB},
			want: 9,
		},
		{
			name: "compression pointer",
			msg:  []byte{0xC0, 0x0C, 0xAA, 0xBB},
			want: 2,
		},
		{
			name: "invalid label length",
			msg:  []byte{64, 'x'},
			want: -1,
		},
		{
			name: "truncated name",
			msg:  []byte{5, 'a', 'b'},
			want: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := skipDNSName(tt.msg, 0); got != tt.want {
				t.Errorf("skipDNSName() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestParseDNSResponse(t *testing.T) {
	const id = uint16(0xBEEF)

	tests := []struct {
		name    string
		resp    []byte
		want    string
		wantErr string
	}{
		{
			name: "single TXT",
			resp: txtResponse(id, 0x8180, []string{"203.0.113.1"}),
			want: "203.0.113.1",
		},
		{
			name: "multi-string TXT joined",
			resp: txtResponse(id, 0x8180, []string{"203.0.", "113.1"}),
			want: "203.0.113.1",
		},
		{
			name:    "response too short",
			resp:    []byte{0x00, 0x01, 0x02},
			wantErr: "too short",
		},
		{
			name:    "ID mismatch",
			resp:    txtResponse(id+1, 0x8180, []string{"203.0.113.1"}),
			wantErr: "ID mismatch",
		},
		{
			name:    "not a response",
			resp:    txtResponse(id, 0x0100, []string{"203.0.113.1"}),
			wantErr: "not a DNS response",
		},
		{
			name:    "rcode error",
			resp:    txtResponse(id, 0x8183, []string{"203.0.113.1"}),
			wantErr: "rcode=3",
		},
		{
			name:    "no TXT records",
			resp:    answerResponse(id, 0x8180, 1, []byte{203, 0, 113, 1}), // A 记录
			wantErr: "no TXT record",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDNSResponse(tt.resp, id)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("parseDNSResponse() error = nil, want containing %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("parseDNSResponse() error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseDNSResponse() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("parseDNSResponse() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidIP(t *testing.T) {
	tests := []struct {
		name       string
		ip         string
		recordType string
		want       bool
	}{
		{name: "IPv4 for A", ip: "203.0.113.1", recordType: "A", want: true},
		{name: "IPv6 for A rejected", ip: "2001:db8::1", recordType: "A", want: false},
		{name: "IPv6 for AAAA", ip: "2001:db8::1", recordType: "AAAA", want: true},
		{name: "IPv4 for AAAA rejected", ip: "203.0.113.1", recordType: "AAAA", want: false},
		{name: "garbage rejected", ip: "not-an-ip", recordType: "A", want: false},
		{name: "empty rejected", ip: "", recordType: "A", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validIP(tt.ip, tt.recordType); got != tt.want {
				t.Errorf("validIP(%q, %q) = %v, want %v", tt.ip, tt.recordType, got, tt.want)
			}
		})
	}
}

func TestStaticIP(t *testing.T) {
	tests := []struct {
		name       string
		staticIP   string
		recordType string
		want       string
		wantErr    bool
	}{
		{name: "single IPv4", staticIP: "203.0.113.1", recordType: "A", want: "203.0.113.1"},
		{name: "dual takes IPv4", staticIP: "203.0.113.1,2001:db8::1", recordType: "A", want: "203.0.113.1"},
		{name: "dual takes IPv6", staticIP: "203.0.113.1,2001:db8::1", recordType: "AAAA", want: "2001:db8::1"},
		{name: "single IPv6", staticIP: "2001:db8::1", recordType: "AAAA", want: "2001:db8::1"},
		{name: "family mismatch", staticIP: "2001:db8::1", recordType: "A", wantErr: true},
		{name: "garbage", staticIP: "nope", recordType: "A", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New(tt.staticIP)
			got, err := d.staticIP(tt.recordType)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("staticIP() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("staticIP() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("staticIP() = %q, want %q", got, tt.want)
			}
		})
	}
}
