package utils

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// CommandResult 命令执行结果
type CommandResult struct {
	Command  string        `json:"command"`
	Stdout   string        `json:"stdout"`
	Stderr   string        `json:"stderr"`
	ExitCode int           `json:"exit_code"`
	Duration time.Duration `json:"duration"`
	Error    string        `json:"error,omitempty"`
}

// ExecuteCommand 执行命令
func ExecuteCommand(command string, timeout time.Duration) *CommandResult {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result := &CommandResult{
		Command: command,
	}

	startTime := time.Now()
	defer func() {
		result.Duration = time.Since(startTime)
	}()

	// 根据操作系统选择shell
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result.Stdout = strings.TrimSpace(stdout.String())
	result.Stderr = strings.TrimSpace(stderr.String())

	if err != nil {
		result.Error = err.Error()
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.ExitCode = -1
		}
	} else {
		result.ExitCode = 0
	}

	return result
}

// ExecuteCommandWithInput 执行带输入的命令
func ExecuteCommandWithInput(command, input string, timeout time.Duration) *CommandResult {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result := &CommandResult{
		Command: command,
	}

	startTime := time.Now()
	defer func() {
		result.Duration = time.Since(startTime)
	}()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = strings.NewReader(input)

	err := cmd.Run()

	result.Stdout = strings.TrimSpace(stdout.String())
	result.Stderr = strings.TrimSpace(stderr.String())

	if err != nil {
		result.Error = err.Error()
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.ExitCode = -1
		}
	} else {
		result.ExitCode = 0
	}

	return result
}

// ValidateCommand 验证命令是否安全
func ValidateCommand(command string) error {
	// 基本的命令安全检查
	dangerousCommands := []string{
		"rm -rf /",
		"format",
		"del /f /s /q",
		"shutdown",
		"reboot",
		"halt",
	}

	lowerCmd := strings.ToLower(strings.TrimSpace(command))
	for _, dangerous := range dangerousCommands {
		if strings.Contains(lowerCmd, dangerous) {
			return fmt.Errorf("dangerous command detected: %s", dangerous)
		}
	}

	return nil
}
