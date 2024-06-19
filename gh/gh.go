// Package gh is a library for CLI Go applications to help interface with the gh CLI tool,
// and the GitHub API.
//
// Note that the examples in this package assume gh and git are installed. They do not run in
// the Go Playground used by pkg.go.dev.

/*
Directly taken from: https://github.com/cli/go-gh
LICENSE:
MIT License

Copyright (c) 2021 GitHub Inc.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/
package gh

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/cli/safeexec"
)

// Exec invokes a gh command in a subprocess and captures the output and error streams.
func Exec(args ...string) (stdout, stderr bytes.Buffer, err error) {
	ghExe, err := Path()
	if err != nil {
		return
	}
	err = run(context.Background(), ghExe, nil, nil, &stdout, &stderr, args)
	return
}

func ExecIn(workingDirectory string, args ...string) (stdout, stderr bytes.Buffer, err error) {
	ghExe, err := Path()
	if err != nil {
		return
	}
	cmd := CreateCommand(context.Background(), ghExe, nil, nil, &stdout, &stderr, args)
	cmd.Dir = workingDirectory
	err = cmd.Run()
	return
}

// ExecContext invokes a gh command in a subprocess and captures the output and error streams.
func ExecContext(ctx context.Context, args ...string) (stdout, stderr bytes.Buffer, err error) {
	ghExe, err := Path()
	if err != nil {
		return
	}
	err = run(ctx, ghExe, nil, nil, &stdout, &stderr, args)
	return
}

// Exec invokes a gh command in a subprocess with its stdin, stdout, and stderr streams connected to
// those of the parent process. This is suitable for running gh commands with interactive prompts.
func ExecInteractive(ctx context.Context, args ...string) error {
	ghExe, err := Path()
	if err != nil {
		return err
	}
	return run(ctx, ghExe, nil, os.Stdin, os.Stdout, os.Stderr, args)
}

// Path searches for an executable named "gh" in the directories named by the PATH environment variable.
// If the executable is found the result is an absolute path.
func Path() (string, error) {
	if ghExe := os.Getenv("GH_PATH"); ghExe != "" {
		return ghExe, nil
	}
	return safeexec.LookPath("gh")
}

func CreateCommand(ctx context.Context, ghExe string, env []string, stdin io.Reader, stdout, stderr io.Writer, args []string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, ghExe, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if env != nil {
		cmd.Env = env
	}
	return cmd
}

func run(ctx context.Context, ghExe string, env []string, stdin io.Reader, stdout, stderr io.Writer, args []string) error {
	cmd := CreateCommand(ctx, ghExe, env, stdin, stdout, stderr, args)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh execution failed: %w", err)
	}
	return nil
}
