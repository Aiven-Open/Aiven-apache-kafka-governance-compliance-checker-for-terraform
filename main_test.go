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

func TestMain(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	tests := []TestCase{
		{
			Name: "Basic",
			Args: Args{
				Requester: "alice",
				Approvers: "bob,charlie",
				Plan:      "data/plan.json",
			},
			ExpectStdout: `{"ok":true,"errors":\[\]}`,
			ExpectStderr: "",
		},
		{
			Name: "MissingArguments",
			Args: Args{
				Requester: "",
				Approvers: "bob,charlie",
				Plan:      "data/plan.json",
			},
			ExpectStdout: "",
			ExpectStderr: "Missing required arguments\nexit status 1",
		},
		{
			Name: "PlanFileDoesNotExist",
			Args: Args{
				Requester: "alice",
				Approvers: "bob,charlie",
				Plan:      "data/nonexistent_plan.json",
			},
			ExpectStdout: "",
			ExpectStderr: `open .*data/nonexistent_plan.json: no such file or directory\nexit status 1`,
		},
		{
			Name: "RequesterIsNil",
			Args: Args{
				Requester: "nonexistent_user",
				Approvers: "bob,charlie",
				Plan:      "data/plan.json",
			},
			ExpectStdout: `{"ok":false,"errors":\[{"error":"requesting user is not a member of the owner group","address":"aiven_kafka_topic.foo"}\]}`, //nolint:lll
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
