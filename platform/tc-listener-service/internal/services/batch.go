package services

import (
	"sync"
	"time"

	"github.com/jordan-wright/email"
	"github.com/rs/zerolog"
)

// BatchEmailer batches emails and sends them periodically.
type BatchEmailer struct {
	sync.Mutex
	batch        []*email.Email
	flushPeriod  time.Duration
	emailService *EmailService
	stopChan     chan struct{}
	logger       *zerolog.Logger
}

// NewBatchEmailer creates a new BatchEmailer.
func NewBatchEmailer(emailService *EmailService, logger *zerolog.Logger, flushPeriod time.Duration) *BatchEmailer {
	be := &BatchEmailer{
		flushPeriod:  flushPeriod,
		emailService: emailService,
		stopChan:     make(chan struct{}),
		logger:       logger,
	}
	go be.start()
	return be
}

// AddEmail adds an email to the batch.
func (be *BatchEmailer) AddEmail(e *email.Email) {
	be.Lock()
	defer be.Unlock()
	be.logger.Debug().Msgf("Adding email with subject: %v to %v to batch", e.Subject, e.To)
	be.batch = append(be.batch, e)
}

// start starts the periodic flush.
func (be *BatchEmailer) start() {
	ticker := time.NewTicker(be.flushPeriod)
	defer ticker.Stop()
	be.logger.Debug().Msgf("Starting batch emailer with flush period: %v", be.flushPeriod)
	for {
		select {
		case <-ticker.C:
			be.flush()
		case <-be.stopChan:
			be.flush()
			return
		}
	}
}

// flush sends all batched emails.
func (be *BatchEmailer) flush() {
	be.Lock()
	defer be.Unlock()

	for _, e := range be.batch {
		be.logger.Debug().Msgf("Sending email with subject: %v to %v", e.Subject, e.To)
		be.emailService.SendBulk(e.To, e.Subject, string(e.HTML))
	}
	be.batch = []*email.Email{}
}

// Stop stops the periodic flush.
func (be *BatchEmailer) Stop() {
	close(be.stopChan)
}



// TODO(WOLFRED): Insert the emails sent out into the database by organization_id.