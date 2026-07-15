package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestRecorderCapturesFirstWriteHeader(t *testing.T) {
	rec := &recorder{ResponseWriter: httptest.NewRecorder(), status: http.StatusOK}

	if rec.status != http.StatusOK {
		t.Fatalf("default status = %d, want 200", rec.status)
	}

	rec.WriteHeader(503)
	rec.WriteHeader(200) // a second write must not overwrite the first
	if rec.status != 503 {
		t.Fatalf("status = %d, want 503 (first write wins)", rec.status)
	}
}

func TestInstrumentRecordsRequest(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /probe", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	h := Instrument(mux)

	counter := requests.WithLabelValues("GET /probe", "GET", "204")
	before := testutil.ToFloat64(counter)
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/probe", nil))
	after := testutil.ToFloat64(counter)

	if after-before != 1 {
		t.Fatalf("counter delta = %v, want 1", after-before)
	}
}

func TestInstrumentSkipsMetricsPath(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", Handler())
	h := Instrument(mux)

	counter := requests.WithLabelValues("GET /metrics", "GET", "200")
	before := testutil.ToFloat64(counter)
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/metrics", nil))
	after := testutil.ToFloat64(counter)

	if after != before {
		t.Fatalf("the metrics path was recorded: delta %v", after-before)
	}
}
