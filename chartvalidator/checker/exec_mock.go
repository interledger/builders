package main

import (
	"context"
	"strings"
)

// MockCommandExecutor captures command execution for testing
type MockCommandExecutor struct {
	LastCommand string
	LastArgs    []string
	Output      []byte
	Error       error
	BehaviorOnRun func() error
	FileExistsMap  map[string]bool
}

func (m *MockCommandExecutor) CommandContext(ctx context.Context, name string, args ...string) Command {
	m.LastCommand = name
	m.LastArgs = args
	return &MockCommand{
		executor: m,
		output:   m.Output,
		err:      m.Error,
	}
}

func (m *MockCommandExecutor) GetFullCommand() string {
	if m.LastCommand == "" {
		return ""
	}
	return m.LastCommand + " " + strings.Join(m.LastArgs, " ")
}

// MockCommand implements Command interface for testing
type MockCommand struct {
	executor *MockCommandExecutor
	output   []byte
	err      error
	dir      string
}

func (m *MockCommand) SetDir(dir string) {
	m.dir = dir
}

func (m *MockCommand) CombinedOutput() ([]byte, error) {
	return m.output, m.err
}

func (m *MockCommand) Run() error {
	if m.executor.BehaviorOnRun != nil {
		return m.executor.BehaviorOnRun()
	}
	return m.err
}

func (m *MockCommand) GetPath() string {
	return m.executor.LastCommand
}

func (m *MockCommand) GetArgs() []string {
	// Return the full args array (including the command name as args[0])
	if m.executor.LastCommand == "" {
		return []string{}
	}
	return append([]string{m.executor.LastCommand}, m.executor.LastArgs...)
}

func (m *MockCommandExecutor) FileExists(path string) bool {
	if m.FileExistsMap != nil {
		exists, found := m.FileExistsMap[path]
		if found {
			return exists
		}
	}
	// Default to true if not specified
	return true
}