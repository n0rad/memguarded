package runner

import "io"

type Exec interface {
	ExecGetStdoutStderr(head string, args ...string) (string, string, error)
	ExecGetStdout(head string, args ...string) (string, error)
	ExecGetStd(head string, args ...string) (string, error)
	ExecSetStdinGetStdoutStderr(stdin io.Reader, head string, args ...string) (string, string, error)

	ExecShellGetStdout(cmd string) (string, error)
	ExecShellGetStd(cmd string) (string, error)

	Close()
}
