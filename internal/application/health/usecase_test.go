package health

import (
	"context"
	"testing"

	"github.com/KryptoStorage/ms-storage/internal/application/dto"
	"github.com/KryptoStorage/ms-storage/internal/application/ports"
)

type fakeProbe struct {
	name   string
	status string
}

func (f fakeProbe) Name() string { return f.name }
func (f fakeProbe) Probe(_ context.Context) dto.ComponentOutput {
	return dto.ComponentOutput{Name: f.name, Status: f.status}
}

func TestGetHealth(t *testing.T) {
	uc := New(Options{Version: "1.2.3", ServiceName: "ms-storage"})
	out := uc.GetHealth(context.Background())

	if out.Status != "healthy" || out.Version != "1.2.3" || out.ServiceName != "ms-storage" {
		t.Fatalf("unexpected output: %+v", out)
	}
	if out.Uptime == "" || out.Timestamp == "" {
		t.Fatalf("expected uptime/timestamp, got %+v", out)
	}
}

func TestGetReadiness(t *testing.T) {
	tests := []struct {
		name     string
		probes   []ports.ReadinessProbe
		expected string
	}{
		{"no probes", nil, "healthy"},
		{"all healthy", []ports.ReadinessProbe{
			fakeProbe{name: "db", status: "healthy"},
			fakeProbe{name: "cache", status: "healthy"},
		}, "healthy"},
		{"degraded wins over healthy", []ports.ReadinessProbe{
			fakeProbe{name: "db", status: "healthy"},
			fakeProbe{name: "cache", status: "degraded"},
		}, "degraded"},
		{"unhealthy wins over degraded", []ports.ReadinessProbe{
			fakeProbe{name: "db", status: "unhealthy"},
			fakeProbe{name: "cache", status: "degraded"},
		}, "unhealthy"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			uc := New(Options{Version: "v", ServiceName: "s", Probes: tc.probes})
			out := uc.GetReadiness(context.Background())
			if out.Status != tc.expected {
				t.Fatalf("expected %s, got %s", tc.expected, out.Status)
			}
			if len(out.Components) != len(tc.probes) {
				t.Fatalf("expected %d components, got %d", len(tc.probes), len(out.Components))
			}
		})
	}
}
