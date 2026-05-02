package health

import (
	"ms-storage/internal/application/dto"
	"ms-storage/internal/application/ports"
	"ms-storage/internal/domain/health"
	"time"
)

type HealthUseCase struct {
	version     string
	serviceName string
	startTime   time.Time
}

func NewHealthUseCase(version, serviceName string) ports.HealthPort {
	return &HealthUseCase{
		version:     version,
		serviceName: serviceName,
		startTime:   time.Now(),
	}
}

func (h *HealthUseCase) GetHealth() dto.HealthOutput {
	uptime := time.Since(h.startTime)

	return dto.HealthOutput{
		Status:      health.StatusHealthy,
		Version:     h.version,
		Uptime:      uptime.String(),
		ServiceName: h.serviceName,
	}
}

func (h *HealthUseCase) GetSync() dto.SyncOutput {
	now := time.Now().UTC()

	return dto.SyncOutput{
		Status:      health.StatusHealthy,
		LastSync:    now.Format(time.RFC3339),
		NextSync:    now.Add(30 * time.Second).Format(time.RFC3339),
		State:       "synced",
		Description: "Service is synchronized and operational",
	}
}
