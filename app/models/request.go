package models

// Request describes the incoming request a mapping matches on.
type Request struct {
	Method string `json:"method"`
	URL    string `json:"url"`
}
