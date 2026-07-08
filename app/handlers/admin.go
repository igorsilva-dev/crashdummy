package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/igorsilva-dev/crashdummy/app/chaos"
)

// adminChaosRequest retunes the fault configuration of a single registered
// route. Route is the mapping pattern ("GET /health") or the proxy path.
type adminChaosRequest struct {
	Route                 string  `json:"route"`
	LatencyInMilliseconds int64   `json:"latencyInMilliseconds"`
	JitterInMilliseconds  int64   `json:"jitterInMilliseconds"`
	ErrorRate             float64 `json:"errorRate"`
	ErrorStatus           int     `json:"errorStatus"`
}

// handleAdminChaos updates a route's fault configuration at runtime so faults
// can be switched on during a demo without a redeploy. It is intended to stay
// internal to the cluster.
func handleAdminChaos(w http.ResponseWriter, r *http.Request) {
	var req adminChaosRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Route == "" {
		writeJSONError(w, http.StatusBadRequest, "route is required")
		return
	}

	fault, ok := routes[req.Route]
	if !ok {
		writeJSONError(w, http.StatusNotFound, "unknown route")
		return
	}

	fault.Update(chaos.Spec{
		LatencyInMilliseconds: req.LatencyInMilliseconds,
		JitterInMilliseconds:  req.JitterInMilliseconds,
		ErrorRate:             req.ErrorRate,
		ErrorStatus:           req.ErrorStatus,
	})

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"route": req.Route,
		"chaos": fault.Snapshot(),
	}); err != nil {
		log.Printf("writing admin chaos response: %v", err)
	}
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": msg}); err != nil {
		log.Printf("writing json error response: %v", err)
	}
}
