package gomake

import (
	"github.com/n0rad/go-erlog/logs"
	"github.com/spf13/cobra"
	"os"
)

type StepClean struct {
	project *Project
}

func (c *StepClean) Init(project *Project) error {
	c.project = project
	return nil
}

func (c *StepClean) Name() string {
	return "clean"
}

func (c *StepClean) GetCommand() *cobra.Command {
	var tools bool

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "clean build directory",
		PreRunE: commandDurationWrapper(func(cmd *cobra.Command, args []string) error {
			logs.Info("Cleaning build")
			if err := os.RemoveAll("./dist/"); err != nil {
				return err
			}

			if tools {
				logs.Info("Cleaning tools")
				if err := os.RemoveAll("./dist-tools/"); err != nil {
					return err
				}
			}
			return nil
		}),
		RunE: commandDurationWrapper(func(cmd *cobra.Command, args []string) error {
			logs.Info("Cleaning build")
			if err := os.RemoveAll("./dist/"); err != nil {
				return err
			}

			if tools {
				logs.Info("Cleaning tools")
				if err := os.RemoveAll("./dist-tools/"); err != nil {
					return err
				}
			}
			return nil
		}),
	}

	cmd.Flags().BoolVarP(&tools, "tools", "t", false, "Also clean build tools")
	//cmd.AddCommand(c.project.MustGetCommand("build"))
	//cmd.AddCommand(c.project.MustGetCommand("test"))
	//cmd.AddCommand(c.project.MustGetCommand("check"))
	//cmd.AddCommand(c.project.MustGetCommand("release"))

	return cmd
}
