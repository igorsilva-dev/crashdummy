package models

type Mapping struct {
	Request Request `json:"request"`
	Response Response `json:"response"`
	MappedResponse string
}