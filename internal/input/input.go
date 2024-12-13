package input

import (
	"flag"
	"fmt"
	"strings"
)

type Input struct {
	Plan      string
	Requester string
	Approvers []string
}

func NewInput(args []string) (*Input, error) {
	flags := flag.NewFlagSet("checker", flag.ExitOnError)

	plan := flags.String("plan", "", "path to a file with terraform plan output in json format")
	requester := flags.String("requester", "", "user identified as the requester of the change")
	approvers := flags.String("approvers", "", "comma separated list of users identified as the approvers of the change")

	if err := flags.Parse(args); err != nil {
		return nil, fmt.Errorf("invalid arguments")
	}

	if *plan == "" {
		return nil, fmt.Errorf("plan is a required argument")
	}

	return &Input{
		Plan:      *plan,
		Requester: *requester,
		Approvers: strings.Split(*approvers, ","),
	}, nil
}
