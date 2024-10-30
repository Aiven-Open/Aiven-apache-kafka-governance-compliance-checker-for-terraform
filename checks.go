package main

import (
	"slices"
)

type CheckResult struct {
	ok     bool
	errors []ResultError
}

func changeIsRequestedByOwner(
	resourceChange ResourceChange,
	requester *StateResource,
	_ []*StateResource,
	plan *Plan,
) CheckResult {
	checkResult := CheckResult{ok: true, errors: []ResultError{}}

	// If the owner is defined but it's a new group it's in the state post-apply so we have to use config to check it
	if resourceChange.Change.AfterUnknown.OwnerUserGroupID {
		if !isUserGroupMemberInConfig(resourceChange, requester, plan) {
			checkResult.ok = false
			checkResult.errors = append(checkResult.errors,
				newRequestError(resourceChange.Address, resourceChange.Change.After.Tag),
			)
		}

		// There is an error in validating topic owner so return the errors immediately
		return checkResult
	}

	// When the resource is created, the requester must be a member of the owner group after the change
	if slices.Contains(resourceChange.Change.Actions, "create") {
		checkResult.errors = append(checkResult.errors,
			validateRequesterFromState(resourceChange.Address, resourceChange.Change.After, requester, plan)...)
	}
	// When the resource is updated, the requester must be a member of the owner group before and after the change
	if slices.Contains(resourceChange.Change.Actions, "update") {
		checkResult.errors = append(checkResult.errors,
			validateRequesterFromState(resourceChange.Address, resourceChange.Change.Before, requester, plan)...)
		checkResult.errors = append(checkResult.errors,
			validateRequesterFromState(resourceChange.Address, resourceChange.Change.After, requester, plan)...)
	}
	// When the resource is deleted, the requester must be a member of the owner group before the change
	if slices.Contains(resourceChange.Change.Actions, "delete") {
		checkResult.errors = append(checkResult.errors,
			validateRequesterFromState(resourceChange.Address, resourceChange.Change.Before, requester, plan)...)
	}

	if len(checkResult.errors) > 0 {
		checkResult.ok = false
	}
	return checkResult
}

func changeIsApprovedByOwner(
	resourceChange ResourceChange,
	_ *StateResource,
	approvers []*StateResource,
	plan *Plan,
) CheckResult {
	checkResult := CheckResult{ok: true, errors: []ResultError{}}

	// If the owner is defined but it's a new group it's in the state post-apply so we have to use config to check it
	if resourceChange.Change.AfterUnknown.OwnerUserGroupID {
		foundApprover := false
		for _, approver := range approvers {
			if isUserGroupMemberInConfig(resourceChange, approver, plan) {
				foundApprover = true // one known approver is enough
			}
		}

		if !foundApprover {
			checkResult.ok = false
			checkResult.errors = append(checkResult.errors,
				newApproveError(resourceChange.Address, resourceChange.Change.After.Tag),
			)

			// There is an error in validating topic owner so return the errors immediately
			return checkResult
		}
	}

	// When the resource is created, the approvers must be a member of the owner group after the change
	if slices.Contains(resourceChange.Change.Actions, "create") {
		checkResult.errors = append(
			checkResult.errors,
			validateApproversFromState(resourceChange.Address, resourceChange.Change.After, approvers, plan)...,
		)
	}

	if slices.Contains(resourceChange.Change.Actions, "update") {
		// updating owner requires approvals from both old and the new owner
		// in other cases checking Change.After would be redundant
		checkResult.errors = append(
			checkResult.errors,
			validateApproversFromState(resourceChange.Address, resourceChange.Change.Before, approvers, plan)...,
		)
		checkResult.errors = append(
			checkResult.errors,
			validateApproversFromState(resourceChange.Address, resourceChange.Change.After, approvers, plan)...,
		)
	}

	// When the resource is deleted, the approvers must be a member of the owner group before the change
	if slices.Contains(resourceChange.Change.Actions, "delete") {
		checkResult.errors = append(
			checkResult.errors,
			validateApproversFromState(resourceChange.Address, resourceChange.Change.Before, approvers, plan)...,
		)
	}

	if len(checkResult.errors) > 0 {
		checkResult.ok = false
	}
	return checkResult
}

func validateApproversFromState(
	address string,
	resource *ChangeResource,
	approvers []*StateResource,
	plan *Plan,
) []ResultError {
	resultErrors := []ResultError{}

	// if the resource in state is missing or doesn't have an owner, return immediately
	if resource == nil {
		return resultErrors
	}
	if resource.OwnerUserGroupID == nil {
		return resultErrors
	}
	if *resource.OwnerUserGroupID == "" {
		return resultErrors
	}

	// At least one approver is required
	for _, approver := range approvers {
		if isUserGroupMemberInState(resource, approver, plan) {
			// found a member, short circuit the function
			return resultErrors
		}
	}

	// did not find a member, add an approve error
	resultErrors = append(resultErrors, newApproveError(address, resource.Tag))
	return resultErrors
}

func validateRequesterFromState(
	address string,
	resource *ChangeResource,
	requester *StateResource,
	plan *Plan,
) []ResultError {
	resultErrors := []ResultError{}

	// if the resource in state is missing or doesn't have an owner, return immediately
	if resource == nil {
		return resultErrors
	}
	if resource.OwnerUserGroupID == nil {
		return resultErrors
	}
	if *resource.OwnerUserGroupID == "" {
		return resultErrors
	}

	// Requester is required
	if requester == nil || !isUserGroupMemberInState(resource, requester, plan) {
		resultErrors = append(resultErrors, newRequestError(address, resource.Tag))
		return resultErrors
	}

	// did not find any member, add an approve error
	return resultErrors
}

func newRequestError(address string, tag []Tag) ResultError {
	return ResultError{
		Error:   "requesting user is not a member of the owner group",
		Address: address,
		Tags:    tag,
	}
}

func newApproveError(address string, tag []Tag) ResultError {
	return ResultError{
		Error:   "approval is required from a member of the owner group",
		Address: address,
		Tags:    tag,
	}
}
