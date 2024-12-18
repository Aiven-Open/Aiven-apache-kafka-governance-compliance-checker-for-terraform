package test

import (
	"aiven/terraform/governance/compliance/checker/internal/terraform"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTerraform_NewPlan(t *testing.T) {

	t.Run("Reads the provided file path and encodes into Plan and returns a pointer to it", func(t *testing.T) {
		plan, err := terraform.NewPlan("../testdata/plan_with_known_owner_user_group_id.json")
		assert.Nil(t, err)
		assert.NotNil(t, plan)
	})

	t.Run("Returns error if path does not point to a file", func(t *testing.T) {
		plan, err := terraform.NewPlan("not-a-file")
		assert.Nil(t, plan)
		assert.NotNil(t, err)
		assert.Equal(t, err.Error(), "invalid plan JSON file")
	})

	t.Run("Returns error if path does not point to valid json file", func(t *testing.T) {
		plan, err := terraform.NewPlan("../testdata/not_json.py")
		assert.Nil(t, plan)
		assert.NotNil(t, err)
		assert.Equal(t, err.Error(), "invalid plan JSON file")
	})

}
