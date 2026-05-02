package health

import "time"

type HealthCheck struct {
	Status      Status
	Version     string
	Uptime      time.Duration
	Timestamp   time.Time
	ServiceName string
}

// Component represents a single dependency probed during readiness checks.
type Component struct {
	Name    string
	Status  Status
	Message string
	Latency time.Duration
}

type Readiness struct {
	Status     Status
	Components []Component
	Timestamp  time.Time
}
