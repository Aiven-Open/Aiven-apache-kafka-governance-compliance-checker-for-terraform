package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
			Name: "Plan file needs to exist",
			Args: Args{
				Requester: "alice",
				Approvers: "bob,charlie",
				Plan:      "testdata/nonexistent_plan.json",
			},
			ExpectStdout: "",
			ExpectStderr: `open .*data/nonexistent_plan.json: no such file or directory\nexit status 1`,
		},
		{
			Name: "Plan file needs to be json",
			Args: Args{
				Requester: "alice",
				Approvers: "bob,charlie",
				Plan:      "testdata/not_json.py",
			},
			ExpectStdout: "",
			ExpectStderr: "Invalid plan JSON file",
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

func TestE2E_Plans(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	all_tests := make([]TestCase, 0)

	plans := make([]string, 0)
	plans = append(plans, "./testdata/plan_with_known_owner_user_group_id.json")
	plans = append(plans, "./testdata/plan_with_unknown_owner_user_group_id.json")

	tests := make([]TestCase, 0)
	for _, plan := range plans {
		tests = []TestCase{
			{
				Name: fmt.Sprintf("[%s] Reports error if requester identity can not be resolved", plan),
				Args: Args{
					Requester: "nonexistent_user",
					Approvers: "bob,charlie",
					Plan:      plan,
				},
				ExpectStdout: `{"ok":false,"errors":\[{"error":"requesting user is not a member of the owner group","address":"aiven_kafka_topic.foo"}\]}`, //nolint:lll
				ExpectStderr: "",
			},
			{
				Name: fmt.Sprintf("[%s] Reports error if requester is not a member of the owner user group", plan),
				Args: Args{
					Requester: "charlie",
					Approvers: "bob,charlie",
					Plan:      plan,
				},
				ExpectStdout: `{"ok":false,"errors":\[{"error":"requesting user is not a member of the owner group","address":"aiven_kafka_topic.foo"}\]}`, //nolint:lll
				ExpectStderr: "",
			},
			{
				Name: fmt.Sprintf("[%s] Does not report error if requester is a member of the owner user group", plan),
				Args: Args{
					Requester: "alice",
					Approvers: "bob,charlie",
					Plan:      plan,
				},
				ExpectStdout: `{"ok":true,"errors":\[\]}`, //nolint:lll
				ExpectStderr: "",
			},
			{
				Name: fmt.Sprintf("[%s] Reports error if approval is missing from a member of the owner user group", plan),
				Args: Args{
					Requester: "alice",
					Approvers: "frank",
					Plan:      plan,
				},
				ExpectStdout: `{"ok":false,"errors":\[{"error":"approval is required from a member of the owner group","address":"aiven_kafka_topic.foo"}\]}`, //nolint:lll
				ExpectStderr: "",
			},
			{
				Name: fmt.Sprintf("[%s] Does not consider requester as approver", plan),
				Args: Args{
					Requester: "alice",
					Approvers: "alice",
					Plan:      plan,
				},
				ExpectStdout: `{"ok":false,"errors":\[{"error":"approval is required from a member of the owner group","address":"aiven_kafka_topic.foo"}\]}`, //nolint:lll
				ExpectStderr: "",
			},
		}
		all_tests = append(all_tests, tests...)
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
	cmdArgs := []string{
		"run",
		"main.go",
		fmt.Sprintf("-requester=%s", args.Requester),
		fmt.Sprintf("-approvers=%s", args.Approvers),
		fmt.Sprintf("-plan=%s", filepath.Join(dir, args.Plan)),
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

	matched, err := regexp.MatchString(expected, actual)
	if err != nil {
		t.Fatalf("Failed to compile regex: %v", err)
	}
	if !matched {
		t.Errorf("Expected %s to match pattern: %q, but got: %q", name, expected, actual)
	}
}
