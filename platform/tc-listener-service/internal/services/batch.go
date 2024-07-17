// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

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
