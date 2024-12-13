package test

import (
	"aiven/terraform/governance/compliance/checker/internal/input"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInput_NewInput(t *testing.T) {

	t.Run("Parses CLI args into Input and returns a pointer to it", func(t *testing.T) {
		args, err := input.NewInput([]string{"-plan=plan.json", "-requester=alice", "-approvers=bob,charlie"})
		assert.Equal(t, err, nil)
		assert.Equal(t, args.Plan, "plan.json")
		assert.Equal(t, args.Requester, "alice")
		assert.Equal(t, args.Approvers, []string{"bob", "charlie"})
	})

	t.Run("Returns error if path is not provided", func(t *testing.T) {
		_, err := input.NewInput([]string{"-requester=alice", "-approvers=bob"})
		assert.Equal(t, err.Error(), "plan is a required argument")
	})

}
