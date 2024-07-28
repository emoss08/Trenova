package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/emoss08/trenova/pkg/utils"

	"github.com/emoss08/trenova/config"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Service struct {
	db         *bun.DB
	logger     *config.ServerLogger
	workQueue  chan *Log
	workerPool chan struct{}
	shutdown   chan struct{}
	wg         sync.WaitGroup
}

func NewAuditService(db *bun.DB, logger *config.ServerLogger, queueSize int, workerCount int) *Service {
	as := &Service{
		db:         db,
		logger:     logger,
		workQueue:  make(chan *Log, queueSize),
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

func (as *Service) insertLog(auditLog *Log) {
	ctx := context.Background()
	_, err := as.db.NewInsert().Model(auditLog).Exec(ctx)
	if err != nil {
		as.logger.Error().Err(err).Msg("Failed to insert audit log")
	}
}

func (as *Service) LogAction(tableName, entityID string, action property.AuditLogAction, user AuditUser, orgID, buID uuid.UUID, opts ...LogOption) {
	auditLog := &Log{
		TableName:      tableName,
		EntityID:       entityID,
		Action:         property.AuditLogAction(action.String()),
		UserID:         user.ID,
		Username:       user.Username,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Timestamp:      time.Now(),
		Status:         property.LogStatusSucceeded,
		Description:    fmt.Sprintf("User: %s performed %s on %s", user.Username, action.String(), tableName),
	}

	for _, opt := range opts {
		opt(auditLog)
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

func (as *Service) LogAttempt(tableName, entityID string, action property.AuditLogAction, user AuditUser, orgID, buID uuid.UUID, opts ...LogOption) uuid.UUID {
	attemptID := uuid.New()

	auditLog := &Log{
		ID:             attemptID,
		TableName:      tableName,
		EntityID:       entityID,
		Action:         action,
		Username:       user.Username,
		UserID:         user.ID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Timestamp:      time.Now(),
		Status:         property.LogStatusAttempted,
		Description:    fmt.Sprintf("User: %s attempted %s on %s", user.Username, action.String(), tableName),
	}

	for _, opt := range opts {
		opt(auditLog)
	}

	as.wg.Add(1)
	select {
	case as.workQueue <- auditLog:
		as.logger.Debug().Msg("Attempt log added to work queue")
	default:
		as.logger.Debug().Msg("Work queue is full, logging attempt synchronously")
		as.wg.Done()
		as.insertLog(auditLog)
	}

	return attemptID
}

func (as *Service) LogError(action property.AuditLogAction, user AuditUser, attemptID, orgID, buID uuid.UUID, errorMsg string, opts ...LogOption) {
	auditLog := &Log{
		ID:             attemptID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Action:         action,
		Timestamp:      time.Now(),
		Status:         property.LogStatusFailed,
		ErrorMessage:   errorMsg,
		Description:    fmt.Sprintf("User: %s failed to perform action", user.Username),
		Username:       user.Username,
		UserID:         user.ID,
	}

	for _, opt := range opts {
		opt(auditLog)
	}

	as.wg.Add(1)
	select {
	case as.workQueue <- auditLog:
		as.logger.Debug().Msg("Error log added to work queue")
	default:
		as.logger.Debug().Msg("Work queue is full, logging error synchronously")
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

type LogOption func(log *Log)

func WithUser(user AuditUser) LogOption {
	return func(log *Log) {
		log.UserID = user.ID
		log.Username = user.Username
	}
}

func WithDiff(before, after any) LogOption {
	return func(log *Log) {
		diff, err := utils.JSONDiff(before, after)
		if err != nil {
			return
		}
		diffJSON, err := json.Marshal(diff)
		if err != nil {
			return
		}
		log.Changes = diffJSON
	}
}
