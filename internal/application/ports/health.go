package ports

import (
	"ms-storage/internal/application/dto"
)

type HealthPort interface {
	GetHealth() dto.HealthOutput
	GetSync() dto.SyncOutput
}
