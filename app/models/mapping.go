package models

// Mapping pairs a stubbed request with the canned response to serve for it.
type Mapping struct {
	Request        Request  `json:"request"`
	Response       Response `json:"response"`
	MappedResponse string   `json:"-"`
}
