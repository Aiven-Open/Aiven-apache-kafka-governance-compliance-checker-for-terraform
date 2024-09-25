package main

import (
	"encoding/json"
	"flag"
	"os"
	"slices"
	"strings"
)

type AivenResourceType string

// resource type enum-like constant
const (
	KafkaTopic                       AivenResourceType = "aiven_kafka_topic"
	AivenExternalIdentity            AivenResourceType = "aiven_external_identity"
	AivenOrganizationUserGroupMember AivenResourceType = "aiven_organization_user_group_member"
)

type (
	PlanResource struct {
		Type   AivenResourceType `json:"type"`
		Name   string            `json:"name"`
		Values map[string]any    `json:"values"`
	}

	PlanResourceChange struct {
		Type   AivenResourceType `json:"type"`
		Name   string            `json:"name"`
		Change struct {
			Actions []string     `json:"actions"`
			Before  PlanResource `json:"before"`
			After   PlanResource `json:"after"`
		} `json:"change"`
	}

	Plan struct {
		PlannedValues struct {
			RootModule struct {
				Resources []PlanResource `json:"resources"`
			} `json:"root_module"`
		} `json:"planned_values"`
		ResourceChanges []PlanResourceChange `json:"resource_changes"`
	}

	Message struct {
		Title        string            `json:"title"`
		Description  string            `json:"description"`
		ResourceType AivenResourceType `json:"resource_type"`
		ResourceName string            `json:"resource_name"`
	}

	Result struct {
		Ok       bool      `json:"ok"`
		Messages []Message `json:"messages"`
	}
)

func main() {
	pathToPlan := flag.String("plan", "", "path to a file with terraform plan output in json format")
	requesterId := flag.String("requester", "", "user identified as the requester of the change")
	approverIds := flag.String("approvers", "", "comma separated list of users identified as the approvers of the change")
	flag.Parse()

	if *pathToPlan == "" || *requesterId == "" || *approverIds == "" {
		os.Stderr.WriteString("missing required arguments")
		os.Exit(1)
	}

	planContent, err := os.ReadFile(*pathToPlan)
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}

	var plan Plan
	err = json.Unmarshal(planContent, &plan)
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}

	result := Result{
		Ok:       true,
		Messages: make([]Message, 0),
	}

	var requester = ExternalIdentity(*requesterId, &plan)
	var approvers []*PlanResource
	for _, approverId := range strings.Split(*approverIds, ",") {
		approver := ExternalIdentity(approverId, &plan)
		if approver != nil {
			approvers = append(approvers, approver)
		}
	}

	for _, resource := range plan.ResourceChanges {
		switch resource.Type {
		case KafkaTopic:
			if slices.Contains(resource.Change.Actions, "create") {
				CheckTopicRequesterAndApprovers(requester, approvers, &resource.Change.After, &plan, &result)
			}
			if slices.Contains(resource.Change.Actions, "update") {
				CheckTopicRequesterAndApprovers(requester, approvers, &resource.Change.Before, &plan, &result)
				CheckTopicRequesterAndApprovers(requester, approvers, &resource.Change.After, &plan, &result)
			}
			if slices.Contains(resource.Change.Actions, "delete") {
				CheckTopicRequesterAndApprovers(requester, approvers, &resource.Change.Before, &plan, &result)
			}
		}
	}

	output, _ := json.Marshal(result)
	os.Stdout.Write(output)
}

func CheckTopicRequesterAndApprovers(requester *PlanResource, approvers []*PlanResource, resource *PlanResource, plan *Plan, result *Result) {
	requesterId, _ := requester.Values["internal_user_id"].(string)
	ownerGroupId, exists := resource.Values["owner_user_group_id"].(string)
	if !exists {
		return
	}

	membership := UserGroupMembership(requesterId, ownerGroupId, plan)
	if membership == nil {
		result.Ok = false
		result.Messages = append(result.Messages, Message{
			Title:        "MembershipRequired",
			Description:  "requester is not a member of the owner user group",
			ResourceType: resource.Type,
			ResourceName: resource.Name,
		})
	}

	var approved bool
	for _, approver := range approvers {
		approverId, _ := approver.Values["internal_user_id"].(string)
		membership := UserGroupMembership(approverId, ownerGroupId, plan)
		if membership != nil {
			approved = true
		}
	}

	if !approved {
		result.Ok = false
		result.Messages = append(result.Messages, Message{
			Title:        "ApprovalRequired",
			Description:  "approval is required from the owner user group",
			ResourceType: resource.Type,
			ResourceName: resource.Name,
		})
	}
}

func ExternalIdentity(userId string, plan *Plan) *PlanResource {
	for _, resource := range plan.PlannedValues.RootModule.Resources {
		if resource.Type == AivenExternalIdentity {
			extUserId, _ := resource.Values["external_user_id"].(string)
			if userId == extUserId {
				return &resource
			}
		}
	}
	return nil
}

func UserGroupMembership(userId string, groupId string, plan *Plan) *PlanResource {
	for _, resource := range plan.PlannedValues.RootModule.Resources {
		if resource.Type == AivenOrganizationUserGroupMember {
			memberUserId, _ := resource.Values["user_id"].(string)
			memberGroupId, _ := resource.Values["group_id"].(string)
			if userId == memberUserId && memberGroupId == groupId {
				return &resource
			}
		}
	}
	return nil
}
