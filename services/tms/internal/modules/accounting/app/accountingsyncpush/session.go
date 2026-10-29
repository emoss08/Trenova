package accountingsyncpush

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/pkg/pagination"
)

type Session struct {
	Tenant            pagination.TenantInfo
	Conn              *accountingconnection.AccountingConnection
	Writer            accountingconnectorports.AccountingDocumentWriter
	Journals          accountingconnectorports.AccountingJournalWriter
	Auth              accountingconnectorports.AccountingDocumentAuth
	Loc               *time.Location
	ProviderName      string
	Profile           accountingprovider.ProviderProfile
	Limits            accountingconnectorports.AccountingDocumentLimits
	Control           *accountingcontrol.AccountingControl
	RedatesToNextOpen bool
}

type Result struct {
	Sync *accountingsync.SyncResult
	Refs map[string]string
}

func Finished(
	sess *Session,
	record *accountingsync.AccountingSyncRecord,
	written *accountingconnectorports.AccountingDocumentResult,
	document any,
	mappingIDs []string,
) (*Result, error) {
	payload, hash, err := PayloadOf(document)
	if err != nil {
		return nil, err
	}
	refs := map[string]string{}
	if written != nil {
		maps.Copy(refs, written.Refs)
	}
	result := &accountingsync.SyncResult{
		PayloadHash:  hash,
		Payload:      payload,
		MappingIDs:   mappingIDs,
		ExternalRefs: refs,
	}
	if written != nil {
		result.ExternalID = written.ExternalID
		result.ExternalDocNumber = written.DocNumber
		result.ExternalURL = sess.Writer.DocumentURL(
			sess.Auth,
			accountingconnectorports.AccountingDocumentLink{
				Kind:       record.ObjectType,
				ExternalID: written.ExternalID,
				DocNumber:  written.DocNumber,
			},
		)
	}
	return &Result{Sync: result, Refs: refs}, nil
}

func Partial(written *accountingconnectorports.AccountingDocumentResult) *Result {
	if written == nil || len(written.Refs) == 0 {
		return nil
	}
	return &Result{Refs: written.Refs}
}

func PayloadOf(document any) (payload map[string]any, hash string, err error) {
	if document == nil {
		return nil, "", nil
	}
	raw, err := sonic.Marshal(document)
	if err != nil {
		return nil, "", fmt.Errorf("accounting sync: encode payload: %w", err)
	}
	if err = sonic.Unmarshal(raw, &payload); err != nil {
		return nil, "", fmt.Errorf("accounting sync: decode payload: %w", err)
	}
	delete(payload, "Auth")
	delete(payload, "Refs")
	stored, err := sonic.ConfigStd.Marshal(payload)
	if err != nil {
		return nil, "", fmt.Errorf("accounting sync: encode payload: %w", err)
	}
	sum := sha256.Sum256(stored)
	return payload, hex.EncodeToString(sum[:]), nil
}

func WithProviderURL(result *Result, err error) (*Result, error) {
	if err != nil || result == nil || result.Sync == nil {
		return result, err
	}
	if url := result.Refs[accountingsync.ExternalRefURL]; url != "" {
		result.Sync.ExternalURL = url
	}
	return result, nil
}

func (sess *Session) CheckBooks(
	currency string,
	documentDate int64,
	label string,
) error {
	home := strings.ToUpper(strings.TrimSpace(sess.Conn.ExternalHomeCurrency))
	code := strings.ToUpper(strings.TrimSpace(currency))
	if home != "" && code != "" && code != home && !sess.Conn.ExternalMultiCurrencyEnabled {
		return Blocked(
			accountingsync.SyncErrorCurrency,
			label+" is in "+code+" but the books are kept in "+home,
			"Turn on multicurrency in "+sess.ProviderName+", or skip this record",
		)
	}
	if sess.Conn.BooksClosedOn(documentDate) {
		resolution := "Reopen the period in " + sess.ProviderName + ", or skip this record"
		if sess.RedatesToNextOpen {
			resolution = "Reopen the period in " + sess.ProviderName +
				", send it dated on the first open day, or skip this record"
		}
		return Blocked(
			accountingsync.SyncErrorClosedPeriod,
			label+" is dated in a period closed in "+sess.ProviderName,
			resolution,
		)
	}
	return nil
}
