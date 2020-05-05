package main

import "github.com/n0rad/gomake"

func main() {
	gomake.ProjectBuilder().
		WithStep(&gomake.StepBuild{
			Programs: []gomake.Program{
				{
					BinaryName: "memguarded-server",
					Package:    "./cmd/server",
				},
				{
					BinaryName: "memguarded",
					Package:    "./cmd/cli",
				},
			},
		}).
		WithStep(&gomake.StepRelease{
			OsArchRelease: []string{"linux-amd64"},
			Upx:           gomake.True,
		}).
		MustBuild().MustExecute()
}
