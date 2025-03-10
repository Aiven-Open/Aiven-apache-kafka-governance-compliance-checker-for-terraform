package main

import (
	"aiven/terraform/governance/compliance/checker/internal/terraform"
	"fmt"
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

func isSubscriptionResource(
	subscriptionData terraform.SubscriptionData,
	acl terraform.SubscriptionACL,
	resource terraform.ResourceChange,
) bool {
	if resource.Type != terraform.AivenKafkaTopic {
		return false
	}

	if resource.Change.After == nil {
		return false
	}

	after := *resource.Change.After
	if after.Project != nil && *after.Project != subscriptionData.Project {
		return false
	}

	if after.ServiceName != nil && *after.ServiceName != subscriptionData.ServiceName {
		return false
	}

	if after.TopicName != nil && *after.TopicName != acl.ResourceName {
		return false
	}

	return true
}

func getSubscriptionResources(
	resourceChange terraform.ResourceChange,
	plan *terraform.Plan,
) []terraform.ResourceChange {
	resources := []terraform.ResourceChange{}

	var subscriptionData terraform.SubscriptionData
	if resourceChange.Change.After.SubscriptionData != nil {
		subscriptionData = (*resourceChange.Change.After.SubscriptionData)[0]
	}

	for _, acl := range subscriptionData.Acls {
		for _, resource := range plan.ResourceChanges {
			if isSubscriptionResource(subscriptionData, acl, resource) {
				resources = append(resources, resource)
			}
		}
	}
	return resources
}

func governanceSubscriptionCreateCheck(
	resourceChange terraform.ResourceChange,
	approvers []*terraform.PriorStateResource,
	plan *terraform.Plan,
) CheckResult {

	checkResult := CheckResult{ok: true, errors: []ResultError{}}

	// Check each subscription resource
resources:
	for _, resource := range getSubscriptionResources(resourceChange, plan) {
		ownerUnknown := resource.Change.AfterUnknown.OwnerUserGroupID != nil && *resource.Change.AfterUnknown.OwnerUserGroupID

		// We need one approver to be a member the resource owner group
		for _, approver := range approvers {
			if ownerUnknown && isUserGroupMemberInConfig(resource, approver, plan) {
				continue resources
			}
			if !ownerUnknown && isUserGroupMemberInState(resource.Change.After, approver, plan) {
				continue resources
			}
		}

		// No approval found, add error
		checkResult.errors = append(checkResult.errors, ResultError{
			Error:   fmt.Sprintf("approval is required from a owner of %s", resource.Address),
			Address: resourceChange.Address,
		})

	}

	if len(checkResult.errors) > 0 {
		checkResult.ok = false
	}

	return checkResult
}

func governanceSubscriptionDeleteCheck(
	resourceChange terraform.ResourceChange,
	approvers []*terraform.PriorStateResource,
	plan *terraform.Plan,
) CheckResult {
	checkResult := CheckResult{ok: true, errors: []ResultError{}}

	checkResult.errors = append(
		checkResult.errors,
		validateApproversFromState(resourceChange.Address, resourceChange.Change.Before, approvers, plan)...,
	)

	if len(checkResult.errors) > 0 {
		checkResult.ok = false
	}

	return checkResult
}

func governanceSubscriptionCheck(
	resourceChange terraform.ResourceChange,
	_ *terraform.PriorStateResource,
	approvers []*terraform.PriorStateResource,
	plan *terraform.Plan,
) CheckResult {
	// For create, approval is required from owners of the resources where the subscription grants access
	if slices.Contains(resourceChange.Change.Actions, terraform.CreateAction) {
		return governanceSubscriptionCreateCheck(resourceChange, approvers, plan)
	}

	return governanceSubscriptionDeleteCheck(resourceChange, approvers, plan)
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
