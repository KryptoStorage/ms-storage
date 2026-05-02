package dto

import "ms-storage/internal/domain/health"

type HealthOutput struct {
	Status      health.Status `json:"status"`
	Version     string `json:"version"`
	Uptime      string `json:"uptime"`
	ServiceName string `json:"service_name"`
}

type SyncOutput struct {
	Status      health.Status `json:"status"`
	LastSync    string `json:"last_sync"`
	NextSync    string `json:"next_sync"`
	State       string `json:"state"`
	Description string `json:"description,omitempty"`
}
