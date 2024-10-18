package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// These are very basic sad path tests cases, other cases are covered by the "e2e" tests of main.go
func TestUnit_ValidateKafkaTopicOwnerFromState(t *testing.T) {
	tests := []struct {
		name       string
		address    string
		topic      *ChangeResource
		requester  *StateResource
		approvers  []*StateResource
		plan       *Plan
		wantErrors []ResultError
	}{
		{
			name:       "Topic is nil",
			address:    "test-address",
			topic:      nil,
			requester:  &StateResource{},
			approvers:  []*StateResource{},
			plan:       &Plan{},
			wantErrors: []ResultError{},
		},
		{
			name:    "Topic owner is nil",
			address: "test-address",
			topic: &ChangeResource{
				OwnerUserGroupID: nil,
			},
			requester:  &StateResource{},
			approvers:  []*StateResource{},
			plan:       &Plan{},
			wantErrors: []ResultError{},
		},
		{
			name:    "Topic owner is empty",
			address: "test-address",
			topic: &ChangeResource{
				OwnerUserGroupID: new(string),
			},
			requester:  &StateResource{},
			approvers:  []*StateResource{},
			plan:       &Plan{},
			wantErrors: []ResultError{},
		},
		{
			name:    "Requester is nil",
			address: "test-address",
			topic: &ChangeResource{
				OwnerUserGroupID: stringPtr("owner-id"),
			},
			requester:  nil,
			approvers:  []*StateResource{},
			plan:       &Plan{},
			wantErrors: []ResultError{newRequestError("test-address", []Tag(nil))},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErrors := validateKafkaTopicOwnerFromState(tt.address, tt.topic, tt.requester, tt.approvers, tt.plan)
			assert.Equal(t, tt.wantErrors, gotErrors)
		})
	}
}

func TestUnit_ValidateKafkaTopicOwnerFromConfig(t *testing.T) {
	tests := []struct {
		name           string
		resourceChange ResourceChange
		requester      *StateResource
		approvers      []*StateResource
		plan           *Plan
		wantErrors     []ResultError
	}{
		{
			name: "Requester is not a member",
			resourceChange: ResourceChange{
				Address: "test-address",
				Change: Change{
					After: &ChangeResource{
						Tag: []Tag{},
					},
				},
			},
			requester:  &StateResource{},
			approvers:  []*StateResource{},
			plan:       &Plan{},
			wantErrors: []ResultError{newRequestError("test-address", []Tag{}), newApproveError("test-address", []Tag{})},
		},
		{
			name: "Approver is not a member",
			resourceChange: ResourceChange{
				Address: "test-address",
				Change: Change{
					After: &ChangeResource{
						Tag: []Tag{},
					},
				},
			},
			requester:  &StateResource{},
			approvers:  []*StateResource{},
			plan:       &Plan{},
			wantErrors: []ResultError{newRequestError("test-address", []Tag{}), newApproveError("test-address", []Tag{})},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErrors := validateKafkaTopicOwnerFromConfig(tt.resourceChange, tt.requester, tt.approvers, tt.plan)
			assert.Equal(t, tt.wantErrors, gotErrors)
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
