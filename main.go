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
			Resources []Resource `json:"resources"`
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

type Resource struct {
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

type ResourceChange struct {
	Type    ResourceType `json:"type"`
	Name    string       `json:"name"`
	Address string       `json:"address"`
	Change  Change       `json:"change"`
}

type Change struct {
	Actions      []string `json:"actions"`
	Before       Resource `json:"before"`
	After        Resource `json:"after"`
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

	if *path == "" || *requesterID == "" || *approverIDs == "" {
		log.Fatal("Missing required arguments")
	}

	content, readErr := os.ReadFile(*path)
	if readErr != nil {
		log.Fatal(readErr)
	}

	var plan Plan
	if unmarshalErr := json.Unmarshal(content, &plan); unmarshalErr != nil {
		log.Fatal(unmarshalErr)
	}

	result := Result{Ok: true, Errors: []ResultError{}}

	requester := findExternalIdentity(*requesterID, &plan)
	approvers := findApprovers(strings.Split(*approverIDs, ","), &plan)

	for _, resourceChange := range plan.Changes {
		if resourceChange.Type == AivenKafkaTopic {
			validateKafkaTopicChange(resourceChange, requester, approvers, &plan, &result)
		}
	}

	output, _ := json.Marshal(result)
	fmt.Println(string(output))
}

func findExternalIdentity(userID string, plan *Plan) *Resource {
	for _, resource := range plan.State.Values.RootModule.Resources {
		if resource.Type == AivenExternalIdentity && userID == resource.Values.ExternalUserID {
			return &resource
		}
	}
	return nil
}

func findApprovers(approverIDs []string, plan *Plan) []*Resource {
	var approvers []*Resource
	for _, approverID := range approverIDs {
		approvers = append(approvers, findExternalIdentity(approverID, plan))
	}
	return approvers
}

func validateKafkaTopicChange(
	resourceChange ResourceChange,
	requester *Resource,
	approvers []*Resource,
	plan *Plan,
	result *Result,
) {
	if resourceChange.Change.AfterUnknown.OwnerUserGroupID {
		validateKafkaTopicOwnerFromConfig(resourceChange, requester, plan, result)
	}
	if slices.Contains(resourceChange.Change.Actions, "create") {
		validateKafkaTopicOwnerFromState(resourceChange.Change.After, requester, approvers, plan, result)
	}
	if slices.Contains(resourceChange.Change.Actions, "update") {
		validateKafkaTopicOwnerFromState(resourceChange.Change.Before, requester, approvers, plan, result)
		validateKafkaTopicOwnerFromState(resourceChange.Change.After, requester, approvers, plan, result)
	}
	if slices.Contains(resourceChange.Change.Actions, "delete") {
		validateKafkaTopicOwnerFromState(resourceChange.Change.Before, requester, approvers, plan, result)
	}
}

func validateKafkaTopicOwnerFromState(
	topic Resource,
	requester *Resource,
	approvers []*Resource,
	plan *Plan,
	result *Result,
) {
	if topic.Values.OwnerUserGroupID == nil {
		return
	}
	if requester == nil {
		result.Ok = false
		result.Errors = append(result.Errors, newRequestError(topic.Address))
		return
	}

	if !isUserGroupMember(*topic.Values.OwnerUserGroupID, requester.Values.InternalUserID, plan) {
		result.Ok = false
		result.Errors = append(result.Errors, newRequestError(topic.Address))
	}

	for _, approver := range approvers {
		if isUserGroupMember(*topic.Values.OwnerUserGroupID, approver.Values.InternalUserID, plan) {
			return
		}
	}
	result.Ok = false
	result.Errors = append(result.Errors, newApproveError(topic.Address))
}

func validateKafkaTopicOwnerFromConfig(resourceChange ResourceChange, requester *Resource, plan *Plan, result *Result) {
	ownerAddress := findOwnerAddress(resourceChange.Address, plan)
	if ownerAddress == nil {
		return
	}

	if requester == nil {
		result.Ok = false
		result.Errors = append(result.Errors, newRequestError(resourceChange.Address))
		return
	}

	userAddress := findUserAddress(requester.Address, plan)
	if userAddress == nil {
		return
	}

	for _, resource := range plan.Configuration.RootModule.Resources {
		if resource.Type == AivenOrganizationUserGroupMember {
			groupReference := resource.Expressions.GroupID.References[1]
			userReference := resource.Expressions.UserID.References[1]
			if groupReference == *ownerAddress && userReference == *userAddress {
				return
			}
		}
	}
	result.Ok = false
	result.Errors = append(result.Errors, newRequestError(resourceChange.Address))
}

func findOwnerAddress(resourceAddress string, plan *Plan) *string {
	for _, resource := range plan.Configuration.RootModule.Resources {
		if resource.Address == resourceAddress {
			return &resource.Expressions.OwnerUserGroupID.References[1]
		}
	}
	return nil
}

func findUserAddress(resourceAddress string, plan *Plan) *string {
	for _, resource := range plan.Configuration.RootModule.Resources {
		if resource.Address == resourceAddress {
			return &resource.Expressions.InternalUserID.References[1]
		}
	}
	return nil
}

func isUserGroupMember(groupID string, userID string, plan *Plan) bool {
	for _, resource := range plan.State.Values.RootModule.Resources {
		if resource.Type == AivenOrganizationUserGroupMember {
			if *resource.Values.GroupID == groupID && *resource.Values.UserID == userID {
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
