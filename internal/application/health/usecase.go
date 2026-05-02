package health

import (
	"context"
	"time"

	"github.com/KryptoStorage/ms-storage/internal/application/dto"
	"github.com/KryptoStorage/ms-storage/internal/application/ports"
	"github.com/KryptoStorage/ms-storage/internal/domain/health"
)

type UseCase struct {
	version     string
	serviceName string
	startTime   time.Time
	probes      []ports.ReadinessProbe
}

type Options struct {
	Version     string
	ServiceName string
	Probes      []ports.ReadinessProbe
}

func New(opts Options) ports.HealthPort {
	return &UseCase{
		version:     opts.Version,
		serviceName: opts.ServiceName,
		startTime:   time.Now(),
		probes:      opts.Probes,
	}
}

func (u *UseCase) GetHealth(_ context.Context) dto.HealthOutput {
	now := time.Now().UTC()
	return dto.HealthOutput{
		Status:      string(health.StatusHealthy),
		Version:     u.version,
		Uptime:      time.Since(u.startTime).Truncate(time.Second).String(),
		ServiceName: u.serviceName,
		Timestamp:   now.Format(time.RFC3339),
	}
}

func (u *UseCase) GetReadiness(ctx context.Context) dto.ReadinessOutput {
	overall := health.StatusHealthy
	components := make([]dto.ComponentOutput, 0, len(u.probes))

	for _, p := range u.probes {
		c := p.Probe(ctx)
		switch c.Status {
		case string(health.StatusUnhealthy):
			overall = health.StatusUnhealthy
		case string(health.StatusDegraded):
			if overall != health.StatusUnhealthy {
				overall = health.StatusDegraded
			}
		}
		components = append(components, c)
	}

	return dto.ReadinessOutput{
		Status:     string(overall),
		Components: components,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}
}
