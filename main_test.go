package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type TestCase struct {
	Name         string
	Args         Args
	ExpectStdout string
	ExpectStderr string
}

type Args struct {
	Requester string
	Approvers string
	Plan      string
}

func TestE2E_Args(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	tests := []TestCase{
		{
			Name: "Plan is required",
			Args: Args{
				Requester: "alice",
				Approvers: "bob,charlie",
				Plan:      "",
			},
			ExpectStdout: "",
			ExpectStderr: "plan is a required argument\nexit status 1",
		},
		{
			Name: "Plan file needs to exist",
			Args: Args{
				Requester: "alice",
				Approvers: "bob,charlie",
				Plan:      "testdata/nonexistent_plan.json",
			},
			ExpectStdout: "",
			ExpectStderr: "invalid plan JSON file\nexit status 1",
		},
		{
			Name: "Plan file needs to be json",
			Args: Args{
				Requester: "alice",
				Approvers: "bob,charlie",
				Plan:      "testdata/not_json.py",
			},
			ExpectStdout: "",
			ExpectStderr: "invalid plan JSON file\nexit status 1",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			stdout, stderr, runErr := runCommand(dir, test.Args)
			if test.ExpectStderr != "" {
				if runErr == nil {
					t.Fatalf("Expected an error but got none")
				}
			} else {
				if runErr != nil {
					t.Fatalf("Command execution failed: %v", runErr)
				}
			}

			assertOutput(t, "stdout", stdout, test.ExpectStdout)
			assertOutput(t, "stderr", stderr, test.ExpectStderr)
		})
	}
}

func TestE2E_PlanWithKnownOwnerUserGroupID(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	plan := "./testdata/plan_with_known_owner_user_group_id.json"

	tests := []TestCase{
		{
			Name: fmt.Sprintf("[%s] Reports error if requester identity can not be resolved", plan),
			Args: Args{
				Requester: "nonexistent_user",
				Approvers: "bob,charlie",
				Plan:      plan,
			},
			ExpectStdout: Result{
				Ok: false,
				Errors: []ResultError{
					newRequestError("aiven_kafka_topic.bar[2]", []Tag{}),
					newRequestError("aiven_kafka_topic.foo", []Tag{{Key: "a", Value: "b"}}),
					newRequestError("aiven_kafka_topic.foo", []Tag{}),
				},
			}.toJSON(),
			ExpectStderr: "",
		},
		{
			Name: fmt.Sprintf("[%s] Reports error if requester is not a member of the owner user group", plan),
			Args: Args{
				Requester: "charlie",
				Approvers: "bob,charlie",
				Plan:      plan,
			},
			ExpectStdout: Result{
				Ok: false,
				Errors: []ResultError{
					newRequestError("aiven_kafka_topic.bar[2]", []Tag{}),
					newRequestError("aiven_kafka_topic.foo", []Tag{{Key: "a", Value: "b"}}),
					newRequestError("aiven_kafka_topic.foo", []Tag{}),
				},
			}.toJSON(),
			ExpectStderr: "",
		},
		{
			Name: fmt.Sprintf("[%s] Does not report error if requester is a member of the owner user group", plan),
			Args: Args{
				Requester: "alice",
				Approvers: "bob,charlie",
				Plan:      plan,
			},
			ExpectStdout: Result{
				Ok:     true,
				Errors: []ResultError{},
			}.toJSON(),
			ExpectStderr: "",
		},
		{
			Name: fmt.Sprintf("[%s] Reports error if approval is missing from a member of the owner user group", plan),
			Args: Args{
				Requester: "alice",
				Approvers: "frank",
				Plan:      plan,
			},
			ExpectStdout: Result{
				Ok: false,
				Errors: []ResultError{
					newApproveError("aiven_kafka_topic.bar[2]", []Tag{}),
					newApproveError("aiven_kafka_topic.foo", []Tag{{Key: "a", Value: "b"}}),
					newApproveError("aiven_kafka_topic.foo", []Tag{}),
				},
			}.toJSON(),
			ExpectStderr: "",
		},
		{
			Name: fmt.Sprintf("[%s] Does not consider requester as approver", plan),
			Args: Args{
				Requester: "alice",
				Approvers: "alice",
				Plan:      plan,
			},
			ExpectStdout: Result{
				Ok: false,
				Errors: []ResultError{
					newApproveError("aiven_kafka_topic.bar[2]", []Tag{}),
					newApproveError("aiven_kafka_topic.foo", []Tag{{Key: "a", Value: "b"}}),
					newApproveError("aiven_kafka_topic.foo", []Tag{}),
				},
			}.toJSON(),
			ExpectStderr: "",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			stdout, stderr, runErr := runCommand(dir, test.Args)
			if test.ExpectStderr != "" {
				if runErr == nil {
					t.Fatalf("Expected an error but got none")
				}
			} else {
				if runErr != nil {
					t.Fatalf("Command execution failed: %v", runErr)
				}
			}

			assertOutput(t, "stdout", stdout, test.ExpectStdout)
			assertOutput(t, "stderr", stderr, test.ExpectStderr)
		})
	}
}

