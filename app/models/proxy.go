package models

// Proxy forwards a local path to an upstream URL with injected latency and
// optional error injection.
type Proxy struct {
	Path                  string  `json:"path"`
	Method                string  `json:"method"`
	Upstream              string  `json:"upstream"`
	LatencyInMilliseconds int     `json:"latencyInMilliseconds"`
	JitterInMilliseconds  int     `json:"jitterInMilliseconds"`
	ErrorRate             float64 `json:"errorRate"`
	ErrorStatus           int     `json:"errorStatus"`
}
