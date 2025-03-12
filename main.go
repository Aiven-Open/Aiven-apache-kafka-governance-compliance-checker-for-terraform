package main

import (
	"log"
	"maps"
	"os"
	"slices"

	"aiven/terraform/governance/compliance/checker/internal/input"
	"aiven/terraform/governance/compliance/checker/internal/terraform"
)

type ResultError struct {
	Error   string          `json:"error"`
	Address string          `json:"address"`
	Tags    []terraform.Tag `json:"tags"`
}

type Check func(
	terraform.ResourceChange,
	*terraform.PriorStateResource,
	[]*terraform.PriorStateResource,
	*terraform.Plan,
) CheckResult

type ResourceErrorKey struct {
	resource string
	error    string
}

var checks = map[terraform.ResourceType][]Check{
	terraform.AivenKafkaTopic:                  {changeIsRequestedByOwner, changeIsApprovedByOwner},
	terraform.AivenExternalIdentity:            {},
	terraform.AivenOrganizationUserGroupMember: {},
	terraform.AivenGovernanceAccess:            {governanceAccessCheck},
}

func main() {
	logger := log.New(os.Stderr, "", 0)

	args, inputErr := input.NewInput(os.Args[1:])
	if inputErr != nil {
		logger.Fatal(inputErr)
	}

	plan, err := terraform.NewPlan(args.Plan)
	if err != nil {
		logger.Fatal(err)
	}

	result := Result{Ok: true, Errors: []ResultError{}}

	requester := findExternalIdentity(args.Requester, plan)
	approvers := findApprovers(args.Approvers, args.Requester, plan)

	for _, resourceChange := range plan.ResourceChanges {
		errors := validateResourceChange(resourceChange, requester, approvers, plan)
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
	resourceChange terraform.ResourceChange,
	requester *terraform.PriorStateResource,
	approvers []*terraform.PriorStateResource,
	plan *terraform.Plan,
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
func findExternalIdentity(userID string, plan *terraform.Plan) *terraform.PriorStateResource {
	for _, resource := range plan.PriorState.Values.RootModule.Resources {
		if resource.Type == terraform.AivenExternalIdentity && userID == resource.Values.ExternalUserID {
			return &resource
		}
	}
	return nil
}

func findApprovers(approverIDs []string, requesterID string, plan *terraform.Plan) []*terraform.PriorStateResource {
	var approvers []*terraform.PriorStateResource
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
func findOwnerAddressFromConfig(address string, plan *terraform.Plan) *string {
	for _, resource := range plan.Configuration.RootModule.Resources {
		if resource.Address == address {
			return &resource.Expressions.OwnerUserGroupID.References[1]
		}
	}
	return nil
}

// Find the user address from the proposed / planned Terraform configuration
func findUserAddressFromConfig(address string, plan *terraform.Plan) *string {
	for _, resource := range plan.Configuration.RootModule.Resources {
		if resource.Address == address {
			return &resource.Expressions.InternalUserID.References[1]
		}
	}
	return nil
}

// Check if the user is a member of the owner group in the proposed / planned Terraform configuration
func isUserGroupMemberInConfig(
	resourceChange terraform.ResourceChange,
	user *terraform.PriorStateResource,
	plan *terraform.Plan,
) bool {
	ownerAddress := findOwnerAddressFromConfig(resourceChange.Address, plan)
	if user == nil || ownerAddress == nil {
		return false
	}

	userAddress := findUserAddressFromConfig(user.Address, plan)
	if userAddress == nil {
		return false
	}

	for _, resource := range plan.Configuration.RootModule.Resources {
		if resource.Type == terraform.AivenOrganizationUserGroupMember {
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
func isUserGroupMemberInState(
	resourceWithOwner *terraform.ResourceChangeValues,
	user *terraform.PriorStateResource,
	plan *terraform.Plan,
) bool {
	if resourceWithOwner == nil {
		return false
	}
	for _, resource := range plan.PriorState.Values.RootModule.Resources {
		if resource.Type == terraform.AivenOrganizationUserGroupMember {
			if *resource.Values.GroupID == *resourceWithOwner.OwnerUserGroupID &&
				*resource.Values.UserID == user.Values.InternalUserID {
				return true
			}
		}
	}
	return false
}
