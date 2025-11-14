// pkg/types/types.go

// Shared types to avoid circular dependencies
package types

// Network interface for docker manager to avoid importing network package
type Network interface {
	GetID() string
	GetConfigPath() string
	GetOrgs() interface{}
	GetOrderers() interface{}
	Cleanup() error
}