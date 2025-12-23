package ports

import "context"

// HealthChecker is used to probe dependencies.
type HealthChecker interface {
	Health(ctx context.Context) error
}
