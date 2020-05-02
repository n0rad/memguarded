package gomake

import (
	"github.com/n0rad/go-erlog/errs"
	"github.com/n0rad/go-erlog/logs"
	_ "github.com/n0rad/go-erlog/register"
	"github.com/spf13/cobra"
	"os"
	"time"
)

func commandDurationWrapper(f func(cmd *cobra.Command, args []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		start := time.Now()
		err := f(cmd, args)
		diff := time.Now().Sub(start)
		logs.WithField("duration", diff.Round(time.Second).String()).Info(cmd.Use + " done")
		return err
	}
}

type Project struct {
	args         []string
	steps        map[string]Step
	commandCache map[string]*cobra.Command
}

func NewProject() *Project {
	p := Project{}
	p.steps = make(map[string]Step)
	return &p
}

func (p *Project) MustGetCommand(name string) *cobra.Command {
	cmd, ok := p.commandCache[name]
	if !ok {
		step, ok := p.steps[name]
		if !ok {
			logs.WithField("step", name).Fatal("Cannot found step")
		}
		cmd = step.GetCommand()
		p.commandCache[name] = cmd
	}
	return cmd
}

func (p *Project) Init() error {
	p.commandCache = make(map[string]*cobra.Command)

	if _, ok := p.steps["clean"]; !ok {
		p.steps["clean"] = &StepClean{}
	}
	if _, ok := p.steps["build"]; !ok {
		p.steps["build"] = &StepBuild{}
	}
	if _, ok := p.steps["check"]; !ok {
		p.steps["check"] = &StepCheck{}
	}
	if _, ok := p.steps["test"]; !ok {
		p.steps["test"] = &StepTest{}
	}
	//if _, ok := p.steps["release"]; ok {
	//	p.steps["release"] = &StepRelease{}
	//}

	if len(p.args) == 0 {
		p.args = os.Args[1:]
	}
	//if len(p.args) == 0 {
	//	p.args = []string{"clean", "build", "test", "quality"}
	//}

	for i := range p.steps {
		if err := p.steps[i].Init(p); err != nil {
			return errs.WithE(err, "Failed to init Step in project")
		}
	}

	return nil
}

///////////////////////

func (p Project) MustExecute() {
	rootCommand := p.GetCommand()
	rootCommand.SetArgs(p.args)
	if err := rootCommand.Execute(); err != nil {
		logs.WithE(err).Fatal("Command failed")
	}
}

func (p Project) GetCommand() *cobra.Command {
	var logLevel string
	cmd := &cobra.Command{
		Use:           "gomake",
		Short:         "handle go project build lifecycle",
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if logLevel != "" {
				level, err := logs.ParseLevel(logLevel)
				if err != nil {
					return err
				}
				logs.SetLevel(level)
			}
			return nil
		},
	}

	cmd.PersistentFlags().StringVarP(&logLevel, "log-level", "L", "", "Set log level")

	for name := range p.steps {
		cmd.AddCommand(p.MustGetCommand(name))
	}

	return cmd
}

///////////////////////////////////////

type Builder struct {
	*Project
}

func ProjectBuilder() Builder {
	builder := Builder{}
	builder.Project = NewProject()
	return builder
}

func (p Builder) MustBuild() *Project {
	project := p.Project
	if err := project.Init(); err != nil {
		logs.WithE(err).Fatal("Failed to prepare project")
	}
	return project
}

func (p Builder) WithStep(step Step) Builder {
	p.steps[step.Name()] = step
	return p
}

func (p Builder) WithArgs(args []string) Builder {
	p.args = args
	return p
}
