package models

type Proxy struct {
	Path                 string `json:"path"`
	Method               string `json:"method"`
	Upstream             string `json:"upstream"`
	LatencyInMillieconds int    `json:"latencyInMilliseconds"`
	JitterInMillieconds  int    `json:"jitterInMilliseconds"`
}
