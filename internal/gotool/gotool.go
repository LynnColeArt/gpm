package gotool

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

func Run(dir string, stdout, stderr io.Writer, env []string, args ...string) error {
	cmd := command(dir, stdout, stderr, env, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go %v: %w", args, err)
	}

	return nil
}

func Output(dir string, env []string, args ...string) ([]byte, error) {
	cmd := command(dir, nil, nil, env, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go %v: %w", args, err)
	}

	return output, nil
}

func command(dir string, stdout, stderr io.Writer, env []string, args ...string) *exec.Cmd {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = os.Stdin

	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	return cmd
}
