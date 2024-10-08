package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
)

type ResourceType string

type Plan struct {
	Changes       []ResourceChange `json:"resource_changes"`
	State         State            `json:"prior_state"`
	Configuration Configuration    `json:"configuration"`
}

type State struct {
	Values struct {
		RootModule struct {
			Resources []StateResource `json:"resources"`
		} `json:"root_module"`
	} `json:"values"`
}

type Configuration struct {
	RootModule struct {
		Resources []ConfigResource `json:"resources"`
	} `json:"root_module"`
}

type ConfigResource struct {
	Type        ResourceType `json:"type"`
	Name        string       `json:"name"`
	Address     string       `json:"address"`
	Expressions struct {
		OwnerUserGroupID struct {
			References []string `json:"references"`
		} `json:"owner_user_group_id"`
		InternalUserID struct {
			References []string `json:"references"`
		} `json:"internal_user_id"`
		GroupID struct {
			References []string `json:"references"`
		} `json:"group_id"`
		UserID struct {
			References []string `json:"references"`
		} `json:"user_id"`
	} `json:"expressions"`
}

type StateResource struct {
	Type    ResourceType `json:"type"`
	Name    string       `json:"name"`
	Address string       `json:"address"`
	Values  struct {
		InternalUserID   string  `json:"internal_user_id"`
		ExternalUserID   string  `json:"external_user_id"`
		OwnerUserGroupID *string `json:"owner_user_group_id"`
		GroupID          *string `json:"group_id"`
		UserID           *string `json:"user_id"`
	} `json:"values"`
}

type ChangeResource struct {
	InternalUserID   string  `json:"internal_user_id"`
	ExternalUserID   string  `json:"external_user_id"`
	OwnerUserGroupID *string `json:"owner_user_group_id"`
	GroupID          *string `json:"group_id"`
	UserID           *string `json:"user_id"`
}

type ResourceChange struct {
	Type    ResourceType `json:"type"`
	Name    string       `json:"name"`
	Address string       `json:"address"`
	Change  Change       `json:"change"`
}

type Change struct {
	Actions      []string       `json:"actions"`
	Before       ChangeResource `json:"before"`
	After        ChangeResource `json:"after"`
	AfterUnknown struct {
		OwnerUserGroupID bool `json:"owner_user_group_id"`
	} `json:"after_unknown"`
}

type ResultError struct {
	Error   string `json:"error"`
	Address string `json:"address"`
}

type Result struct {
	Ok     bool          `json:"ok"`
	Errors []ResultError `json:"errors"`
}

const (
	AivenKafkaTopic                  ResourceType = "aiven_kafka_topic"
	AivenExternalIdentity            ResourceType = "aiven_external_identity"
	AivenOrganizationUserGroupMember ResourceType = "aiven_organization_user_group_member"
)

func main() {
	path := flag.String("plan", "", "path to a file with terraform plan output in json format")
	requesterID := flag.String("requester", "", "user identified as the requester of the change")
	approverIDs := flag.String("approvers", "", "comma separated list of users identified as the approvers of the change")
	flag.Parse()

	if *path == "" {
		log.Fatal("Missing required arguments")
	}

	content, readErr := os.ReadFile(*path)
	if readErr != nil {
		log.Fatal(readErr)
	}

	var plan Plan
	if unmarshalErr := json.Unmarshal(content, &plan); unmarshalErr != nil {
		log.Fatal("Invalid plan JSON file")
	}

	result := Result{Ok: true, Errors: []ResultError{}}

	requester := findExternalIdentity(*requesterID, &plan)
	approvers := findApprovers(strings.Split(*approverIDs, ","), *requesterID, &plan)

	for _, resourceChange := range plan.Changes {
		if resourceChange.Type == AivenKafkaTopic {
			validateKafkaTopicChange(resourceChange, requester, approvers, &plan, &result)
		}
	}

	output, _ := json.Marshal(result)
	fmt.Println(string(output))
}

func findExternalIdentity(userID string, plan *Plan) *StateResource {
	for _, resource := range plan.State.Values.RootModule.Resources {
		if resource.Type == AivenExternalIdentity && userID == resource.Values.ExternalUserID {
			return &resource
		}
	}
	return nil
}

func findApprovers(approverIDs []string, requesterID string, plan *Plan) []*StateResource {
	var approvers []*StateResource
	for _, approverID := range approverIDs {
		approver := findExternalIdentity(approverID, plan)
		if approver != nil && requesterID != approverID {
			approvers = append(approvers, approver)
		}
	}
	return approvers
}

