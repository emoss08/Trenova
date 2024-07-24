package audit

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/emoss08/trenova/config"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Service struct {
	db         *bun.DB
	logger     *config.ServerLogger
	workQueue  chan *models.AuditLog
	workerPool chan struct{}
	shutdown   chan struct{}
	wg         sync.WaitGroup
}

func NewAuditService(db *bun.DB, logger *config.ServerLogger, queueSize int, workerCount int) *Service {
	as := &Service{
		db:         db,
		logger:     logger,
		workQueue:  make(chan *models.AuditLog, queueSize),
		workerPool: make(chan struct{}, workerCount),
		shutdown:   make(chan struct{}),
	}

	for i := 0; i < workerCount; i++ {
		go as.worker()
	}

	return as
}

func (as *Service) worker() {
	for {
		select {
		case <-as.shutdown:
			return
		case log := <-as.workQueue:
			as.insertLog(log)
			as.logger.Debug().Interface("log", log).Msg("Audit log processed")
			as.wg.Done()
		}
	}
}

func (as *Service) insertLog(auditLog *models.AuditLog) {
	ctx := context.Background()
	_, err := as.db.NewInsert().Model(auditLog).Exec(ctx)
	if err != nil {
		as.logger.Error().Err(err).Msg("Failed to insert audit log")
	}
}

func (as *Service) LogAction(tableName, entityID string, action property.AuditLogAction, data any, userID, orgID, buID uuid.UUID) {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		as.logger.Error().Err(err).Msg("Failed to marshal data for audit log")
		return
	}

	auditLog := &models.AuditLog{
		TableName:      tableName,
		EntityID:       entityID,
		Action:         property.AuditLogAction(action.String()),
		Data:           dataJSON,
		UserID:         userID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Timestamp:      time.Now(),
	}

	as.wg.Add(1)
	select {
	case as.workQueue <- auditLog:
		as.logger.Debug().Msg("Log added to work queue")
		// Log added to work queue successfully
	default:
		as.logger.Debug().Msg("Work queue is full, logging synchronously")
		// Work queue is full, log synchronously as a fallback
		as.wg.Done()
		as.insertLog(auditLog)
	}
}

func (as *Service) Shutdown(ctx context.Context) error {
	close(as.shutdown)

	doneChan := make(chan struct{})
	go func() {
		as.wg.Wait()
		close(doneChan)
	}()

	select {
	case <-doneChan:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
