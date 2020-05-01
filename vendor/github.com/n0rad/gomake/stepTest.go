package gomake

import (
	"github.com/spf13/cobra"
)

type StepTest struct {
}

func (c *StepTest) Init() error {
	return nil
}

func (c *StepTest) Name() string {
	return "test"
}

func (c *StepTest) GetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "run tests",
		RunE: commandDurationWrapper(func(cmd *cobra.Command, args []string) error {
			return ExecShell("go test $(go list ./... | grep -v '/vendor/')")
		}),
	}
	return cmd
}
