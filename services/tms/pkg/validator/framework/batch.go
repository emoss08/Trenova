package framework

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/sourcegraph/conc/pool"
)

// BatchValidator validates collections of items efficiently
type BatchValidator[T any] struct {
	itemValidator func(context.Context, T, int) *errortypes.MultiError
	config        *BatchConfig
	progress      *BatchProgress
}

type BatchConfig struct {
	ChunkSize        int
	MaxParallel      int
	FailFast         bool
	EnableProgress   bool
	CollectAllErrors bool
	MaxErrors        int
}

type BatchProgress struct {
	Total     int32
	Processed int32
	Failed    int32
	mu        sync.RWMutex
	callbacks []func(processed, total, failed int)
}

func DefaultBatchConfig() *BatchConfig {
	return &BatchConfig{
		ChunkSize:        100,
		MaxParallel:      10,
		FailFast:         false,
		EnableProgress:   false,
		CollectAllErrors: true,
		MaxErrors:        1000,
	}
}

func NewBatchValidator[T any](
	validator func(context.Context, T, int) *errortypes.MultiError,
) *BatchValidator[T] {
	return &BatchValidator[T]{
		itemValidator: validator,
		config:        DefaultBatchConfig(),
		progress:      &BatchProgress{},
	}
}

func (bv *BatchValidator[T]) WithConfig(config *BatchConfig) *BatchValidator[T] {
	bv.config = config
	if config.EnableProgress {
		bv.progress = &BatchProgress{
			callbacks: make([]func(int, int, int), 0),
		}
	}
	return bv
}

func (bv *BatchValidator[T]) OnProgress(
	callback func(processed, total, failed int),
) *BatchValidator[T] {
	if bv.progress != nil {
		bv.progress.mu.Lock()
		bv.progress.callbacks = append(bv.progress.callbacks, callback)
		bv.progress.mu.Unlock()
	}
	return bv
}

