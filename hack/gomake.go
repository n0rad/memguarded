package main

import "github.com/n0rad/gomake"

func main() {
	gomake.ProjectBuilder().
		MustBuild().MustExecute()
}
