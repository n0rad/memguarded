package main

import "github.com/n0rad/gomake"

func main() {
	gomake.ProjectBuilder().
		WithStep(&gomake.StepBuild{
			Upx: true,
			Package: "./cmd",
		}).
		MustBuild().MustExecute()
}
