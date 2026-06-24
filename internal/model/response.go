package model

import "encoding/json"

// Response is the standard JSON envelope for all CLI output.
type Response struct {
	Meta Meta        `json:"meta"`
	Data interface{} `json:"data"`
}

// Meta contains query metadata.
type Meta struct {
	Command  string      `json:"command"`
	Database string      `json:"database,omitempty"`
	Filter   interface{} `json:"filter,omitempty"`
	Total    int         `json:"total"`
	Returned int         `json:"returned"`
	Limit    int         `json:"limit"`
	Offset   int         `json:"offset"`
}

// Marshal returns the JSON representation of a Response.
func (r *Response) Marshal() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
