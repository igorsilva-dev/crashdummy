package models

// Mapping pairs a stubbed request with the canned response to serve for it,
// plus optional fault injection (latency and errors) applied to that route.
type Mapping struct {
	Request               Request  `json:"request"`
	Response              Response `json:"response"`
	LatencyInMilliseconds int      `json:"latencyInMilliseconds"`
	JitterInMilliseconds  int      `json:"jitterInMilliseconds"`
	ErrorRate             float64  `json:"errorRate"`
	ErrorStatus           int      `json:"errorStatus"`
	MappedResponse        string   `json:"-"`
}
