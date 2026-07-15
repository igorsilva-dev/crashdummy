package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/igorsilva-dev/crashdummy/app/chaos"
	"github.com/igorsilva-dev/crashdummy/app/models"
)

// newMux resets the shared route registry and returns a fresh mux, so each
// test starts from a clean fault configuration.
func newMux(t *testing.T) *http.ServeMux {
	t.Helper()
	routes = map[string]*chaos.Chaos{}
	return http.NewServeMux()
}

func TestRegisterMappingHonorsMethodAndStatus(t *testing.T) {
	mux := newMux(t)
	registerMapping(mux, models.Mapping{
		Request:        models.Request{Method: "POST", URL: "/orders"},
		Response:       models.Response{Status: 201},
		MappedResponse: `{"ok":true}`,
	})

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest("POST", "/orders", nil))

	if rr.Code != 201 {
		t.Fatalf("status = %d, want 201", rr.Code)
	}
	if got := rr.Header().Get("Chaos-Type"); got != "mock" {
		t.Fatalf("Chaos-Type = %q, want mock", got)
	}
	if got := strings.TrimSpace(rr.Body.String()); got != `{"ok":true}` {
		t.Fatalf("body = %q", got)
	}
}

func TestRegisterMappingWrongMethodReturns405(t *testing.T) {
	mux := newMux(t)
	registerMapping(mux, models.Mapping{
		Request:  models.Request{Method: "POST", URL: "/orders"},
		Response: models.Response{Status: 201},
	})

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest("GET", "/orders", nil))

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rr.Code)
	}
}

func TestRegisterMappingInjectsError(t *testing.T) {
	mux := newMux(t)
	registerMapping(mux, models.Mapping{
		Request:        models.Request{Method: "GET", URL: "/health"},
		Response:       models.Response{Status: 200},
		ErrorRate:      1,
		ErrorStatus:    503,
		MappedResponse: `{}`,
	})

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest("GET", "/health", nil))

	if rr.Code != 503 {
		t.Fatalf("status = %d, want 503", rr.Code)
	}
	if got := rr.Header().Get("Chaos-Type"); got != "error" {
		t.Fatalf("Chaos-Type = %q, want error", got)
	}
}

func TestRegisterProxyPassesUpstreamThrough(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte(`{"tea":true}`))
	}))
	defer upstream.Close()

	mux := newMux(t)
	registerProxy(mux, models.Proxy{Path: "/tea", Method: "GET", Upstream: upstream.URL})

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest("GET", "/tea", nil))

	if rr.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want 418", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"tea":true`) {
		t.Fatalf("body = %q", rr.Body.String())
	}
}

func TestRegisterProxyInjectedErrorSkipsUpstream(t *testing.T) {
	called := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer upstream.Close()

	mux := newMux(t)
	registerProxy(mux, models.Proxy{
		Path: "/tea", Method: "GET", Upstream: upstream.URL,
		ErrorRate: 1, ErrorStatus: 500,
	})

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest("GET", "/tea", nil))

	if rr.Code != 500 {
		t.Fatalf("status = %d, want 500", rr.Code)
	}
	if called {
		t.Fatal("upstream was called despite an injected error")
	}
}

func TestRegisterProxyUpstreamFailureReturns502(t *testing.T) {
	mux := newMux(t)
	// Port 1 is not listening; the upstream call fails immediately.
	registerProxy(mux, models.Proxy{Path: "/tea", Method: "GET", Upstream: "http://127.0.0.1:1"})

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest("GET", "/tea", nil))

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", rr.Code)
	}
}

func TestHandleAdminChaosUpdatesRouteAtRuntime(t *testing.T) {
	mux := newMux(t)
	registerMapping(mux, models.Mapping{
		Request:        models.Request{Method: "GET", URL: "/health"},
		Response:       models.Response{Status: 200},
		MappedResponse: `{}`,
	})
	mux.HandleFunc("POST /admin/chaos", handleAdminChaos)

	body := `{"route":"GET /health","errorRate":1,"errorStatus":503}`
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest("POST", "/admin/chaos", strings.NewReader(body)))
	if rr.Code != http.StatusOK {
		t.Fatalf("admin status = %d, want 200", rr.Code)
	}

	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, httptest.NewRequest("GET", "/health", nil))
	if rr2.Code != 503 {
		t.Fatalf("after admin update, status = %d, want 503", rr2.Code)
	}
}