func validateKafkaTopicChange(
	topic ResourceChange,
	requester *StateResource,
	approvers []*StateResource,
	plan *Plan,
	result *Result,
) {
	if topic.Change.AfterUnknown.OwnerUserGroupID {
		validateKafkaTopicOwnerFromConfig(topic, requester, approvers, plan, result)
		return
	}
	if slices.Contains(topic.Change.Actions, "create") {
		validateKafkaTopicOwnerFromState(topic.Address, topic.Change.After, requester, approvers, plan, result)
	}
	if slices.Contains(topic.Change.Actions, "update") {
		validateKafkaTopicOwnerFromState(topic.Address, topic.Change.Before, requester, approvers, plan, result)
		validateKafkaTopicOwnerFromState(topic.Address, topic.Change.After, requester, approvers, plan, result)
	}
	if slices.Contains(topic.Change.Actions, "delete") {
		validateKafkaTopicOwnerFromState(topic.Address, topic.Change.Before, requester, approvers, plan, result)
	}
}

func validateKafkaTopicOwnerFromState(
	resourceAddress string,
	topic ChangeResource,
	requester *StateResource,
	approvers []*StateResource,
	plan *Plan,
	result *Result,
) {
	if topic.OwnerUserGroupID == nil {
		return
	}
	if requester == nil {
		result.Ok = false
		result.Errors = append(result.Errors, newRequestError(resourceAddress))
		return
	}

	if !isUserGroupMemberFromState(topic, requester, plan) {
		result.Ok = false
		result.Errors = append(result.Errors, newRequestError(resourceAddress))
	}

	for _, approver := range approvers {
		if isUserGroupMemberFromState(topic, approver, plan) {
			return
		}
	}
	result.Ok = false
	result.Errors = append(result.Errors, newApproveError(resourceAddress))
}

func validateKafkaTopicOwnerFromConfig(
	resourceChange ResourceChange,
	requester *StateResource,
	approvers []*StateResource,
	plan *Plan,
	result *Result,
) {
	if !isUserGroupMemberFromConfig(resourceChange, requester, plan) {
		result.Ok = false
		result.Errors = append(result.Errors, newRequestError(resourceChange.Address))
	}
	for _, approver := range approvers {
		if isUserGroupMemberFromConfig(resourceChange, approver, plan) {
			return
		}
	}
	result.Ok = false
	result.Errors = append(result.Errors, newApproveError(resourceChange.Address))

}

func findOwnerAddressFromConfig(resourceAddress string, plan *Plan) *string {
	for _, resource := range plan.Configuration.RootModule.Resources {
		if resource.Address == resourceAddress {
			return &resource.Expressions.OwnerUserGroupID.References[1]
		}
	}
	return nil
}

func findUserAddressFromConfig(resourceAddress string, plan *Plan) *string {
	for _, resource := range plan.Configuration.RootModule.Resources {
		if resource.Address == resourceAddress {
			return &resource.Expressions.InternalUserID.References[1]
		}
	}
	return nil
}

func isUserGroupMemberFromState(topic ChangeResource, user *StateResource, plan *Plan) bool {
	for _, resource := range plan.State.Values.RootModule.Resources {
		if resource.Type == AivenOrganizationUserGroupMember {
			if *resource.Values.GroupID == *topic.OwnerUserGroupID && *resource.Values.UserID == user.Values.InternalUserID {
				return true
			}
		}
	}
	return false
}

func isUserGroupMemberFromConfig(resourceChange ResourceChange, user *StateResource, plan *Plan) bool {
	ownerAddress := findOwnerAddressFromConfig(resourceChange.Address, plan)
	if user == nil || ownerAddress == nil {
		return false
	}

	userAddress := findUserAddressFromConfig(user.Address, plan)
	if userAddress == nil {
		return false
	}

	for _, resource := range plan.Configuration.RootModule.Resources {
		if resource.Type == AivenOrganizationUserGroupMember {
			groupReference := resource.Expressions.GroupID.References[1]
			userReference := resource.Expressions.UserID.References[1]
			if groupReference == *ownerAddress && userReference == *userAddress {
				return true
			}
		}
	}
	return false
}

func newRequestError(resourceAddress string) ResultError {
	return ResultError{
		Error:   "requesting user is not a member of the owner group",
		Address: resourceAddress,
	}
}

func newApproveError(resourceAddress string) ResultError {
	return ResultError{
		Error:   "approval is required from a member of the owner group",
		Address: resourceAddress,
	}
}
