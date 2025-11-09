// fabricx-core/pkg/errors/errors.go
package errors

import (
	"errors"
	"fmt"
)

// Common error types for better error handling
var (
	// ErrBinaryMissing is returned when a required binary is not found
	ErrBinaryMissing = errors.New("required binary not found")
	
	// ErrTimeout is returned when an operation times out
	ErrTimeout = errors.New("operation timeout")
	
	// ErrNetworkNotFound is returned when a network ID doesn't exist
	ErrNetworkNotFound = errors.New("network not found")
	
	// ErrDockerUnavailable is returned when Docker is not available
	ErrDockerUnavailable = errors.New("docker is not available")
	
	// ErrContainerFailed is returned when a container operation fails
	ErrContainerFailed = errors.New("container operation failed")
	
	// ErrCryptoGenFailed is returned when crypto generation fails
	ErrCryptoGenFailed = errors.New("crypto generation failed")
	
	// ErrChaincodeDeployFailed is returned when chaincode deployment fails
	ErrChaincodeDeployFailed = errors.New("chaincode deployment failed")
	
	// ErrTransactionFailed is returned when a transaction fails
	ErrTransactionFailed = errors.New("transaction failed")
	
	// ErrInvalidConfig is returned when configuration is invalid
	ErrInvalidConfig = errors.New("invalid configuration")
)

// FabricXError wraps errors with additional context
type FabricXError struct {
	Op      string // Operation that failed
	Err     error  // Underlying error
	Context map[string]interface{} // Additional context
}

// Error implements the error interface
func (e *FabricXError) Error() string {
	if len(e.Context) > 0 {
		return fmt.Sprintf("%s failed: %v (context: %+v)", e.Op, e.Err, e.Context)
	}
	return fmt.Sprintf("%s failed: %v", e.Op, e.Err)
}

// Unwrap returns the underlying error
func (e *FabricXError) Unwrap() error {
	return e.Err
}

// Is checks if the error matches the target
func (e *FabricXError) Is(target error) bool {
	return errors.Is(e.Err, target)
}

// NewError creates a new FabricXError
func NewError(op string, err error, context ...map[string]interface{}) *FabricXError {
	e := &FabricXError{
		Op:  op,
		Err: err,
	}
	
	if len(context) > 0 {
		e.Context = context[0]
	}
	
	return e
}

// Wrap wraps an error with operation context
func Wrap(op string, err error) error {
	if err == nil {
		return nil
	}
	return &FabricXError{
		Op:  op,
		Err: err,
	}
}

// WrapWithContext wraps an error with operation and additional context
func WrapWithContext(op string, err error, context map[string]interface{}) error {
	if err == nil {
		return nil
	}
	return &FabricXError{
		Op:      op,
		Err:     err,
		Context: context,
	}
}

// IsBinaryMissing checks if error is due to missing binary
func IsBinaryMissing(err error) bool {
	return errors.Is(err, ErrBinaryMissing)
}

// IsTimeout checks if error is due to timeout
func IsTimeout(err error) bool {
	return errors.Is(err, ErrTimeout)
}

// IsNetworkNotFound checks if error is due to network not found
func IsNetworkNotFound(err error) bool {
	return errors.Is(err, ErrNetworkNotFound)
}

// IsDockerUnavailable checks if error is due to Docker unavailability
func IsDockerUnavailable(err error) bool {
	return errors.Is(err, ErrDockerUnavailable)
}