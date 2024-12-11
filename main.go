package main

import (
	"aiven/terraform/governance/compliance/checker/internal/input"
	"encoding/json"
	"log"
	"maps"
	"os"
	"slices"
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
		Tag              []Tag   `json:"tag"` // if supported by the resource
		OwnerUserGroupID *string `json:"owner_user_group_id"`
		GroupID          *string `json:"group_id"`
		UserID           *string `json:"user_id"`
	} `json:"values"`
}

type ChangeResource struct {
	InternalUserID   string  `json:"internal_user_id"`
	ExternalUserID   string  `json:"external_user_id"`
	Tag              []Tag   `json:"tag"`
	OwnerUserGroupID *string `json:"owner_user_group_id"`
	GroupID          *string `json:"group_id"`
	UserID           *string `json:"user_id"`
}

// Represents a single change in the plan (resource_changes array)
// including the resource type, name, address and the change itself
type ResourceChange struct {
	Type    ResourceType `json:"type"`
	Name    string       `json:"name"`
	Address string       `json:"address"`
	Change  Change       `json:"change"`
}

// Actual change of a resource
// including the actions (create, update, delete), the resource before and after the change
type Change struct {
	Actions      []string        `json:"actions"`
	Before       *ChangeResource `json:"before"`
	After        *ChangeResource `json:"after"`
	AfterUnknown struct {
		OwnerUserGroupID bool `json:"owner_user_group_id"`
	} `json:"after_unknown"`
}

type ResultError struct {
	Error   string `json:"error"`
	Address string `json:"address"`
	Tags    []Tag  `json:"tags"`
}

type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

const (
	AivenKafkaTopic                  ResourceType = "aiven_kafka_topic"
	AivenExternalIdentity            ResourceType = "aiven_external_identity"
	AivenOrganizationUserGroupMember ResourceType = "aiven_organization_user_group_member"
)

type Check func(ResourceChange, *StateResource, []*StateResource, *Plan) CheckResult

type ResourceErrorKey struct {
	resource string
	error    string
}

var checks = map[ResourceType][]Check{
	AivenKafkaTopic:                  {changeIsRequestedByOwner, changeIsApprovedByOwner},
	AivenExternalIdentity:            {},
	AivenOrganizationUserGroupMember: {},
}

func main() {
	logger := log.New(os.Stderr, "", 0)

	args, inputErr := input.NewInput(os.Args[1:])
	if inputErr != nil {
		logger.Fatal(inputErr)
	}

	content, readErr := os.ReadFile(args.Plan)
	if readErr != nil {
		logger.Fatal("invalid plan JSON file")
	}

	var plan Plan
	if unmarshErr := json.Unmarshal(content, &plan); unmarshErr != nil {
		logger.Fatal("invalid plan JSON file")
	}

	result := Result{Ok: true, Errors: []ResultError{}}

	requester := findExternalIdentity(args.Requester, &plan)
	approvers := findApprovers(args.Approvers, args.Requester, &plan)

	for _, resourceChange := range plan.Changes {
		errors := validateResourceChange(resourceChange, requester, approvers, &plan)
		result.Errors = append(result.Errors, errors...)
	}

	// result.Ok is the source of truth for the result of the validation
	if len(result.Errors) > 0 {
		result.Ok = false
	}

	logger.SetOutput(os.Stdout)
	logger.Println(result.toJSON())
}

func validateResourceChange(
	resourceChange ResourceChange,
	requester *StateResource,
	approvers []*StateResource,
	plan *Plan,
) []ResultError {

	resourceChecks, ok := checks[resourceChange.Type]
	if !ok {
		// no checks for this resource type
		return []ResultError{}
	}

	var checkErrors = make([]ResultError, 0)

	//  run the checks and collect errors
	for _, check := range resourceChecks {
		singleCheckResult := check(resourceChange, requester, approvers, plan)
		if !singleCheckResult.ok {
			checkErrors = append(checkErrors, singleCheckResult.errors...)
		}
	}

	// Remove duplicate errors
	errorMap := make(map[ResourceErrorKey]ResultError)
	for _, err := range checkErrors {
		errorMap[ResourceErrorKey{resource: resourceChange.Name, error: err.Error}] = err
	}

	// Convert the map back into a slice
	return slices.Collect(maps.Values(errorMap))

}

// Finds external identity resource for a given user ID from the current (prior) state
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
		// requester can't approve their own request
		if approver != nil && requesterID != approverID {
			approvers = append(approvers, approver)
		}
	}
	return approvers
}

// Find the owner address from the proposed / planned Terraform configuration
func findOwnerAddressFromConfig(address string, plan *Plan) *string {
	for _, resource := range plan.Configuration.RootModule.Resources {
		if resource.Address == address {
			return &resource.Expressions.OwnerUserGroupID.References[1]
		}
	}
	return nil
}

// Find the user address from the proposed / planned Terraform configuration
func findUserAddressFromConfig(address string, plan *Plan) *string {
	for _, resource := range plan.Configuration.RootModule.Resources {
		if resource.Address == address {
			return &resource.Expressions.InternalUserID.References[1]
		}
	}
	return nil
}

// Check if the user is a member of the owner group in the proposed / planned Terraform configuration
func isUserGroupMemberInConfig(resourceChange ResourceChange, user *StateResource, plan *Plan) bool {
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

// Check if the user is a member of the owner group in the current Terraform state
func isUserGroupMemberInState(resourceWithOwner *ChangeResource, user *StateResource, plan *Plan) bool {
	if resourceWithOwner == nil {
		return false
	}
	for _, resource := range plan.State.Values.RootModule.Resources {
		if resource.Type == AivenOrganizationUserGroupMember {
			if *resource.Values.GroupID == *resourceWithOwner.OwnerUserGroupID &&
				*resource.Values.UserID == user.Values.InternalUserID {
				return true
			}
		}
	}
	return false
}
