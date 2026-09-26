package aitrainingservice

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
)

var _ services.AITrainingScorer = (*Scorer)(nil)

const scoreReaderSize = 1 << 20

type Scorer struct {
	reader services.ExtractionReplyReader
}

func NewScorer(reader services.ExtractionReplyReader) *Scorer {
	return &Scorer{reader: reader}
}

func (s *Scorer) Score(
	ctx context.Context,
	req *services.ScoreTrainingPredictionsRequest,
) (*aitraining.ScoreReport, error) {
	if req == nil || req.Evaluation == nil || req.Predictions == nil {
		return nil, errortypes.NewValidationError(
			"predictions", errortypes.ErrRequired, "Both the evaluation set and the predictions are required",
		)
	}

	predictions, err := readPredictions(req.Predictions)
	if err != nil {
		return nil, err
	}

	report := &aitraining.ScoreReport{}
	model := aitraining.NewScoreTally()
	baseline := aitraining.NewScoreTally()
	err = eachJSONLine(req.Evaluation, func(line []byte) error {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		record := new(aitraining.EvaluationRecord)
		if decodeErr := sonic.Unmarshal(line, record); decodeErr != nil {
			return fmt.Errorf("decode evaluation record: %w", decodeErr)
		}
		if record.ID == "" || record.Expected == nil {
			return errortypes.NewBusinessError("An evaluation record is missing its id or expected values")
		}

		report.Examples++
		expected := record.Expected.Correction()
		model.Add(aicorrection.Score(s.modelPrediction(report, predictions, record.ID), expected))
		baseline.Add(aicorrection.Score(&aicorrection.Prediction{Snapshot: record.Baseline.Correction()}, expected))

		return nil
	})
	if err != nil {
		return nil, err
	}

	report.UnknownPredictions = len(predictions)
	report.Model = model.Side()
	report.Baseline = baseline.Side()
	report.AccuracyDelta = report.Model.Accuracy - report.Baseline.Accuracy

	return report, nil
}

func (s *Scorer) modelPrediction(
	report *aitraining.ScoreReport,
	predictions map[string]*aitraining.PredictionRecord,
	id string,
) *aicorrection.Prediction {
	empty := &aicorrection.Prediction{Snapshot: &aicorrection.Snapshot{Fields: map[string]string{}}}
	prediction, ok := predictions[id]
	if !ok {
		report.MissingPredictions++
		return empty
	}
	delete(predictions, id)
	if prediction.Error != "" {
		report.FailedPredictions++
		return empty
	}

	read, err := s.reader.ReadReply(prediction.Reply)
	if err != nil {
		report.UnreadablePredictions++
		return empty
	}
	report.Predicted++

	return read
}

func readPredictions(source io.Reader) (map[string]*aitraining.PredictionRecord, error) {
	predictions := map[string]*aitraining.PredictionRecord{}
	err := eachJSONLine(source, func(line []byte) error {
		record := new(aitraining.PredictionRecord)
		if err := sonic.Unmarshal(line, record); err != nil {
			return fmt.Errorf("decode prediction: %w", err)
		}
		if record.ID == "" {
			return errortypes.NewBusinessError("A prediction is missing its example id")
		}
		if _, duplicate := predictions[record.ID]; duplicate {
			return errortypes.NewBusinessError("Example {0} has more than one prediction", record.ID)
		}
		predictions[record.ID] = record

		return nil
	})
	if err != nil {
		return nil, err
	}

	return predictions, nil
}

func eachJSONLine(source io.Reader, handle func(line []byte) error) error {
	reader := bufio.NewReaderSize(source, scoreReaderSize)
	for {
		line, readErr := reader.ReadBytes('\n')
		if trimmed := bytes.TrimSpace(line); len(trimmed) > 0 {
			if err := handle(trimmed); err != nil {
				return err
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return fmt.Errorf("read lines: %w", readErr)
		}
	}
}
