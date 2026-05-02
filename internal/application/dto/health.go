package dto

type HealthOutput struct {
	Status      string `json:"status"`
	Version     string `json:"version"`
	Uptime      string `json:"uptime"`
	ServiceName string `json:"service_name"`
	Timestamp   string `json:"timestamp"`
}

type ComponentOutput struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Latency string `json:"latency,omitempty"`
}

type ReadinessOutput struct {
	Status     string            `json:"status"`
	Components []ComponentOutput `json:"components"`
	Timestamp  string            `json:"timestamp"`
}
