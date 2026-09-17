package intelkit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/restx"
)

const keyFingerprintLength = 16

func LimiterKeyPrefix(namespace, secret string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(secret)))
	return namespace + ":" + hex.EncodeToString(sum[:])[:keyFingerprintLength]
}

type Recorder struct {
	record services.CarrierIntelCallRecorder
}

func NewRecorder(record services.CarrierIntelCallRecorder) Recorder {
	return Recorder{record: record}
}

type CallOutcome struct {
	Endpoint  carrierintel.Endpoint
	DOTNumber string
	Status    int
	Started   time.Time
	Found     bool
	Units     int
	RawErr    error
	Mapped    error
}

func (r Recorder) Record(ctx context.Context, outcome *CallOutcome) {
	if r.record == nil {
		return
	}
	r.record(ctx, services.CarrierIntelCall{
		Endpoint:   outcome.Endpoint,
		DOTNumber:  outcome.DOTNumber,
		StatusCode: outcome.statusCode(),
		Latency:    time.Since(outcome.Started),
		Found:      outcome.Found,
		Units:      max(outcome.Units, 1),
		Err:        outcome.Mapped,
	})
}

func (o *CallOutcome) statusCode() int {
	if o.Status != 0 {
		return o.Status
	}
	return CallStatus(o.RawErr, o.Mapped)
}

func CallStatus(rawErr, mapped error) int {
	if rawErr == nil && mapped == nil {
		return http.StatusOK
	}
	if status := restx.StatusCode(rawErr); status != 0 {
		return status
	}
	if services.IsCarrierIntelErrorKind(mapped, services.CarrierIntelErrorNotFound) {
		return http.StatusOK
	}
	return 0
}
