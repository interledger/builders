package main

import (
	"context"
	"os"
	"os/exec"
)

// CommandExecutor interface allows for mocking exec.Command
type CommandExecutor interface {
	CommandContext(ctx context.Context, name string, args ...string) Command
	FileExists(path string) bool
}

// Command interface wraps exec.Cmd for testing
type Command interface {
	SetDir(dir string)
	CombinedOutput() ([]byte, error)
	Run() error
	GetPath() string
	GetArgs() []string
}

// RealCommandExecutor implements CommandExecutor using the real exec package
type RealCommandExecutor struct{}

func (r *RealCommandExecutor) CommandContext(ctx context.Context, name string, args ...string) Command {
	return &RealCommand{cmd: exec.CommandContext(ctx, name, args...)}
}

// RealCommand wraps exec.Cmd
type RealCommand struct {
	cmd *exec.Cmd
}

func (r *RealCommand) SetDir(dir string) {
	r.cmd.Dir = dir
}

func (r *RealCommand) CombinedOutput() ([]byte, error) {
	return r.cmd.CombinedOutput()
}

func (r *RealCommand) Run() error {
	return r.cmd.Run()
}

func (r *RealCommand) GetPath() string {
	return r.cmd.Path
}

func (r *RealCommand) GetArgs() []string {
	return r.cmd.Args
}

func (r *RealCommandExecutor) FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}