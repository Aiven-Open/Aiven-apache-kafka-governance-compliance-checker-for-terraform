package terraform

import (
	"encoding/json"
	"fmt"
	"os"
)

type Plan struct {
	ResourceChanges []ResourceChange `json:"resource_changes"`
	PriorState      PriorState       `json:"prior_state"`
	Configuration   Configuration    `json:"configuration"`
}

type PriorState struct {
	Values PriorStateValues `json:"values"`
}

type PriorStateValues struct {
	RootModule PriorStateModule `json:"root_module"`
}

type PriorStateModule struct {
	Resources []PriorStateResource `json:"resources"`
}

type PriorStateResource struct {
	Type    ResourceType             `json:"type"`
	Name    string                   `json:"name"`
	Address string                   `json:"address"`
	Values  PriorStateResourceValues `json:"values"`
}

type PriorStateResourceValues struct {
	InternalUserID   string  `json:"internal_user_id"`
	ExternalUserID   string  `json:"external_user_id"`
	Tag              []Tag   `json:"tag"`
	OwnerUserGroupID *string `json:"owner_user_group_id"`
	GroupID          *string `json:"group_id"`
	UserID           *string `json:"user_id"`
}

type Configuration struct {
	RootModule ConfigurationModule `json:"root_module"`
}

type ConfigurationModule struct {
	Resources []ConfigurationResource `json:"resources"`
}

type ConfigurationResource struct {
	Type        ResourceType `json:"type"`
	Name        string       `json:"name"`
	Address     string       `json:"address"`
	Expressions Expressions  `json:"expressions"`
}

type Expressions struct {
	OwnerUserGroupID *Expression `json:"owner_user_group_id"`
	InternalUserID   *Expression `json:"internal_user_id"`
	GroupID          *Expression `json:"group_id"`
	UserID           *Expression `json:"user_id"`
}

type Expression struct {
	References []string `json:"references"`
}

type ResourceChange struct {
	Type    ResourceType `json:"type"`
	Name    string       `json:"name"`
	Address string       `json:"address"`
	Change  Change       `json:"change"`
}

type Change struct {
	Actions      []ActionType          `json:"actions"`
	Before       *ResourceChangeValues `json:"before"`
	After        *ResourceChangeValues `json:"after"`
	AfterUnknown AfterUnknown          `json:"after_unknown"`
}

type ResourceChangeValues struct {
	InternalUserID   *string `json:"internal_user_id"`
	ExternalUserID   *string `json:"external_user_id"`
	Tag              *[]Tag  `json:"tag"`
	OwnerUserGroupID *string `json:"owner_user_group_id"`
	GroupID          *string `json:"group_id"`
	UserID           *string `json:"user_id"`
}

type AfterUnknown struct {
	OwnerUserGroupID *bool `json:"owner_user_group_id"`
}

type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ResourceType string

type ActionType string

const (
	AivenKafkaTopic                  ResourceType = "aiven_kafka_topic"
	AivenExternalIdentity            ResourceType = "aiven_external_identity"
	AivenOrganizationUserGroupMember ResourceType = "aiven_organization_user_group_member"
)

const (
	CreateAction ActionType = "create"
	UpdateAction ActionType = "update"
	DeleteAction ActionType = "delete"
)

func NewPlan(path string) (*Plan, error) {
	var plan Plan
	var err error
	var data []byte

	if data, err = os.ReadFile(path); err != nil {
		return nil, fmt.Errorf("invalid plan JSON file")
	}

	if err = json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("invalid plan JSON file")
	}

	return &plan, nil
}
