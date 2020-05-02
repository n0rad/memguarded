package gomake

import (
	"github.com/spf13/cobra"
)

type StepTest struct {
	project *Project
}

func (c *StepTest) Init(project *Project) error {
	c.project = project
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

	//cmd.AddCommand(c.project.MustGetCommand("build"))
	//cmd.AddCommand(c.project.MustGetCommand("check"))
	//cmd.AddCommand(c.project.MustGetCommand("release"))

	return cmd
}
