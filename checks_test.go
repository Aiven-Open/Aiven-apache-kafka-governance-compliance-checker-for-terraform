package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnit_ValidateApproversFromStateSimpleCases(t *testing.T) {
	tests := []struct {
		name           string
		address        string
		resource       *ChangeResource
		approvers      []*StateResource
		plan           *Plan
		expectedErrors int
	}{
		{
			name:           "Resource has no owner group ID",
			address:        "resource1",
			resource:       &ChangeResource{OwnerUserGroupID: nil},
			approvers:      []*StateResource{{}},
			expectedErrors: 0,
		},
		{
			name:           "Resource owner group ID is empty",
			address:        "resource2",
			resource:       &ChangeResource{OwnerUserGroupID: stringPtr("")},
			approvers:      []*StateResource{{}},
			expectedErrors: 0,
		},
		{
			name:           "Resource is nil",
			address:        "resource3",
			resource:       nil,
			approvers:      []*StateResource{{}},
			expectedErrors: 0,
		},
		{
			name:           "No approvers",
			address:        "resource4",
			resource:       &ChangeResource{OwnerUserGroupID: stringPtr("owner-group")},
			approvers:      []*StateResource{},
			expectedErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultErrors := validateApproversFromState(tt.address, tt.resource, tt.approvers, tt.plan)
			if len(resultErrors) != tt.expectedErrors {
				t.Errorf("expected %d errors, got %d", tt.expectedErrors, len(resultErrors))
			}
		})
	}
}
func TestUnit_ValidateRequesterFromStateSimpleCases(t *testing.T) {
	tests := []struct {
		name           string
		address        string
		resource       *ChangeResource
		requester      *StateResource
		plan           *Plan
		expectedErrors int
	}{
		{
			name:           "Resource has no owner group ID",
			address:        "resource3",
			resource:       &ChangeResource{OwnerUserGroupID: nil},
			requester:      &StateResource{},
			expectedErrors: 0,
		},
		{
			name:           "Resource owner group ID is empty",
			address:        "resource4",
			resource:       &ChangeResource{OwnerUserGroupID: stringPtr("")},
			requester:      &StateResource{},
			expectedErrors: 0,
		},
		{
			name:     "Requester is nil",
			address:  "resource5",
			resource: &ChangeResource{OwnerUserGroupID: stringPtr("owner-group")},

			requester:      nil,
			expectedErrors: 1,
		},
		{
			name:           "Resource is nil",
			address:        "resource6",
			resource:       nil,
			requester:      &StateResource{},
			expectedErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultErrors := validateRequesterFromState(tt.address, tt.resource, tt.requester, tt.plan)
			if len(resultErrors) != tt.expectedErrors {
				t.Errorf("expected %d errors, got %d", tt.expectedErrors, len(resultErrors))
			}
		})
	}
}

func TestUnit_NewRequestError(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		tag      []Tag
		expected ResultError
	}{
		{
			name:    "Single tag",
			address: "resource1",
			tag:     []Tag{{Key: "env", Value: "prod"}},
			expected: ResultError{
				Error:   "requesting user is not a member of the owner group",
				Address: "resource1",
				Tags:    []Tag{{Key: "env", Value: "prod"}},
			},
		},
		{
			name:    "Multiple tags",
			address: "resource2",
			tag:     []Tag{{Key: "env", Value: "prod"}, {Key: "team", Value: "devops"}},
			expected: ResultError{
				Error:   "requesting user is not a member of the owner group",
				Address: "resource2",
				Tags:    []Tag{{Key: "env", Value: "prod"}, {Key: "team", Value: "devops"}},
			},
		},
		{
			name:    "No tags",
			address: "resource3",
			tag:     []Tag{},
			expected: ResultError{
				Error:   "requesting user is not a member of the owner group",
				Address: "resource3",
				Tags:    []Tag{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := newRequestError(tt.address, tt.tag)
			if !assert.ObjectsAreEqual(tt.expected, result) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestUnit_NewApproveError(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		tag      []Tag
		expected ResultError
	}{
		{
			name:    "Single tag",
			address: "resource1",
			tag:     []Tag{{Key: "env", Value: "prod"}},
			expected: ResultError{
				Error:   "approval is required from a member of the owner group",
				Address: "resource1",
				Tags:    []Tag{{Key: "env", Value: "prod"}},
			},
		},
		{
			name:    "Multiple tags",
			address: "resource2",
			tag:     []Tag{{Key: "env", Value: "prod"}, {Key: "team", Value: "devops"}},
			expected: ResultError{
				Error:   "approval is required from a member of the owner group",
				Address: "resource2",
				Tags:    []Tag{{Key: "env", Value: "prod"}, {Key: "team", Value: "devops"}},
			},
		},
		{
			name:    "No tags",
			address: "resource3",
			tag:     []Tag{},
			expected: ResultError{
				Error:   "approval is required from a member of the owner group",
				Address: "resource3",
				Tags:    []Tag{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := newApproveError(tt.address, tt.tag)
			if !assert.ObjectsAreEqual(tt.expected, result) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
