package cliprovider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	maximumProcessStdout = 2_000_000
	maximumProcessStderr = 256_000
)

var errProcessOutputTooLarge = errors.New("process output exceeded the safe limit")

// Command is an argv-based process request. Path and Args are always passed
// directly to the operating system; no command line is assembled for a shell.
type Command struct {
	Path  string
	Args  []string
	Dir   string
	Stdin []byte
	// Env is the exact child-process environment. A nil value inherits the
	// current process environment; generation workers always pass a non-nil,
	// allowlisted value so unrelated parent secrets are not exposed to the CLI.
	Env []string
}

type ProcessResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// Runner makes command construction independently testable and prevents the
// application layer from falling back to shell strings.
type Runner interface {
	LookPath(file string) (string, error)
	Run(ctx context.Context, command Command) (ProcessResult, error)
}

type OSRunner struct{}

func (OSRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (OSRunner) Run(ctx context.Context, command Command) (ProcessResult, error) {
	if strings.TrimSpace(command.Path) == "" {
		return ProcessResult{}, errors.New("process path is empty")
	}
	cmd := exec.CommandContext(ctx, command.Path, command.Args...)
	cmd.Dir = command.Dir
	cmd.Stdin = bytes.NewReader(command.Stdin)
	if command.Env != nil {
		cmd.Env = command.Env
	}
	stdout := newLimitedBuffer(maximumProcessStdout)
	stderr := newLimitedBuffer(maximumProcessStderr)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// Do not wait forever for inherited pipe handles after cancellation.
	cmd.WaitDelay = 2 * time.Second
	err := cmd.Run()
	result := ProcessResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes(), ExitCode: 0}
	if err == nil && (stdout.overflow || stderr.overflow) {
		err = errProcessOutputTooLarge
	}
	if err != nil {
		result.ExitCode = -1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
		}
	}
	return result, err
}

type limitedBuffer struct {
	bytes.Buffer
	remaining int
	overflow  bool
}

func newLimitedBuffer(limit int) limitedBuffer {
	return limitedBuffer{remaining: limit}
}

func (buffer *limitedBuffer) Write(value []byte) (int, error) {
	originalLength := len(value)
	if len(value) > buffer.remaining {
		value = value[:buffer.remaining]
		buffer.overflow = true
	}
	if len(value) > 0 {
		_, _ = buffer.Buffer.Write(value)
		buffer.remaining -= len(value)
	}
	return originalLength, nil
}

type invocation struct {
	path       string
	prefixArgs []string
}

func resolveInvocation(runner Runner, provider Provider, override string) (invocation, error) {
	override = strings.TrimSpace(override)
	if override != "" {
		if strings.IndexByte(override, 0) >= 0 {
			return invocation{}, errors.New("executable path contains a null byte")
		}
		if runtime.GOOS == "windows" && isWindowsScript(override) {
			if provider == ProviderClaude {
				return resolveWindowsClaudeShim(runner, override)
			}
			return invocation{}, errors.New("script launchers are not supported on Windows")
		}
		return invocation{path: override}, nil
	}

	if provider == ProviderClaude && runtime.GOOS == "windows" {
		if nativePath, err := runner.LookPath("claude.exe"); err == nil {
			return invocation{path: nativePath}, nil
		}
		shimPath, err := runner.LookPath("claude.cmd")
		if err != nil {
			return invocation{}, fmt.Errorf("claude executable not found")
		}
		return resolveWindowsClaudeShim(runner, shimPath)
	}

	name := string(provider)
	resolved, err := runner.LookPath(name)
	if err != nil {
		return invocation{}, fmt.Errorf("%s executable not found", provider)
	}
	if runtime.GOOS == "windows" && isWindowsScript(resolved) {
		return invocation{}, fmt.Errorf("%s resolved to a script launcher", provider)
	}
	return invocation{path: resolved}, nil
}

func isWindowsScript(path string) bool {
	extension := strings.ToLower(filepath.Ext(path))
	return extension == ".cmd" || extension == ".bat" || extension == ".ps1"
}

// The npm-installed Claude launcher on Windows is a cmd shim. Running it would
// necessarily involve cmd.exe. Instead, invoke the package's fixed native
// executable (current releases), or its fixed JS entry point through node.exe
// (older releases), so user brief/schema text is never parsed by a shell.
func resolveWindowsClaudeShim(runner Runner, shimPath string) (invocation, error) {
	if !strings.EqualFold(filepath.Ext(shimPath), ".cmd") {
		return invocation{}, errors.New("Claude override must be claude.exe or claude.cmd on Windows")
	}
	packageRoot := filepath.Join(filepath.Dir(shimPath), "node_modules", "@anthropic-ai", "claude-code")
	nativePath := filepath.Join(packageRoot, "bin", "claude.exe")
	if info, err := os.Stat(nativePath); err == nil && !info.IsDir() {
		return invocation{path: nativePath}, nil
	}
	entryPoint := filepath.Join(packageRoot, "cli.js")
	info, err := os.Stat(entryPoint)
	if err != nil || info.IsDir() {
		return invocation{}, errors.New("Claude npm entry point was not found beside claude.cmd")
	}
	nodePath, err := runner.LookPath("node.exe")
	if err != nil {
		return invocation{}, errors.New("node.exe is required for the npm-installed Claude CLI")
	}
	return invocation{path: nodePath, prefixArgs: []string{entryPoint}}, nil
}
