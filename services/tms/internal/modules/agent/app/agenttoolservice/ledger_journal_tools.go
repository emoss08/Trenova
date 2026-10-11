package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/accounting/app/journalreversalservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/manualjournalservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/ledgertools"
)

var _ ledgertools.ManualJournalKeeper = (*manualjournalservice.Service)(nil)

var _ ledgertools.JournalReversalKeeper = (*journalreversalservice.Service)(nil)
