package openaiadapter

import (
	"fmt"
	"math"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func checkVectorCount(call *modeladapter.EmbedCall, got int) error {
	if got == len(call.Inputs) {
		return nil
	}

	return errortypes.NewBusinessError(
		"Embedding provider {0} returned {1} vectors for {2} texts",
		call.Provider.Name,
		got,
		len(call.Inputs),
	).WithInternal(aiusageports.ErrEmbeddingResponseInvalid)
}

func checkEmbeddings(call *modeladapter.EmbedCall, vectors [][]float32) error {
	if err := checkVectorCount(call, len(vectors)); err != nil {
		return err
	}

	want := call.Dimensions()
	for idx, vector := range vectors {
		if len(vector) != want {
			return errortypes.NewBusinessError(
				"Embedding provider {0} returned {1} dimensions, but it is configured for {2}. Set the dimension the model actually returns, or choose a model that supports {2}",
				call.Provider.Name,
				len(vector),
				want,
			).WithInternal(aiusageports.ErrEmbeddingDimensionMismatch)
		}
		if err := checkFinite(vector); err != nil {
			return errortypes.NewBusinessError(
				"Embedding provider {0} returned an unusable response",
				call.Provider.Name,
			).WithInternal(fmt.Errorf("vector %d: %w: %w", idx, err, aiusageports.ErrEmbeddingResponseInvalid))
		}
	}

	return nil
}

func checkFinite(vector []float32) error {
	for pos, value := range vector {
		f := float64(value)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return fmt.Errorf("component %d is not a finite number", pos)
		}
	}

	return nil
}