func TestE2E_PlanWithUnknownOwnerUserGroupID(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	plan := "./testdata/plan_with_unknown_owner_user_group_id.json"

	tests := []TestCase{
		{
			Name: fmt.Sprintf("[%s] Reports error if requester identity can not be resolved", plan),
			Args: Args{
				Requester: "nonexistent_user",
				Approvers: "bob,charlie",
				Plan:      plan,
			},
			ExpectStdout: Result{
				Ok: false,
				Errors: []ResultError{
					newRequestError("aiven_kafka_topic.foo", []Tag{}),
				},
			}.toJSON(),
			ExpectStderr: "",
		},
		{
			Name: fmt.Sprintf("[%s] Reports error if requester is not a member of the owner user group", plan),
			Args: Args{
				Requester: "charlie",
				Approvers: "bob,charlie",
				Plan:      plan,
			},
			ExpectStdout: Result{
				Ok: false,
				Errors: []ResultError{
					newRequestError("aiven_kafka_topic.foo", []Tag{}),
				},
			}.toJSON(),
			ExpectStderr: "",
		},
		{
			Name: fmt.Sprintf("[%s] Does not report error if requester is a member of the owner user group", plan),
			Args: Args{
				Requester: "alice",
				Approvers: "bob,charlie",
				Plan:      plan,
			},
			ExpectStdout: Result{
				Ok:     true,
				Errors: []ResultError{},
			}.toJSON(),
			ExpectStderr: "",
		},
		{
			Name: fmt.Sprintf("[%s] Reports error if approval is missing from a member of the owner user group", plan),
			Args: Args{
				Requester: "alice",
				Approvers: "frank",
				Plan:      plan,
			},
			ExpectStdout: Result{
				Ok: false,
				Errors: []ResultError{
					newApproveError("aiven_kafka_topic.foo", []Tag{}),
				},
			}.toJSON(),
			ExpectStderr: "",
		},
		{
			Name: fmt.Sprintf("[%s] Does not consider requester as approver", plan),
			Args: Args{
				Requester: "alice",
				Approvers: "alice",
				Plan:      plan,
			},
			ExpectStdout: Result{
				Ok: false,
				Errors: []ResultError{
					newApproveError("aiven_kafka_topic.foo", []Tag{}),
				},
			}.toJSON(),
			ExpectStderr: "",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			stdout, stderr, runErr := runCommand(dir, test.Args)
			if test.ExpectStderr != "" {
				if runErr == nil {
					t.Fatalf("Expected an error but got none")
				}
			} else {
				if runErr != nil {
					t.Fatalf("Command execution failed: %v", runErr)
				}
			}

			assertOutput(t, "stdout", stdout, test.ExpectStdout)
			assertOutput(t, "stderr", stderr, test.ExpectStderr)
		})
	}
}

