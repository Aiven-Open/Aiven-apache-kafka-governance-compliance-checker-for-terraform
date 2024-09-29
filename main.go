package main

import (
	"encoding/json"
	"flag"
	"os"
	"slices"
	"strings"
)

type (
	ResourceType 		string

	Plan struct {
		Changes []ResourceChange `json:"resource_changes"`
		State	struct {
			Values	struct {
				RootModule struct {
					Resources []Resource `json:"resources"`
				} `json:"root_module"`
			}	`json:"values"`
		} `json:"prior_state"`
		Configuration struct {
			RootModule struct {
				Resources []struct {
					Type        ResourceType 	`json:"type"`
					Name        string          `json:"name"`
					Address     string          `json:"address"`
					Expressions struct {
						OwnerUserGroupId	struct {
							References	[]string	`json:"references"`
						} `json:"owner_user_group_id"`
						InternalUserId	struct {
							References	[]string	`json:"references"`
						} `json:"internal_user_id"`
						GroupId	struct {
							References	[]string	`json:"references"`
						} `json:"group_id"`
						UserId	struct {
							References	[]string	`json:"references"`
						} `json:"user_id"`
					} `json:"expressions"`
				} `json:"resources"`
			} `json:"root_module"`
		} `json:"configuration"`
	}

	Resource struct {
		Type    ResourceType 	  `json:"type"`
		Name    string            `json:"name"`
		Address string            `json:"address"`
		Values  struct {
			InternalUserId		string	`json:"internal_user_id"`
			ExternalUserId		string	`json:"external_user_id"`
			OwnerUserGroupId	*string	`json:"owner_user_group_id"`
			GroupId				*string `json:"group_id"`
			UserId				*string	`json:"user_id"`
		} `json:"values"`
	}

	ResourceChange struct {
		Type   	ResourceType 	 `json:"type"`
		Name   	string           `json:"name"`
		Address	string			 `json:"address"`
		Change 	Change 			 `json:"change"`
	}

	Change struct {
		Actions 		[]string     	`json:"actions"`
		Before  		Resource 		`json:"before"`
		After   		Resource 		`json:"after"`
		AfterUnknown	struct {
			OwnerUserGroupId		bool	`json:"owner_user_group_id"`
		} `json:"after_unknown"`
	}

	ResultError struct {
		Error			string          `json:"error"`
		Address			string			`json:"address"`
	}

	Result struct {
		Ok       bool      	 `json:"ok"`
		Errors []ResultError `json:"errors"`
	}
)

const (
	AivenKafkaTopic                  ResourceType = "aiven_kafka_topic"
	AivenExternalIdentity            ResourceType = "aiven_external_identity"
	AivenOrganizationUserGroupMember ResourceType = "aiven_organization_user_group_member"
)

func main() {
	path := flag.String("plan", "", "path to a file with terraform plan output in json format")
	requesterId := flag.String("requester", "", "user identified as the requester of the change")
	approverIds := flag.String("approvers", "", "comma separated list of users identified as the approvers of the change")
	flag.Parse()

	if *path == "" || *requesterId == "" || *approverIds == "" {
		os.Stderr.WriteString("Missing required arguments\n")
		os.Exit(1)
	}

	content, err := os.ReadFile(*path)
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}

	var plan Plan
	err = json.Unmarshal(content, &plan)
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}

	result := Result{
		Ok:       true,
		Errors:   make([]ResultError, 0),
	}

	requester := GetExternalIdentity(*requesterId, &plan)
	var approvers []*Resource
	for _, approverId := range strings.Split(*approverIds, ",") {
		approvers = append(approvers, GetExternalIdentity(approverId, &plan))
	}

	for _, resourceChange := range plan.Changes {
		switch resourceChange.Type {
		case AivenKafkaTopic:
			CheckAivenKafkaTopicResource(resourceChange, requester, approvers, &plan, &result)
		}
	}

	output, _ := json.Marshal(result)
	os.Stdout.Write(output)
}

