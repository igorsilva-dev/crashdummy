package models

type Response struct {
    Status   int `json:"status"`
    BodyFileName   string `json:"bodyFileName"` 
}