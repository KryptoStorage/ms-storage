package health

import "time"

type HealthCheck struct {
	Status      Status   `json:"status"`
	Version     string   `json:"version"`
	Uptime      string   `json:"uptime"`
	Timestamp   time.Time `json:"timestamp"`
	ServiceName string   `json:"service_name"`
}
