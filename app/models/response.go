package models

// Response describes the canned response a mapping serves.
type Response struct {
	Status       int    `json:"status"`
	BodyFileName string `json:"bodyFileName"`
}
