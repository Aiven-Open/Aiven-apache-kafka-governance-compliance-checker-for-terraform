package main

import (
	"aiven/terraform/governance/compliance/checker/internal/terraform"
	"slices"
)

type CheckResult struct {
	ok     bool
	errors []ResultError
}

func changeIsRequestedByOwner(
	resourceChange terraform.ResourceChange,
	requester *terraform.PriorStateResource,
	_ []*terraform.PriorStateResource,
	plan *terraform.Plan,
) CheckResult {
	checkResult := CheckResult{ok: true, errors: []ResultError{}}

	// If the owner is defined but it's a new group it's in the state post-apply so we have to use config to check it
	if ownerAfterApply := resourceChange.Change.AfterUnknown.OwnerUserGroupID; ownerAfterApply != nil && *ownerAfterApply {
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
	if slices.Contains(resourceChange.Change.Actions, terraform.CreateAction) {
		checkResult.errors = append(checkResult.errors,
			validateRequesterFromState(resourceChange.Address, resourceChange.Change.After, requester, plan)...)
	}
	// When the resource is updated, the requester must be a member of the owner group before and after the change
	if slices.Contains(resourceChange.Change.Actions, terraform.UpdateAction) {
		checkResult.errors = append(checkResult.errors,
			validateRequesterFromState(resourceChange.Address, resourceChange.Change.Before, requester, plan)...)
		checkResult.errors = append(checkResult.errors,
			validateRequesterFromState(resourceChange.Address, resourceChange.Change.After, requester, plan)...)
	}
	// When the resource is deleted, the requester must be a member of the owner group before the change
	if slices.Contains(resourceChange.Change.Actions, terraform.DeleteAction) {
		checkResult.errors = append(checkResult.errors,
			validateRequesterFromState(resourceChange.Address, resourceChange.Change.Before, requester, plan)...)
	}

	if len(checkResult.errors) > 0 {
		checkResult.ok = false
	}
	return checkResult
}

func changeIsApprovedByOwner(
	resourceChange terraform.ResourceChange,
	_ *terraform.PriorStateResource,
	approvers []*terraform.PriorStateResource,
	plan *terraform.Plan,
) CheckResult {
	checkResult := CheckResult{ok: true, errors: []ResultError{}}

	// If the owner is defined but it's a new group it's in the state post-apply so we have to use config to check it
	if ownerAfterApply := resourceChange.Change.AfterUnknown.OwnerUserGroupID; ownerAfterApply != nil && *ownerAfterApply {
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
	if slices.Contains(resourceChange.Change.Actions, terraform.CreateAction) {
		checkResult.errors = append(
			checkResult.errors,
			validateApproversFromState(resourceChange.Address, resourceChange.Change.After, approvers, plan)...,
		)
	}

	if slices.Contains(resourceChange.Change.Actions, terraform.UpdateAction) {
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
	if slices.Contains(resourceChange.Change.Actions, terraform.DeleteAction) {
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
	resource *terraform.ResourceChangeValues,
	approvers []*terraform.PriorStateResource,
	plan *terraform.Plan,
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
	resource *terraform.ResourceChangeValues,
	requester *terraform.PriorStateResource,
	plan *terraform.Plan,
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

func newRequestError(address string, tag *[]terraform.Tag) ResultError {
	err := "requesting user is not a member of the owner group"
	if tag != nil {
		return ResultError{
			Error:   err,
			Address: address,
			Tags:    *tag,
		}
	}
	return ResultError{
		Error:   err,
		Address: address,
		Tags:    []terraform.Tag{},
	}
}

func newApproveError(address string, tag *[]terraform.Tag) ResultError {
	err := "approval is required from a member of the owner group"
	if tag != nil {
		return ResultError{
			Error:   err,
			Address: address,
			Tags:    *tag,
		}
	}
	return ResultError{
		Error:   err,
		Address: address,
		Tags:    []terraform.Tag{},
	}
}
