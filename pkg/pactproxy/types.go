package pactproxy

import "encoding/json"

type Interaction struct {
	Alias          string                 `json:"alias"`
	Description    string                 `json:"description"`
	Definition     map[string]interface{} `json:"definition"`
	Method         string                 `json:"method"`
	RequestCount   int                    `json:"request_count"`
	RequestHistory []RequestDocument      `json:"request_history,omitempty"`
}

type RequestDocument struct {
	Headers map[string]string `json:"headers"`
	Query   json.RawMessage   `json:"query"`
	Body    json.RawMessage   `json:"body"`
	Path    string            `json:"path"`
}
