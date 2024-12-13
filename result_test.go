package main

import (
	"aiven/terraform/governance/compliance/checker/internal/terraform"
	"testing"
)

func TestResultToJSON(t *testing.T) {
	tests := []struct {
		name     string
		result   Result
		expected string
	}{
		{
			name: "Success result",
			result: Result{
				Ok:     true,
				Errors: []ResultError{},
			},
			expected: `{"ok":true,"errors":[]}`,
		},
		{
			name: "Failure result with errors",
			result: Result{
				Ok: false,
				Errors: []ResultError{
					{Error: "Error 1"},
					{Error: "Error 2", Address: "Address 2"},
					{Error: "Error 3", Address: "Address 3", Tags: []terraform.Tag{{Key: "Key 1", Value: "Value 1"}}},
				},
			},
			//nolint: lll
			expected: `{"ok":false,"errors":[{"error":"Error 1","address":"","tags":null},{"error":"Error 2","address":"Address 2","tags":null},{"error":"Error 3","address":"Address 3","tags":[{"key":"Key 1","value":"Value 1"}]}]}`,
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			resultJSON := testcase.result.toJSON()

			if resultJSON != testcase.expected {
				t.Errorf("Expected %v, but got %v", testcase.expected, resultJSON)
			}

		})
	}
}
