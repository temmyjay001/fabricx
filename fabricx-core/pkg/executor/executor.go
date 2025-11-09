// fabricx-core/pkg/executor/executor.go
package executor

import (
	"context"
	"os/exec"
)

// Executor defines the interface for executing commands
// This allows us to mock command execution in tests
type Executor interface {
	// Execute runs a command with the given context and returns output and error
	Execute(ctx context.Context, name string, args ...string) ([]byte, error)
	
	// ExecuteCombined runs a command and returns combined stdout/stderr
	ExecuteCombined(ctx context.Context, name string, args ...string) ([]byte, error)
	
	// ExecuteStream runs a command and returns separate stdout/stderr channels
	ExecuteStream(ctx context.Context, name string, args ...string) (<-chan string, <-chan error, error)
}

// RealExecutor implements Executor using actual exec.Command
type RealExecutor struct{}

// NewRealExecutor creates a new real command executor
func NewRealExecutor() *RealExecutor {
	return &RealExecutor{}
}

// Execute runs a command with context and returns output
func (e *RealExecutor) Execute(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Output()
}

// ExecuteCombined runs a command and returns combined output
func (e *RealExecutor) ExecuteCombined(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

// ExecuteStream runs a command and streams output
func (e *RealExecutor) ExecuteStream(ctx context.Context, name string, args ...string) (<-chan string, <-chan error, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, err
	}
	
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	
	outChan := make(chan string, 100)
	errChan := make(chan error, 1)
	
	go func() {
		defer close(outChan)
		defer close(errChan)
		
		// Stream stdout
		go func() {
			buf := make([]byte, 1024)
			for {
				n, err := stdout.Read(buf)
				if n > 0 {
					select {
					case outChan <- string(buf[:n]):
					case <-ctx.Done():
						return
					}
				}
				if err != nil {
					break
				}
			}
		}()
		
		// Stream stderr
		go func() {
			buf := make([]byte, 1024)
			for {
				n, err := stderr.Read(buf)
				if n > 0 {
					select {
					case outChan <- string(buf[:n]):
					case <-ctx.Done():
						return
					}
				}
				if err != nil {
					break
				}
			}
		}()
		
		// Wait for command to complete
		if err := cmd.Wait(); err != nil {
			errChan <- err
		}
	}()
	
	return outChan, errChan, nil
}

// MockExecutor implements Executor for testing
type MockExecutor struct {
	ExecuteFunc         func(ctx context.Context, name string, args ...string) ([]byte, error)
	ExecuteCombinedFunc func(ctx context.Context, name string, args ...string) ([]byte, error)
	ExecuteStreamFunc   func(ctx context.Context, name string, args ...string) (<-chan string, <-chan error, error)
	
	// Recording for verification
	Calls []Call
}

// Call records a command execution
type Call struct {
	Name string
	Args []string
}

// NewMockExecutor creates a new mock executor
func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		Calls: make([]Call, 0),
	}
}

// Execute mocks command execution
func (m *MockExecutor) Execute(ctx context.Context, name string, args ...string) ([]byte, error) {
	m.Calls = append(m.Calls, Call{Name: name, Args: args})
	
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, name, args...)
	}
	
	return []byte("mock output"), nil
}

// ExecuteCombined mocks combined output execution
func (m *MockExecutor) ExecuteCombined(ctx context.Context, name string, args ...string) ([]byte, error) {
	m.Calls = append(m.Calls, Call{Name: name, Args: args})
	
	if m.ExecuteCombinedFunc != nil {
		return m.ExecuteCombinedFunc(ctx, name, args...)
	}
	
	return []byte("mock output"), nil
}

// ExecuteStream mocks streaming execution
func (m *MockExecutor) ExecuteStream(ctx context.Context, name string, args ...string) (<-chan string, <-chan error, error) {
	m.Calls = append(m.Calls, Call{Name: name, Args: args})
	
	if m.ExecuteStreamFunc != nil {
		return m.ExecuteStreamFunc(ctx, name, args...)
	}
	
	outChan := make(chan string, 1)
	errChan := make(chan error, 1)
	
	go func() {
		outChan <- "mock log output"
		close(outChan)
		close(errChan)
	}()
	
	return outChan, errChan, nil
}

// GetCalls returns all recorded calls
func (m *MockExecutor) GetCalls() []Call {
	return m.Calls
}

// Reset clears all recorded calls
func (m *MockExecutor) Reset() {
	m.Calls = make([]Call, 0)
}

// WasCalledWith checks if executor was called with specific command
func (m *MockExecutor) WasCalledWith(name string, args ...string) bool {
	for _, call := range m.Calls {
		if call.Name == name {
			if len(args) == 0 {
				return true
			}
			if len(call.Args) == len(args) {
				match := true
				for i, arg := range args {
					if call.Args[i] != arg {
						match = false
						break
					}
				}
				if match {
					return true
				}
			}
		}
	}
	return false
}