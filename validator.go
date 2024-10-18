package main

type Validator interface {
	ValidateResourceChange(
		resource ResourceChange,
		requester *StateResource,
		approvers []*StateResource,
		plan *Plan,
	) []ResultError
}
