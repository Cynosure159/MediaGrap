package webhooks

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mediagrap/mediagrap/internal/platform/database"
)

func fixture(t *testing.T) (*Service, Created) {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err = database.Migrate(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO sources(id,name,root_path) VALUES(1,'one','/private/root')`); err != nil {
		t.Fatal(err)
	}
	key := filepath.Join(t.TempDir(), "key")
	if err = os.WriteFile(key, []byte(strings.Repeat("01", 32)), 0600); err != nil {
		t.Fatal(err)
	}
	s := New(db, Config{KeyFile: key}, nil)
	created, err := s.Create(t.Context(), Input{Name: "test", URL: "https://8.8.8.8/hook?private=credential", Enabled: true, EventTypes: []string{"job.succeeded"}, SourceIDs: []int64{1}})
	if err != nil {
		t.Fatal(err)
	}
	return s, created
}
func addEvent(t *testing.T, s *Service, id string, source int64) {
	t.Helper()
	now := time.Now().UnixMilli()
	body := []byte(`{"id":"` + id + `","type":"job.succeeded","data":{"sourceId":1}}`)
	if _, err := s.db.Exec(`INSERT INTO system_events(id,event_type,aggregate_id,aggregate_version,source_id,payload_bytes,occurred_at) VALUES(?,'job.succeeded',1,?,?,?,?)`, id, now, source, body, now); err != nil {
		t.Fatal(err)
	}
}

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestDispatchCrashReplayAndSignature(t *testing.T) {
	s, created := fixture(t)
	addEvent(t, s, "evt_one", 1)
	if err := s.Dispatch(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := s.Dispatch(t.Context()); err != nil {
		t.Fatal(err)
	}
	c, err := s.claim(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.claim(t.Context()); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("endpoint lease not exclusive: %v", err)
	}
	var bodies []string
	s.newClient = func(context.Context, string) (*http.Client, error) {
		return &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
			body, _ := io.ReadAll(r.Body)
			bodies = append(bodies, string(body))
			if r.Header.Get("X-MediaGrap-Signature") != Signature([]byte(created.Secret), r.Header.Get("X-MediaGrap-Timestamp"), body) {
				t.Fatal("invalid HMAC")
			}
			return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("sensitive response"))}, nil
		})}, nil
	}
	// Remote accepted, process lost ownership before it could commit success.
	if _, err = s.db.Exec(`UPDATE webhook_deliveries SET lease_owner='other',lease_expires_at=0`); err != nil {
		t.Fatal(err)
	}
	if err = s.deliver(t.Context(), c); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale owner wrote result: %v", err)
	}
	recovered, err := s.claim(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if recovered.id != c.id || string(recovered.body) != string(c.body) {
		t.Fatal("replay changed event")
	}
	if err = s.deliver(t.Context(), recovered); err != nil {
		t.Fatal(err)
	}
	if len(bodies) != 2 || bodies[0] != bodies[1] {
		t.Fatal("replay not identical")
	}
	items, err := s.Deliveries(t.Context(), created.Endpoint.ID, 0)
	if err != nil || len(items) != 1 || items[0].State != "succeeded" || items[0].Attempts != 2 {
		t.Fatalf("deliveries: %+v %v", items, err)
	}
	listing, _ := s.List(t.Context())
	encoded, _ := json.Marshal(listing)
	if strings.Contains(string(encoded), created.Secret) || strings.Contains(string(encoded), "credential") {
		t.Fatal("secret leaked")
	}
}
func TestFanoutRollsBackAndSourceIsolation(t *testing.T) {
	s, _ := fixture(t)
	addEvent(t, s, "evt_one", 1)
	if _, err := s.db.Exec(`CREATE TRIGGER reject_delivery BEFORE INSERT ON webhook_deliveries BEGIN SELECT RAISE(ABORT,'injected'); END`); err != nil {
		t.Fatal(err)
	}
	if err := s.Dispatch(t.Context()); err == nil {
		t.Fatal("expected error")
	}
	var dispatched *int64
	s.db.QueryRow(`SELECT dispatched_at FROM system_events`).Scan(&dispatched)
	if dispatched != nil {
		t.Fatal("fanout escaped rollback")
	}
	s.db.Exec(`DROP TRIGGER reject_delivery`)
	s.db.Exec(`UPDATE system_events SET source_id=2`)
	if err := s.Dispatch(t.Context()); err != nil {
		t.Fatal(err)
	}
	var count int
	s.db.QueryRow(`SELECT COUNT(*) FROM webhook_deliveries`).Scan(&count)
	if count != 0 {
		t.Fatal("source scope bypass")
	}
}
func TestPauseRotateRetryAndTestEvent(t *testing.T) {
	s, created := fixture(t)
	id, err := s.Test(t.Context(), created.Endpoint.ID)
	if err != nil {
		t.Fatal(err)
	}
	rotated, err := s.Rotate(t.Context(), created.Endpoint.ID)
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.claim(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if string(c.secret) != rotated["secret"] || c.kind != "webhook.test" || c.id != id {
		t.Fatal("rotation/test mismatch")
	}
	s.newClient = func(context.Context, string) (*http.Client, error) {
		return &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 400, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(""))}, nil
		})}, nil
	}
	if err = s.deliver(t.Context(), c); err != nil {
		t.Fatal(err)
	}
	if err = s.Retry(t.Context(), created.Endpoint.ID, id); err != nil {
		t.Fatal(err)
	}
	if err = s.Retry(t.Context(), created.Endpoint.ID, id); !errors.Is(err, ErrConflict) {
		t.Fatal("concurrent retry accepted")
	}
	e, _ := s.Get(t.Context(), created.Endpoint.ID)
	if err = s.Update(t.Context(), e.ID, Input{Name: e.Name, Enabled: false, SourceIDs: e.SourceIDs, EventTypes: e.EventTypes, Version: e.Version}); err != nil {
		t.Fatal(err)
	}
	items, _ := s.Deliveries(t.Context(), e.ID, 0)
	if items[0].State != "cancelled" {
		t.Fatal("pause did not cancel queue")
	}
	if err = s.Retry(t.Context(), e.ID, id); !errors.Is(err, ErrConflict) {
		t.Fatal("paused retry allowed")
	}
}
func TestRetryClassification(t *testing.T) {
	for _, status := range []int{200, 204, 301, 400, 401, 408, 429, 500, 503} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			s, e := fixture(t)
			_, err := s.Test(t.Context(), e.Endpoint.ID)
			if err != nil {
				t.Fatal(err)
			}
			s.newClient = func(context.Context, string) (*http.Client, error) {
				return &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: status, Header: http.Header{"Retry-After": []string{"120"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
				})}, nil
			}
			c, err := s.claim(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if err = s.deliver(t.Context(), c); err != nil {
				t.Fatal(err)
			}
			items, _ := s.Deliveries(t.Context(), e.Endpoint.ID, 0)
			want := "dead"
			if status >= 200 && status < 300 {
				want = "succeeded"
			} else if status == 408 || status == 429 || status >= 500 {
				want = "retry_wait"
			}
			if items[0].State != want {
				t.Fatalf("%d: %s", status, items[0].State)
			}
		})
	}
}
func TestNetworkPolicy(t *testing.T) {
	p := NetworkPolicy{}
	for _, raw := range []string{"file:///etc/passwd", "http://8.8.8.8", "https://user:pass@8.8.8.8", "https://127.0.0.1", "https://[::1]", "https://169.254.169.254", "https://[::ffff:127.0.0.1]", "https://10.0.0.1", "https://100.100.100.200"} {
		if _, _, err := p.validate(t.Context(), raw); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	p = NetworkPolicy{AllowHTTP: true, AllowedTargets: []string{"10.0.0.1:8080", "100.100.100.200:80"}, AllowedCIDRs: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8"), netip.MustParsePrefix("100.64.0.0/10")}}
	if _, _, err := p.validate(t.Context(), "http://10.0.0.1:8080"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := p.validate(t.Context(), "http://100.100.100.200"); err == nil {
		t.Fatal("metadata allowlist bypass")
	}
	p.lookup = func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}, {IP: net.ParseIP("127.0.0.1")}}, nil
	}
	if _, _, err := p.validate(t.Context(), "https://rebind.example"); err == nil {
		t.Fatal("mixed DNS result accepted")
	}
	p.lookup = func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}}, nil
	}
	client, err := p.client(t.Context(), "https://rebind.example")
	if err != nil {
		t.Fatal(err)
	}
	if client.CheckRedirect(nil, nil) != http.ErrUseLastResponse {
		t.Fatal("redirect enabled")
	}
	if client.Transport.(*http.Transport).Proxy != nil {
		t.Fatal("proxy inherited")
	}
}
func TestKeyAndRetention(t *testing.T) {
	s, e := fixture(t)
	if _, err := decrypt(s.cipher, "wrong", encrypt(s.cipher, "right", []byte("secret"))); err == nil {
		t.Fatal("AAD not bound")
	}
	id, err := s.Test(t.Context(), e.Endpoint.ID)
	if err != nil {
		t.Fatal(err)
	}
	s.db.Exec(`UPDATE system_events SET occurred_at=0`)
	if err = s.cleanup(t.Context()); err != nil {
		t.Fatal(err)
	}
	var n int
	s.db.QueryRow(`SELECT COUNT(*) FROM webhook_deliveries WHERE id=?`, id).Scan(&n)
	if n != 1 {
		t.Fatal("pending deleted")
	}
}

func TestDeletedSourceSubscriptionDoesNotFollowReusedID(t *testing.T) {
	s, _ := fixture(t)
	if _, err := s.db.Exec(`DELETE FROM sources WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`INSERT INTO sources(id,name,root_path) VALUES(1,'new source','/different-root')`); err != nil {
		t.Fatal(err)
	}
	addEvent(t, s, "new-source-event", 1)
	if err := s.Dispatch(t.Context()); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM webhook_deliveries`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("stale source subscription")
	}
}

func TestSignatureKnownVectorAndRetryAfter(t *testing.T) {
	body := []byte(`{"id":"evt_1"}`)
	want := "sha256=c5ec2a07e4d006f47bccf98e08c35205f5930f2ac806f975cba53c07a6eba300"
	if got := Signature([]byte("test-secret"), "1710000000", body); got != want {
		t.Fatalf("signature %s", got)
	}
	if Signature([]byte("test-secret"), "1710000000", append(body, ' ')) == want {
		t.Fatal("raw body mutation ignored")
	}
	now := time.Now()
	if delay := retryDelay(1, "120", now); delay < 2*time.Minute {
		t.Fatal("Retry-After ignored")
	}
	if delay := retryDelay(1, "2000000000", now); delay != 24*time.Hour {
		t.Fatal("Retry-After unbounded")
	}
}

func TestDNSUnavailableRetriesWithoutLeakingURL(t *testing.T) {
	s, e := fixture(t)
	if _, err := s.Test(t.Context(), e.Endpoint.ID); err != nil {
		t.Fatal(err)
	}
	c, err := s.claim(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	s.newClient = func(context.Context, string) (*http.Client, error) { return nil, ErrResolution }
	if err = s.deliver(t.Context(), c); err != nil {
		t.Fatal(err)
	}
	items, err := s.Deliveries(t.Context(), e.Endpoint.ID, 0)
	if err != nil || items[0].State != "retry_wait" || items[0].ErrorCode != "DNS_UNAVAILABLE" {
		t.Fatalf("DNS result %+v %v", items, err)
	}
}