func (bv *BatchValidator[T]) Validate(ctx context.Context, items []T) *errortypes.MultiError {
	if len(items) == 0 {
		return nil
	}

	multiErr := errortypes.NewMultiError()

	if bv.config.EnableProgress && bv.progress != nil {
		atomic.StoreInt32(&bv.progress.Total, utils.SafeInt32(len(items)))
		atomic.StoreInt32(&bv.progress.Processed, 0)
		atomic.StoreInt32(&bv.progress.Failed, 0)
	}

	if bv.config.ChunkSize > 0 && len(items) > bv.config.ChunkSize {
		bv.validateInChunks(ctx, items, multiErr)
	} else {
		bv.validateItems(ctx, items, multiErr)
	}

	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (bv *BatchValidator[T]) validateInChunks( //nolint:gocognit // TODO: Refactor this
	ctx context.Context,
	items []T,
	multiErr *errortypes.MultiError,
) {
	chunks := bv.createChunks(items)
	var errMu sync.Mutex
	var errorCount int32

	p := pool.New()
	if bv.config.MaxParallel > 0 {
		p = p.WithMaxGoroutines(bv.config.MaxParallel)
	}

	for chunkIdx, chunk := range chunks {
		startIdx := chunkIdx * bv.config.ChunkSize

		p.Go(func() {
			if bv.config.FailFast && atomic.LoadInt32(&errorCount) > 0 {
				return
			}

			if bv.config.MaxErrors > 0 &&
				atomic.LoadInt32(&errorCount) >= utils.SafeInt32(bv.config.MaxErrors) {
				return
			}

			for i, item := range chunk {
				itemIdx := startIdx + i

				if bv.config.FailFast && atomic.LoadInt32(&errorCount) > 0 {
					break
				}

				if bv.config.MaxErrors > 0 &&
					atomic.LoadInt32(&errorCount) >= utils.SafeInt32(bv.config.MaxErrors) {
					break
				}

				itemErr := bv.itemValidator(ctx, item, itemIdx)

				if bv.config.EnableProgress && bv.progress != nil {
					atomic.AddInt32(&bv.progress.Processed, 1)
					if itemErr != nil && itemErr.HasErrors() {
						atomic.AddInt32(&bv.progress.Failed, 1)
					}
					bv.notifyProgress()
				}

				if itemErr != nil && itemErr.HasErrors() {
					errMu.Lock()
					for _, e := range itemErr.Errors {
						multiErr.AddError(e)
						atomic.AddInt32(&errorCount, 1)
					}
					errMu.Unlock()
				}
			}
		})
	}

	p.Wait()
}

func (bv *BatchValidator[T]) validateItems(
	ctx context.Context,
	items []T,
	multiErr *errortypes.MultiError,
) {
	var errMu sync.Mutex
	var errorCount int32

	p := pool.New()
	if bv.config.MaxParallel > 0 {
		p = p.WithMaxGoroutines(bv.config.MaxParallel)
	}

	for idx, item := range items {
		p.Go(func() {
			if bv.config.FailFast && atomic.LoadInt32(&errorCount) > 0 {
				return
			}

			if bv.config.MaxErrors > 0 &&
				atomic.LoadInt32(&errorCount) >= utils.SafeInt32(bv.config.MaxErrors) {
				return
			}

			itemErr := bv.itemValidator(ctx, item, idx)
			if bv.config.EnableProgress && bv.progress != nil {
				atomic.AddInt32(&bv.progress.Processed, 1)
				if itemErr != nil && itemErr.HasErrors() {
					atomic.AddInt32(&bv.progress.Failed, 1)
				}
				bv.notifyProgress()
			}

			if itemErr != nil && itemErr.HasErrors() {
				errMu.Lock()
				for _, e := range itemErr.Errors {
					multiErr.AddError(e)
					atomic.AddInt32(&errorCount, 1)
				}
				errMu.Unlock()
			}
		})
	}

	p.Wait()
}

func (bv *BatchValidator[T]) createChunks(items []T) [][]T {
	var chunks [][]T
	chunkSize := bv.config.ChunkSize

	for i := 0; i < len(items); i += chunkSize {
		end := i + chunkSize
		if end > len(items) {
			end = len(items)
		}
		chunks = append(chunks, items[i:end])
	}

	return chunks
}

func (bv *BatchValidator[T]) notifyProgress() {
	if bv.progress == nil {
		return
	}

	processed := atomic.LoadInt32(&bv.progress.Processed)
	total := atomic.LoadInt32(&bv.progress.Total)
	failed := atomic.LoadInt32(&bv.progress.Failed)

	bv.progress.mu.RLock()
	callbacks := bv.progress.callbacks
	bv.progress.mu.RUnlock()

	for _, callback := range callbacks {
		callback(int(processed), int(total), int(failed))
	}
}

func (bv *BatchValidator[T]) GetProgress() (processed, total, failed int) {
	if bv.progress == nil {
		return 0, 0, 0
	}

	return int(atomic.LoadInt32(&bv.progress.Processed)),
		int(atomic.LoadInt32(&bv.progress.Total)),
		int(atomic.LoadInt32(&bv.progress.Failed))
}

type CollectionValidator[T any] struct {
	*BatchValidator[T]
	preloadFunc  func(context.Context, []T) error
	postloadFunc func(context.Context, []T, *errortypes.MultiError) error
}

func NewCollectionValidator[T any](
	validator func(context.Context, T, int) *errortypes.MultiError,
) *CollectionValidator[T] {
	return &CollectionValidator[T]{
		BatchValidator: NewBatchValidator(validator),
	}
}

func (cv *CollectionValidator[T]) WithPreload(
	fn func(context.Context, []T) error,
) *CollectionValidator[T] {
	cv.preloadFunc = fn
	return cv
}

func (cv *CollectionValidator[T]) WithPostload(
	fn func(context.Context, []T, *errortypes.MultiError) error,
) *CollectionValidator[T] {
	cv.postloadFunc = fn
	return cv
}

func (cv *CollectionValidator[T]) Validate(ctx context.Context, items []T) *errortypes.MultiError {
	if len(items) == 0 {
		return nil
	}

	multiErr := errortypes.NewMultiError()
	if cv.preloadFunc != nil {
		if err := cv.preloadFunc(ctx, items); err != nil {
			multiErr.Add("", errortypes.ErrSystemError,
				fmt.Sprintf("Preload failed: %v", err))
			return multiErr
		}
	}

	if err := cv.BatchValidator.Validate(ctx, items); err != nil {
		for _, e := range err.Errors {
			multiErr.AddError(e)
		}
	}

	if cv.postloadFunc != nil {
		if err := cv.postloadFunc(ctx, items, multiErr); err != nil {
			multiErr.Add("", errortypes.ErrSystemError,
				fmt.Sprintf("Postload failed: %v", err))
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

type IndexedBatchValidator[T any] struct {
	*BatchValidator[T]
	errorsByIndex map[int]*errortypes.MultiError
	mu            sync.RWMutex
}

func NewIndexedBatchValidator[T any](
	validator func(context.Context, T, int) *errortypes.MultiError,
) *IndexedBatchValidator[T] {
	return &IndexedBatchValidator[T]{
		BatchValidator: NewBatchValidator(validator),
		errorsByIndex:  make(map[int]*errortypes.MultiError),
	}
}

func (ibv *IndexedBatchValidator[T]) Validate(
	ctx context.Context,
	items []T,
	fieldName string,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	originalValidator := ibv.itemValidator
	ibv.itemValidator = func(ctx context.Context, item T, idx int) *errortypes.MultiError {
		itemErr := originalValidator(ctx, item, idx)
		if itemErr != nil && itemErr.HasErrors() {
			ibv.mu.Lock()
			ibv.errorsByIndex[idx] = itemErr
			ibv.mu.Unlock()
		}
		return itemErr
	}

	if err := ibv.BatchValidator.Validate(ctx, items); err != nil {
		ibv.mu.RLock()
		for idx, indexErr := range ibv.errorsByIndex {
			indexedErr := multiErr.WithIndex(fieldName, idx)
			for _, e := range indexErr.Errors {
				indexedErr.AddError(e)
			}
		}
		ibv.mu.RUnlock()
	}

	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (ibv *IndexedBatchValidator[T]) GetErrorsByIndex() map[int]*errortypes.MultiError {
	ibv.mu.RLock()
	defer ibv.mu.RUnlock()

	result := make(map[int]*errortypes.MultiError)
	for k, v := range ibv.errorsByIndex {
		result[k] = v
	}
	return result
}

func (ibv *IndexedBatchValidator[T]) ClearErrors() {
	ibv.mu.Lock()
	defer ibv.mu.Unlock()
	ibv.errorsByIndex = make(map[int]*errortypes.MultiError)
}
