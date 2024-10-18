package main

import "encoding/json"

type Result struct {
	Ok     bool          `json:"ok"`
	Errors []ResultError `json:"errors"`
}

func (result Result) toJSON() string {
	encoded, _ := json.Marshal(result)
	return string(encoded)
}
