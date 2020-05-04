package main

import "github.com/n0rad/gomake"

func main() {
	gomake.ProjectBuilder().
		WithStep(&gomake.StepBuild{
			BinaryName: "memguarded-server",
			Package: "./cmd/server",
		}).
		//WithStep(&gomake.StepBuild{
		//	BinaryName: "memguarded",
		//	Package: "./cmd/cli",
		//}).
		MustBuild().MustExecute()
}
