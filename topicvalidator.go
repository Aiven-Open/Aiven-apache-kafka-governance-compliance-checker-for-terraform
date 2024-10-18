package main

import (
	"maps"
	"slices"
)

type TopicErrorKey struct {
	topic string
	error string
}

type TopicValidator struct{}

// ValidateResourceChange implements Validator.
func (tv TopicValidator) ValidateResourceChange(
	resource ResourceChange, // topic
	requester *StateResource,
	approvers []*StateResource,
	plan *Plan,
) []ResultError {
	var topicErrors = make([]ResultError, 0)
	if resource.Change.AfterUnknown.OwnerUserGroupID {
		topicErrors = append(topicErrors, validateKafkaTopicOwnerFromConfig(resource, requester, approvers, plan)...)

		// There is an error in validating topic owner so return the errors immediately
		return topicErrors
	}
	if slices.Contains(resource.Change.Actions, "create") {
		topicErrors = append(
			topicErrors,
			validateKafkaTopicOwnerFromState(resource.Address, resource.Change.After, requester, approvers, plan)...,
		)
	}
	if slices.Contains(resource.Change.Actions, "update") {
		// updating topic owner requires approvals from both old and the new owner
		// in other cases checking Change.After is redundant
		topicErrors = append(
			topicErrors,
			validateKafkaTopicOwnerFromState(resource.Address, resource.Change.Before, requester, approvers, plan)...,
		)
		topicErrors = append(
			topicErrors,
			validateKafkaTopicOwnerFromState(resource.Address, resource.Change.After, requester, approvers, plan)...,
		)
	}
	if slices.Contains(resource.Change.Actions, "delete") {
		topicErrors = append(
			topicErrors,
			validateKafkaTopicOwnerFromState(resource.Address, resource.Change.Before, requester, approvers, plan)...,
		)
	}

	// Convert the topicErrors into a map to remove duplicates
	topicErrorMap := make(map[TopicErrorKey]ResultError)
	for _, topicError := range topicErrors {
		topicErrorMap[TopicErrorKey{topic: resource.Name, error: topicError.Error}] = topicError
	}

	// Convert the map back into a slice
	return slices.Collect(maps.Values(topicErrorMap))
}

func validateKafkaTopicOwnerFromState(
	address string,
	topic *ChangeResource,
	requester *StateResource,
	approvers []*StateResource,
	plan *Plan,
) []ResultError {
	resultErrors := []ResultError{}

	// if the topic in state is missing or doesn't have an owner, return immediately
	if topic == nil {
		return resultErrors
	}
	if topic.OwnerUserGroupID == nil {
		return resultErrors
	}
	if *topic.OwnerUserGroupID == "" {
		return resultErrors
	}

	// Requester is required
	if requester == nil {
		resultErrors = append(resultErrors, newRequestError(address, topic.Tag))
		return resultErrors
	}

	if !isUserGroupMemberInState(topic, requester, plan) {
		resultErrors = append(resultErrors, newRequestError(address, topic.Tag))
		return resultErrors
	}

	// At least one approver is required
	for _, approver := range approvers {
		if isUserGroupMemberInState(topic, approver, plan) {
			// found a member, short circuit the function
			return resultErrors
		}
	}

	// did not find a member, add approve error
	resultErrors = append(resultErrors, newApproveError(address, topic.Tag))
	return resultErrors
}

func validateKafkaTopicOwnerFromConfig(
	resourceChange ResourceChange,
	requester *StateResource,
	approvers []*StateResource,
	plan *Plan,
) []ResultError {
	validationErrors := []ResultError{}
	if !isUserGroupMemberInConfig(resourceChange, requester, plan) {
		validationErrors = append(validationErrors, newRequestError(resourceChange.Address, resourceChange.Change.After.Tag))
	}
	for _, approver := range approvers {
		if isUserGroupMemberInConfig(resourceChange, approver, plan) {
			return validationErrors
		}
	}

	validationErrors = append(validationErrors, newApproveError(resourceChange.Address, resourceChange.Change.After.Tag))
	return validationErrors
}
