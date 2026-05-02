package ports

import (
	"context"

	"github.com/KryptoStorage/ms-storage/internal/application/dto"
)

// ReadinessProbe is implemented by components that must be reachable for the
// service to be considered ready (DB pool, cache, message bus, etc.). The
// returned ComponentOutput is reported in the /readyz response.
type ReadinessProbe interface {
	Name() string
	Probe(ctx context.Context) dto.ComponentOutput
}

type HealthPort interface {
	GetHealth(ctx context.Context) dto.HealthOutput
	GetReadiness(ctx context.Context) dto.ReadinessOutput
}
