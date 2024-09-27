package main

import (
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
			ExpectStdout: "{\"ok\":true,\"errors\":[]}\n",
			ExpectStderr: "",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			stdout, stderr, runErr := runCommand(dir, test.Args)
			if runErr != nil {
				t.Fatalf("Command execution failed: %v", runErr)
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
	if expected != "" && actual != expected {
		t.Errorf("Expected %s: %s, got: %s", name, expected, actual)
	}
}