func runCommand(dir string, args Args) (string, string, error) {

	cmdArgs := make([]string, 0)
	cmdArgs = append(cmdArgs, "run")
	cmdArgs = append(cmdArgs, "main.go")
	if args.Requester != "" {
		cmdArgs = append(cmdArgs, fmt.Sprintf("-requester=%s", args.Requester))
	}
	if args.Approvers != "" {
		cmdArgs = append(cmdArgs, fmt.Sprintf("-approvers=%s", args.Approvers))
	}
	if args.Plan != "" {
		cmdArgs = append(cmdArgs, fmt.Sprintf("-plan=%s", filepath.Join(dir, args.Plan)))
	}

	var stdoutBuffer, stderrBuffer strings.Builder

	cmd := exec.Command("go", cmdArgs...)
	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer

	runErr := cmd.Run()
	return stdoutBuffer.String(), stderrBuffer.String(), runErr
}

func assertOutput(t *testing.T, name, actual, expected string) {
	actual = strings.TrimSpace(actual)
	expected = strings.TrimSpace(expected)

	if expected == "" {
		return
	}

	if actual != expected {
		t.Errorf("Expected %s to equal: %q, but got: %q", name, expected, actual)
	}
}

func getTestPlan(t *testing.T, path string) *Plan {
	content, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}

	var plan Plan
	if unmarshalErr := json.Unmarshal(content, &plan); unmarshalErr != nil {
		t.Fatal("Invalid plan JSON file")
	}

	return &plan
}

func TestUnit_findExternalIdentity(t *testing.T) {
	plan := getTestPlan(t, "testdata/plan_with_known_owner_user_group_id.json")

	t.Run("Finds existing external_identity", func(t *testing.T) {
		user := findExternalIdentity("alice", plan)
		if user == nil {
			t.Error()
		}
	})

	t.Run("Returns nil if not found", func(t *testing.T) {
		user := findExternalIdentity("frank", plan)
		if user != nil {
			t.Error()
		}
	})
}

func TestUnit_findApprovers(t *testing.T) {
	plan := getTestPlan(t, "testdata/plan_with_known_owner_user_group_id.json")

	t.Run("Finds existing external_identities for approvers", func(t *testing.T) {
		approverIDs := make([]string, 0)
		approverIDs = append(approverIDs, "bob")
		resources := findApprovers(approverIDs, "alice", plan)
		if len(resources) != 1 {
			t.Error()
		}
		if resources[0].Values.ExternalUserID != "bob" {
			t.Error()
		}
	})

	t.Run("Filters out requester if present", func(t *testing.T) {
		approverIDs := make([]string, 0)
		approverIDs = append(approverIDs, "alice")
		approverIDs = append(approverIDs, "bob")
		resources := findApprovers(approverIDs, "alice", plan)
		if len(resources) != 1 {
			t.Error()
		}
		if resources[0].Values.ExternalUserID != "bob" {
			t.Error()
		}
	})

	t.Run("Does not return user if not found (nil)", func(t *testing.T) {
		approverIDs := make([]string, 0)
		approverIDs = append(approverIDs, "frank")
		resources := findApprovers(approverIDs, "alice", plan)
		if len(resources) != 0 {
			t.Error()
		}
	})
}

func TestUnit_findOwnerAddressFromConfig(t *testing.T) {
	plan := getTestPlan(t, "testdata/plan_with_known_owner_user_group_id.json")

	t.Run("Return address if exists", func(t *testing.T) {
		address := findOwnerAddressFromConfig("aiven_kafka_topic.foo", plan)
		if *address != "aiven_organization_user_group.foo" {
			t.Error()
		}
	})

	t.Run("Return nil if not found", func(t *testing.T) {
		address := findOwnerAddressFromConfig("aiven_kafka_topic.test", plan)
		if address != nil {
			t.Error()
		}
	})
}

