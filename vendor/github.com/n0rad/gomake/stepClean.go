package gomake

import (
	"github.com/n0rad/go-erlog/logs"
	"github.com/spf13/cobra"
	"os"
)

type StepClean struct {
}

func (c *StepClean) Init() error {
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

	return cmd
}
