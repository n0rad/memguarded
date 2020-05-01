package gomake

import "github.com/spf13/cobra"

type Step interface {
	Name() string
	Init() error
	GetCommand() *cobra.Command
	//ensureRun()
}
