package gomake

import (
	"bytes"
	"github.com/n0rad/go-erlog/data"
	"github.com/n0rad/go-erlog/errs"
	"github.com/n0rad/go-erlog/logs"
	"io"
	"os"
	"os/exec"
	"strings"
)

func Exec(head string, args ...string) error {
	return ExecStdinStdoutStderr(os.Stdin, os.Stdout, os.Stderr, head, args...)
}

func ExecShell(cmd string) error {
	return Exec("sh", "-c", cmd)
}

func ExecGetStd(head string, args ...string) (string, error) {
	stdout, stderr, err := ExecGetStdoutStderr(head, args...)
	stdout += stderr
	return stdout, err
}

func ExecGetStdout(head string, args ...string) (string, error) {
	stdout, _, err := ExecGetStdoutStderr(head, args...)
	return stdout, err
}

func ExecGetStdoutStderr(head string, args ...string) (string, string, error) {
	return ExecSetStdinGetStdoutStderr(nil, head, args...)
}

func ExecSetStdinGetStdoutStderr(stdin io.Reader, head string, args ...string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := ExecStdinStdoutStderr(stdin, &stdout, &stderr, head, args...)
	return strings.TrimSpace(stdout.String()), stderr.String(), err
}

func ExecStdinStdoutStderr(stdin io.Reader, stdout io.Writer, stderr io.Writer, head string, args ...string) error {
	commandDebug := strings.Join([]string{head, " ", strings.Join(args, " ")}, " ")
	if logs.IsDebugEnabled() {
		logs.WithField("command", commandDebug).Debug("Running command")
	}
	cmd := exec.Command(head, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = stdin
	if err := cmd.Start(); err != nil {
		return errs.WithEF(err, data.WithField("command", commandDebug), "Failed to start command")
	}
	return cmd.Wait()
}

/////

func ExecShellGetStd(cmd string) (string, error) {
	stdout, stderr, err := ExecGetStdoutStderr("bash", "-o", "pipefail", "-c", cmd)
	stdout += stderr
	return stdout, err
}
