package models

type Request struct {
    Method   string `json:"method"`
    Url   string `json:"url"`   
}