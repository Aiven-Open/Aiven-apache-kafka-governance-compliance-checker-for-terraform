package main

import (
	"testing"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type (
	TestCase struct {
		Name	string
		Args	Args
		ExpectStdout	string
		ExpectStderr	string
	}
	Args struct {
		Requester	string
		Approvers	string
		Plan		string
	}
)

func TestMain(t *testing.T) {
	dir, _ := os.Getwd()

	tests := make([]TestCase, 0)
	tests = append(tests, TestCase{ 
		Name: "Basic", 
		Args: Args{
			Requester: "alice", 
			Approvers: "bob,charlie",
			Plan: "data/plan.json",
		},
		ExpectStderr: "ok",
	})

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			args := make([]string, 0)
			args = append(args, "run")
			args = append(args, "main.go")
			args = append(args, fmt.Sprintf("-requester=\"%s\"", test.Args.Requester))
    		args = append(args, fmt.Sprintf("-approvers=\"%s\"", test.Args.Approvers))
    		args = append(args, fmt.Sprintf("-plan=\"%s\"", filepath.Join(dir, test.Args.Plan)))		

			var stdoutBuffer strings.Builder
			var stderrBuffer strings.Builder

			cmd := exec.Command("go", args...)
			cmd.Stdout = &stdoutBuffer
			cmd.Stderr = &stderrBuffer
			err := cmd.Run()
			if err != nil {
				t.Errorf("%s", err.Error())
			}

			if test.ExpectStdout != "" {
				stdout := stdoutBuffer.String()
				t.Errorf("%s != %s", test.ExpectStdout, stdout)
			}

			if test.ExpectStderr != "" {
				stderr := stderrBuffer.String()
				t.Errorf("%s != %s", test.ExpectStderr, stderr)
			}

		})
	}
}