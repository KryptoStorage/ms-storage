package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KryptoStorage/ms-storage/internal/application/dto"
	"github.com/rs/zerolog"
)

type fakeUseCase struct {
	healthOut    dto.HealthOutput
	readinessOut dto.ReadinessOutput
}

func (f fakeUseCase) GetHealth(_ context.Context) dto.HealthOutput       { return f.healthOut }
func (f fakeUseCase) GetReadiness(_ context.Context) dto.ReadinessOutput { return f.readinessOut }

func newHandler(uc fakeUseCase) *HealthHandler {
	l := zerolog.New(io.Discard)
	return NewHealthHandler(uc, &l)
}

func TestLivenessReturns200(t *testing.T) {
	h := newHandler(fakeUseCase{healthOut: dto.HealthOutput{Status: "healthy", Version: "1"}})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/livez", nil)
	h.Liveness(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body dto.HealthOutput
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Status != "healthy" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestReadinessStatusMapping(t *testing.T) {
	cases := map[string]int{
		"healthy":   http.StatusOK,
		"degraded":  http.StatusServiceUnavailable,
		"unhealthy": http.StatusServiceUnavailable,
	}
	for status, want := range cases {
		t.Run(status, func(t *testing.T) {
			h := newHandler(fakeUseCase{readinessOut: dto.ReadinessOutput{Status: status}})
			rec := httptest.NewRecorder()
			h.Readiness(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
			if rec.Code != want {
				t.Fatalf("status=%s expected %d, got %d", status, want, rec.Code)
			}
		})
	}
}
