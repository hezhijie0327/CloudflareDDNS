package cloudflare

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"zjddns/config"
)

// testCfg returns a fully populated Cloudflare config section.
func testCfg() *config.Config {
	return &config.Config{
		Cloudflare: &config.CloudflareConfig{
			APIToken:   "test-token",
			ZoneName:   "example.com",
			RecordName: "ddns.example.com",
			Mode:       config.ModeUpsert,
			Type:       config.TypeA,
			TTL:        1,
		},
	}
}

// newTestClient returns a Client pointed at an httptest server serving
// the given handler, with zoneID preset (as New would resolve it).
func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return &Client{
		httpClient: server.Client(),
		baseURL:    server.URL,
		cfg:        testCfg(),
		zoneID:     "zone-1",
	}
}

// jsonResponse writes a success envelope carrying the given raw result.
func jsonResponse(w http.ResponseWriter, result string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"success":true,"result":` + result + `,"errors":[]}`))
}

func TestZoneID(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/client/v4/zones" || r.URL.Query().Get("name") != "example.com" {
			t.Errorf("request = %s %s, want zone lookup for example.com", r.Method, r.URL.String())
		}
		jsonResponse(w, `[{"id":"zone-1","name":"example.com"}]`)
	})

	got, err := client.ZoneID()
	if err != nil {
		t.Fatalf("ZoneID() error = %v", err)
	}
	if got != "zone-1" {
		t.Errorf("ZoneID() = %q, want zone-1", got)
	}
}

func TestZoneIDNotFound(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		jsonResponse(w, `[]`)
	})

	if _, err := client.ZoneID(); !errors.Is(err, ErrZoneNotFound) {
		t.Errorf("ZoneID() error = %v, want ErrZoneNotFound", err)
	}
}

func TestRecordID(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if q := r.URL.Query(); q.Get("name") != "ddns.example.com" || q.Get("type") != "A" {
				t.Errorf("query = %s, want name=ddns.example.com&type=A", r.URL.RawQuery)
			}
			jsonResponse(w, `[{"id":"rec-1","type":"A","name":"ddns.example.com","content":"1.2.3.4"}]`)
		})

		got, err := client.RecordID("zone-1", "A")
		if err != nil {
			t.Fatalf("RecordID() error = %v", err)
		}
		if got != "rec-1" {
			t.Errorf("RecordID() = %q, want rec-1", got)
		}
	})

	t.Run("absent is not an error", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			jsonResponse(w, `[]`)
		})

		got, err := client.RecordID("zone-1", "A")
		if err != nil {
			t.Fatalf("RecordID() error = %v", err)
		}
		if got != "" {
			t.Errorf("RecordID() = %q, want empty", got)
		}
	})
}

func TestRecordContent(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/client/v4/zones/zone-1/dns_records/rec-1" {
				t.Errorf("path = %s, want record endpoint", r.URL.Path)
			}
			jsonResponse(w, `{"id":"rec-1","type":"A","content":"1.2.3.4"}`)
		})

		got, err := client.RecordContent("zone-1", "rec-1")
		if err != nil {
			t.Fatalf("RecordContent() error = %v", err)
		}
		if got != "1.2.3.4" {
			t.Errorf("RecordContent() = %q, want 1.2.3.4", got)
		}
	})

	t.Run("empty content", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			jsonResponse(w, `{"id":"rec-1","type":"A"}`)
		})

		if _, err := client.RecordContent("zone-1", "rec-1"); !errors.Is(err, ErrRecordNotFound) {
			t.Errorf("RecordContent() error = %v, want ErrRecordNotFound", err)
		}
	})
}

func TestCreateRecord(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/client/v4/zones/zone-1/dns_records" {
			t.Errorf("path = %s, want dns_records", r.URL.Path)
		}

		var sent Record
		if err := json.NewDecoder(r.Body).Decode(&sent); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		want := Record{Type: "A", Name: "ddns.example.com", Content: "1.2.3.4", TTL: 1}
		if sent != want {
			t.Errorf("payload = %+v, want %+v", sent, want)
		}

		jsonResponse(w, `{"id":"rec-1"}`)
	})

	if err := client.CreateRecord("zone-1", Record{Type: "A", Name: "ddns.example.com", Content: "1.2.3.4", TTL: 1}); err != nil {
		t.Fatalf("CreateRecord() error = %v", err)
	}
}

func TestUpdateRecord(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/client/v4/zones/zone-1/dns_records/rec-1" {
			t.Errorf("request = %s %s, want PUT record endpoint", r.Method, r.URL.Path)
		}
		jsonResponse(w, `{"id":"rec-1"}`)
	})

	if err := client.UpdateRecord("zone-1", "rec-1", Record{Type: "A", Name: "ddns.example.com", Content: "1.2.3.4", TTL: 1}); err != nil {
		t.Fatalf("UpdateRecord() error = %v", err)
	}
}

func TestDeleteRecord(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/client/v4/zones/zone-1/dns_records/rec-1" {
			t.Errorf("request = %s %s, want DELETE record endpoint", r.Method, r.URL.Path)
		}
		jsonResponse(w, `{"id":"rec-1"}`)
	})

	if err := client.DeleteRecord("zone-1", "rec-1"); err != nil {
		t.Fatalf("DeleteRecord() error = %v", err)
	}
}

func TestUpsert(t *testing.T) {
	const ip = "5.6.7.8"

	t.Run("creates when missing", func(t *testing.T) {
		posted := false
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				jsonResponse(w, `[]`) // no existing record
			case http.MethodPost:
				posted = true
				jsonResponse(w, `{"id":"rec-new"}`)
			default:
				t.Errorf("unexpected method %s", r.Method)
			}
		})

		if err := client.Upsert("A", ip); err != nil {
			t.Fatalf("Upsert() error = %v", err)
		}
		if !posted {
			t.Error("Upsert() did not create the record")
		}
	})

	t.Run("updates when content differs", func(t *testing.T) {
		updated := false
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				if strings.HasSuffix(r.URL.Path, "/rec-1") {
					jsonResponse(w, `{"id":"rec-1","content":"1.2.3.4"}`)
					return
				}
				jsonResponse(w, `[{"id":"rec-1"}]`)
			case http.MethodPut:
				updated = true
				var sent Record
				if err := json.NewDecoder(r.Body).Decode(&sent); err != nil {
					t.Fatalf("decode body: %v", err)
				}
				if sent.Content != ip {
					t.Errorf("update content = %q, want %q", sent.Content, ip)
				}
				jsonResponse(w, `{"id":"rec-1"}`)
			default:
				t.Errorf("unexpected method %s", r.Method)
			}
		})

		if err := client.Upsert("A", ip); err != nil {
			t.Fatalf("Upsert() error = %v", err)
		}
		if !updated {
			t.Error("Upsert() did not update the record")
		}
	})

	t.Run("skips when content unchanged", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				if strings.HasSuffix(r.URL.Path, "/rec-1") {
					jsonResponse(w, `{"id":"rec-1","content":"`+ip+`"}`)
					return
				}
				jsonResponse(w, `[{"id":"rec-1"}]`)
			default:
				t.Errorf("unexpected method %s, want no write", r.Method)
			}
		})

		if err := client.Upsert("A", ip); err != nil {
			t.Fatalf("Upsert() error = %v", err)
		}
	})
}

func TestDelete(t *testing.T) {
	t.Run("absent record", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("unexpected method %s, want no write", r.Method)
			}
			jsonResponse(w, `[]`)
		})

		if err := client.Delete("A"); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
	})

	t.Run("existing record", func(t *testing.T) {
		deleted := false
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				jsonResponse(w, `[{"id":"rec-1"}]`)
			case http.MethodDelete:
				deleted = true
				jsonResponse(w, `{"id":"rec-1"}`)
			default:
				t.Errorf("unexpected method %s", r.Method)
			}
		})

		if err := client.Delete("A"); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
		if !deleted {
			t.Error("Delete() did not delete the record")
		}
	})
}

func TestAuthHeaders(t *testing.T) {
	t.Run("API token", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
				t.Errorf("Authorization = %q, want Bearer test-token", got)
			}
			jsonResponse(w, `[]`)
		})

		if _, err := client.RecordID("zone-1", "A"); err != nil {
			t.Fatalf("RecordID() error = %v", err)
		}
	})

	t.Run("legacy keys", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("X-Auth-Email"); got != "a@b.c" {
				t.Errorf("X-Auth-Email = %q, want a@b.c", got)
			}
			if got := r.Header.Get("X-Auth-Key"); got != "secret" {
				t.Errorf("X-Auth-Key = %q, want secret", got)
			}
			if got := r.Header.Get("Authorization"); got != "" {
				t.Errorf("Authorization = %q, want empty", got)
			}
			jsonResponse(w, `[]`)
		}))
		t.Cleanup(server.Close)

		cfg := testCfg()
		cfg.Cloudflare.APIToken = ""
		cfg.Cloudflare.XAuthEmail = "a@b.c" //nolint:staticcheck // deprecated field under backward-compat test
		cfg.Cloudflare.XAuthKey = "secret"  //nolint:staticcheck // deprecated field under backward-compat test
		client := &Client{httpClient: server.Client(), baseURL: server.URL, cfg: cfg, zoneID: "zone-1"}

		if _, err := client.RecordID("zone-1", "A"); err != nil {
			t.Fatalf("RecordID() error = %v", err)
		}
	})
}
