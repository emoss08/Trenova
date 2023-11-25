package tasks

import (
	"backend/service"
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hibiken/asynq"
)

// A list of task types.
const (
	TypeGenerateThumbnail = "image:generate_thumbnail"
)

type GenerateThumbnailPayload struct {
	InputPath  string `json:"inputPath"`
	OutputPath string `json:"outputPath"`
	Width      uint   `json:"width"`
	Height     uint   `json:"height"`
}

func NewGenerateThumbnailTask(inputPath, outputPath string, width, height uint) (*asynq.Task, error) {
	payload, err := json.Marshal(GenerateThumbnailPayload{
		InputPath:  inputPath,
		OutputPath: outputPath,
		Width:      width,
		Height:     height,
	})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeGenerateThumbnail, payload), nil
}

// ThumbnailProcessor implements asynq.Handler interface.
type ThumbnailProcessor struct {
}

// ProcessTask implements asynq.Handler interface.
func (tp *ThumbnailProcessor) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload GenerateThumbnailPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	log.Printf("Processing task: %v", payload)
	service.GenerateThumbnail(payload.InputPath, payload.OutputPath, payload.Width, payload.Height)

	return nil
}

func NewThumbnailProcessor() *ThumbnailProcessor {
	return &ThumbnailProcessor{}
}