func CheckAivenKafkaTopicResource(resourceChange ResourceChange, requester *Resource, approvers []*Resource, plan *Plan, result *Result) {
	if resourceChange.Change.AfterUnknown.OwnerUserGroupId {
		CheckAivenKafkaTopicOwnerMembershipFromConfiguration(resourceChange, requester, approvers, plan, result)
	}
	if slices.Contains(resourceChange.Change.Actions, "create") {
		CheckAivenKafkaTopicOwnerMembershipFromState(resourceChange.Change.After, requester, approvers, plan, result)
	}
	if slices.Contains(resourceChange.Change.Actions, "update") {
		CheckAivenKafkaTopicOwnerMembershipFromState(resourceChange.Change.Before, requester, approvers, plan, result)
		CheckAivenKafkaTopicOwnerMembershipFromState(resourceChange.Change.After, requester, approvers, plan, result)
	}
	if slices.Contains(resourceChange.Change.Actions, "delete") {
		CheckAivenKafkaTopicOwnerMembershipFromState(resourceChange.Change.Before, requester, approvers, plan, result)
	}
}

func CheckAivenKafkaTopicOwnerMembershipFromState(topic Resource, requester *Resource, approvers []*Resource, plan *Plan, result *Result) {
	if topic.Values.OwnerUserGroupId == nil {
		return
	}
	if requester == nil {
		result.Ok = false
		result.Errors = append(result.Errors, GetRequestError(topic.Address))
		return
	}

	if !CheckUserGroupMembershipFromState(*topic.Values.OwnerUserGroupId, requester.Values.InternalUserId, plan) {
		result.Ok = false
		result.Errors = append(result.Errors, GetRequestError(topic.Address))
	}

	for _, approver := range approvers {
		if CheckUserGroupMembershipFromState(*topic.Values.OwnerUserGroupId, approver.Values.InternalUserId, plan) {
			return
		}
	}
	result.Ok = false
	result.Errors = append(result.Errors, GetApproveError(topic.Address))
}

func CheckUserGroupMembershipFromState(groupId string, userId string, plan *Plan) bool {
	for _, resource := range plan.State.Values.RootModule.Resources {
		if resource.Type == AivenOrganizationUserGroupMember {
			if *resource.Values.GroupId == groupId && *resource.Values.UserId == userId {
				return true
			}
		}
	}
	return false
}

func CheckAivenKafkaTopicOwnerMembershipFromConfiguration(resourceChange ResourceChange, requester *Resource, approvers []*Resource, plan *Plan, result *Result) {
	if !CheckUserGroupMembershipFromConfiguration(resourceChange, requester, plan) {
		result.Ok = false
		result.Errors = append(result.Errors, GetRequestError(resourceChange.Address))	
	}
	for _, approver := range approvers {
		if CheckUserGroupMembershipFromConfiguration(resourceChange, approver, plan) {
			return
		}
	}
	result.Ok = false
	result.Errors = append(result.Errors, GetApproveError(resourceChange.Address))
}

func CheckUserGroupMembershipFromConfiguration(resourceChange ResourceChange, user *Resource, plan *Plan) bool {
	var ownerAddress *string
	for _, resource := range plan.Configuration.RootModule.Resources {
		if resource.Address == resourceChange.Address {
			ownerAddress = &resource.Expressions.OwnerUserGroupId.References[1]
			break
		}
	}
	if user == nil || ownerAddress == nil {
		return false
	}

	var userAddress *string
	for _, resource := range plan.Configuration.RootModule.Resources {
		if resource.Address == user.Address {
			userAddress = &resource.Expressions.InternalUserId.References[1]
			break
		}
	}
	if userAddress == nil {
		return false
	}

	for _, resource := range plan.Configuration.RootModule.Resources {
		if resource.Type == AivenOrganizationUserGroupMember {
			groupReference := resource.Expressions.GroupId.References[1]
			userReference := resource.Expressions.UserId.References[1]
			if groupReference == *ownerAddress && userReference == *userAddress {	
				return true
			}
		}
	}
	return false
}

func GetExternalIdentity(userId string, plan *Plan) *Resource {
	for _, resource := range plan.State.Values.RootModule.Resources {
		if resource.Type == AivenExternalIdentity {
			if userId == resource.Values.ExternalUserId {
				return &resource
			}
		}
	}
	return nil
}

func GetRequestError(resourceAddress string) ResultError {
	return ResultError{
		Error:  "requesting user is not a member of the owner group",
		Address:  resourceAddress,
	}
}

func GetApproveError(resourceAddress string) ResultError {
	return ResultError{
		Error: "approval is required from a member of the owner group",
		Address: resourceAddress,
	}
}
