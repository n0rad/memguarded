package runner

import (
	"bytes"
	"github.com/n0rad/go-erlog/logs"
	"io"
	"os/exec"
	"strings"
)

var Local Exec = LocalExec{}

type LocalExec struct {
	//UnSudo bool
}

func (s LocalExec) String() string {
	return "local"
}

func (s LocalExec) Close() {}

func (s LocalExec) ExecGetStd(head string, args ...string) (string, error) {
	stdout, stderr, err := s.ExecGetStdoutStderr(head, args...)
	stdout += stderr
	return stdout, err
}

func (s LocalExec) ExecGetStdout(head string, args ...string) (string, error) {
	stdout, _, err := s.ExecGetStdoutStderr(head, args...)
	return stdout, err
}

func (s LocalExec) ExecGetStdoutStderr(head string, args ...string) (string, string, error) {
	return s.ExecSetStdinGetStdoutStderr(nil, head, args...)
}


func (s LocalExec) ExecSetStdinGetStdoutStderr(stdin io.Reader, head string, args ...string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	//if !s.UnSudo {
	//	a := append([]string{}, head)
	//	args = append(a, args...)
	//	head = "sudo"
	//}

	if logs.IsDebugEnabled() {
		logs.WithField("command", strings.Join([]string{head, " ", strings.Join(args, " ")}, " ")).Debug("Running command")
	}
	cmd := exec.Command(head, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = stdin
	cmd.Start()
	err := cmd.Wait()
	return strings.TrimSpace(stdout.String()), stderr.String(), err
}

/////

func (s LocalExec) ExecShellGetStd(cmd string) (string, error) {
	stdout, stderr, err := s.ExecGetStdoutStderr("bash", "-o", "pipefail", "-c", cmd)
	stdout += stderr
	return stdout, err
}

func (s LocalExec) ExecShellGetStdout(cmd string) (string, error) {
	return s.ExecGetStdout("bash", "-o", "pipefail", "-c", cmd)
}