func TestUnit_findUserAddressFromConfig(t *testing.T) {
	plan := getTestPlan(t, "testdata/plan_with_known_owner_user_group_id.json")

	t.Run("Return address if exists", func(t *testing.T) {
		address := findUserAddressFromConfig("data.aiven_external_identity.alice", plan)
		if address == nil {
			t.Fatal()
		}
		if *address != "data.aiven_organization_user.foo" {
			t.Error()
		}
	})

	t.Run("Return nil if not found", func(t *testing.T) {
		address := findUserAddressFromConfig("data.aiven_external_identity.frank", plan)
		if address != nil {
			t.Error()
		}
	})
}

func TestUnit_isUserGroupMemberFromState(t *testing.T) {
	plan := getTestPlan(t, "testdata/plan_with_known_owner_user_group_id.json")

	t.Run("Return true if exists", func(t *testing.T) {
		topic := ChangeResource{}
		ownerUserGroupID := "ug4e3b20cee48"
		topic.OwnerUserGroupID = &ownerUserGroupID

		user := StateResource{}
		user.Values.InternalUserID = "u4e3706199a0"

		ok := isUserGroupMemberFromState(&topic, &user, plan)
		if !ok {
			t.Fatal()
		}
	})

	t.Run("Return false if not found", func(t *testing.T) {
		topic := ChangeResource{}
		ownerUserGroupID := "ug4e3b20cee48"
		topic.OwnerUserGroupID = &ownerUserGroupID

		user := StateResource{}
		user.Values.InternalUserID = "abc"

		ok := isUserGroupMemberFromState(&topic, &user, plan)
		if ok {
			t.Fatal()
		}
	})
}

func TestUnit_isUserGroupMemberFromConfig(t *testing.T) {
	plan := getTestPlan(t, "testdata/plan_with_unknown_owner_user_group_id.json")

	t.Run("Return true if exists", func(t *testing.T) {
		topic := ResourceChange{}
		topic.Address = "aiven_kafka_topic.foo"

		user := StateResource{}
		user.Address = "data.aiven_external_identity.alice"

		ok := isUserGroupMemberFromConfig(topic, &user, plan)
		if !ok {
			t.Fatal()
		}
	})

	t.Run("Return false if not found", func(t *testing.T) {
		topic := ResourceChange{}
		topic.Address = "aiven_kafka_topic.foo"

		user := StateResource{}
		user.Address = "data.aiven_external_identity.frank"

		ok := isUserGroupMemberFromConfig(topic, &user, plan)
		if ok {
			t.Fatal()
		}
	})

	t.Run("Return false if owner address not found", func(t *testing.T) {
		topic := ResourceChange{}
		topic.Address = "aiven_kafka_topic.magic"

		user := StateResource{}
		user.Address = "data.aiven_external_identity.frank"

		ok := isUserGroupMemberFromConfig(topic, &user, plan)
		if ok {
			t.Fatal()
		}
	})

	t.Run("Return false if user is nil", func(t *testing.T) {
		topic := ResourceChange{}
		topic.Address = "aiven_kafka_topic.foo"

		ok := isUserGroupMemberFromConfig(topic, nil, plan)
		if ok {
			t.Fatal()
		}
	})
}

func TestUnit_newRequestError(t *testing.T) {
	t.Run("Return error about requesting", func(t *testing.T) {
		address := "aiven_kafka_topic.foo"
		tags := make([]Tag, 0)
		err := newRequestError(address, tags)
		if err.Error != "requesting user is not a member of the owner group" {
			t.Error()
		}
		if err.Address != address {
			t.Error()
		}
	})
}

func TestUnit_newApproveError(t *testing.T) {
	t.Run("Return error about approving", func(t *testing.T) {
		address := "aiven_kafka_topic.foo"
		tags := make([]Tag, 0)
		err := newApproveError(address, tags)
		if err.Error != "approval is required from a member of the owner group" {
			t.Error()
		}
		if err.Address != address {
			t.Error()
		}
	})
}