func TestHandleAdminChaosErrors(t *testing.T) {
	mux := newMux(t)
	mux.HandleFunc("POST /admin/chaos", handleAdminChaos)

	cases := []struct {
		name string
		body string
		want int
	}{
		{"unknown route", `{"route":"GET /nope"}`, http.StatusNotFound},
		{"missing route", `{"errorRate":1}`, http.StatusBadRequest},
		{"bad json", `not-json`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, httptest.NewRequest("POST", "/admin/chaos", strings.NewReader(tc.body)))
			if rr.Code != tc.want {
				t.Fatalf("status = %d, want %d", rr.Code, tc.want)
			}
		})
	}
}

func TestLoadMappingValidation(t *testing.T) {
	dir := t.TempDir()
	mDir := filepath.Join(dir, "mappings")
	sDir := filepath.Join(dir, "stubs")
	mustMkdir(t, mDir)
	mustMkdir(t, sDir)
	mustWrite(t, filepath.Join(sDir, "body.json"), `{"ok":true}`)

	prevM, prevS := mappingsDir, stubsDir
	mappingsDir, stubsDir = mDir, sDir
	t.Cleanup(func() { mappingsDir, stubsDir = prevM, prevS })

	cases := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{"valid, method normalized", `{"request":{"method":"post","url":"/x"},"response":{"status":201,"bodyFileName":"body.json"}}`, false},
		{"bad method", `{"request":{"method":"FLY","url":"/x"},"response":{"status":200,"bodyFileName":"body.json"}}`, true},
		{"status out of range", `{"request":{"method":"GET","url":"/x"},"response":{"status":700,"bodyFileName":"body.json"}}`, true},
		{"missing url", `{"request":{"method":"GET","url":""},"response":{"status":200,"bodyFileName":"body.json"}}`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mustWrite(t, filepath.Join(mDir, "m.json"), tc.json)
			_, err := loadMapping("m.json")
			if tc.wantErr && err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestLoadMappingsSkipsConfigMapHiddenEntries(t *testing.T) {
	dir := t.TempDir()
	mDir := filepath.Join(dir, "mappings")
	sDir := filepath.Join(dir, "stubs")
	mustMkdir(t, mDir)
	mustMkdir(t, sDir)
	mustWrite(t, filepath.Join(sDir, "body.json"), `{"ok":true}`)
	mustWrite(t, filepath.Join(mDir, "health.json"),
		`{"request":{"method":"GET","url":"/health"},"response":{"status":200,"bodyFileName":"body.json"}}`)

	// Reproduce what a Kubernetes ConfigMap volume mount looks like: a
	// timestamped hidden directory and a dot-prefixed entry alongside the
	// real files. The loader must skip these, not read them as config.
	mustMkdir(t, filepath.Join(mDir, "..2026_07_15_15_56_10.3475533494"))
	mustWrite(t, filepath.Join(mDir, ".hidden"), "ignored")

	prevM, prevS := mappingsDir, stubsDir
	mappingsDir, stubsDir = mDir, sDir
	t.Cleanup(func() { mappingsDir, stubsDir = prevM, prevS })

	mappings, err := loadMappings()
	if err != nil {
		t.Fatalf("loadMappings() error: %v", err)
	}
	if len(mappings) != 1 {
		t.Fatalf("loaded %d mappings, want 1 (hidden/dir entries must be skipped)", len(mappings))
	}
	if mappings[0].Request.URL != "/health" {
		t.Fatalf("unexpected mapping loaded: %+v", mappings[0])
	}
}

func TestLoadProxiesSkipsConfigMapHiddenEntries(t *testing.T) {
	dir := t.TempDir()
	pDir := filepath.Join(dir, "proxies")
	mustMkdir(t, pDir)
	mustWrite(t, filepath.Join(pDir, "up.json"),
		`{"path":"/up","upstream":"http://example.com","method":"GET"}`)
	mustMkdir(t, filepath.Join(pDir, "..2026_07_15_15_56_10.3475533494"))
	mustWrite(t, filepath.Join(pDir, ".hidden"), "ignored")

	prev := proxiesDir
	proxiesDir = pDir
	t.Cleanup(func() { proxiesDir = prev })

	proxies, err := loadProxies()
	if err != nil {
		t.Fatalf("loadProxies() error: %v", err)
	}
	if len(proxies) != 1 {
		t.Fatalf("loaded %d proxies, want 1 (hidden/dir entries must be skipped)", len(proxies))
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
