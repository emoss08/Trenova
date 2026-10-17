package agentwaitservice

import (
	"cmp"
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentwait"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentwaitjobs/agentwaitcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentwaitports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	// documentReadRecheckSeconds is how often a file being read is looked at
	// again. The extraction ends the wait the moment it is saved; the timer
	// is for the readings that save none, such as a file that could not be
	// read, and it sits past rearmSlack so a check that finds the file still
	// being read is put off rather than taken as due.
	documentReadRecheckSeconds int64 = 60
	// documentReadLifetimeSeconds gives up on a reading that has not
	// finished in half an hour; the work is picked up told so.
	documentReadLifetimeSeconds int64 = 30 * 60
	documentReadSuffix                = " to finish being read"
	documentReadThen                  = "Read the file with read_attachment and carry on with " +
		"what the person asked of it, without asking them to send their message again."
	// maxWaitedFileNameRunes keeps the description within its limit however
	// long the file's name is.
	maxWaitedFileNameRunes = agentwait.MaxDescriptionRunes - len(documentReadSuffix)
	// maxReadingErrorRunes keeps why a file could not be read to a line.
	maxReadingErrorRunes = 300
)

func NewDocumentReadWaiter(s *Service) agentwaitports.DocumentReadWaiter { return s }

// WaitForDocumentRead parks the conversation on a file still being read. One
// wait serves a file in a conversation, so finding the file still being read
// again keeps the wait already open.
func (s *Service) WaitForDocumentRead(
	ctx context.Context,
	req *agentwaitports.WaitForDocumentReadRequest,
) (*agentwait.Wait, error) {
	if req.Actor == nil || req.ThreadID.IsNil() || req.DocumentID.IsNil() {
		return nil, errortypes.NewBusinessError(
			"A file is waited on by the conversation it was attached to",
		)
	}

	open, err := s.openDocumentReads(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(open) > 0 {
		return open[0], nil
	}

	name := clip(cmp.Or(req.FileName, "the attached file"), maxWaitedFileNameRunes)

	return s.Register(ctx, &agentwaitports.RegisterWaitRequest{
		Actor:           req.Actor,
		DefinitionID:    req.DefinitionID,
		ThreadID:        req.ThreadID,
		Kind:            agentwait.KindDocumentRead,
		Condition:       agentwait.Condition{DocumentID: req.DocumentID},
		Description:     name + documentReadSuffix,
		Then:            documentReadThen,
		LifetimeSeconds: documentReadLifetimeSeconds,
	})
}

func (s *Service) StopWaitingForDocumentRead(
	ctx context.Context,
	req *agentwaitports.WaitForDocumentReadRequest,
) error {
	if req.Actor == nil || req.ThreadID.IsNil() || req.DocumentID.IsNil() {
		return nil
	}
	open, err := s.openDocumentReads(ctx, req)
	if err != nil {
		return err
	}
	for _, wait := range open {
		if _, err = s.Cancel(ctx, &agentwaitports.CancelWaitRequest{
			ID:         wait.ID,
			TenantInfo: tenantOf(wait),
			ThreadID:   wait.ThreadID,
			UserID:     wait.UserID,
			By:         "the agent, which read the file in full",
		}); err != nil {
			return err
		}
	}

	return nil
}

// openDocumentReads is the conversation's open waits on the file.
func (s *Service) openDocumentReads(
	ctx context.Context,
	req *agentwaitports.WaitForDocumentReadRequest,
) ([]*agentwait.Wait, error) {
	open, err := s.waits.ListOpenWatching(ctx, &agentwaitports.ListOpenWaitsWatchingRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: req.Actor.OrganizationID,
			BuID:  req.Actor.BusinessUnitID,
		},
		Kinds:    []agentwait.Kind{agentwait.KindDocumentRead},
		WatchIDs: []pulid.ID{req.DocumentID},
	})
	if err != nil {
		return nil, err
	}

	return slices.DeleteFunc(open, func(wait *agentwait.Wait) bool {
		return wait.ThreadID != req.ThreadID
	}), nil
}

// fileReading is a file and what has been read off it, nil while nothing
// has.
type fileReading struct {
	doc     *document.Document
	content *documentcontent.Content
}

// stillBeingRead is read_attachment's own test: the text is not in, or a
// model's reading of it is under way.
func (r *fileReading) stillBeingRead(now int64) bool {
	if r.content != nil {
		return r.content.StillBeingRead(now)
	}

	return r.doc.ContentStatus == document.ContentStatusPending ||
		r.doc.ContentStatus == document.ContentStatusExtracting
}

// outcome says what reading the file came to.
func (r *fileReading) outcome() string {
	name := cmp.Or(r.doc.OriginalName, r.doc.FileName, "The file")
	why := r.doc.ContentError
	if r.content != nil {
		why = cmp.Or(r.content.FailureMessage, why)
	}
	if why == "" && r.doc.ContentStatus != document.ContentStatusFailed {
		return name + " has finished being read."
	}
	if why == "" {
		return name + " could not be read."
	}

	return name + " could not be read: " + clip(why, maxReadingErrorRunes)
}

func (f *facts) reading(
	ctx context.Context,
	tenant pagination.TenantInfo,
	id pulid.ID,
) (*fileReading, error) {
	if f.documents == nil {
		return nil, errortypes.NewBusinessError("Files cannot be waited on here")
	}
	doc, err := f.documents.GetByID(ctx, documentports.GetDocumentByIDRequest{
		ID:         id,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, missing(err, "file")
	}
	reading := &fileReading{doc: doc}
	if f.contents == nil {
		return reading, nil
	}
	content, err := f.contents.GetByDocumentID(ctx, id, tenant)
	switch {
	case err == nil:
		reading.content = content
	case !errortypes.IsNotFoundError(err):
		return nil, err
	}

	return reading, nil
}

func (f *facts) assessDocumentRead(
	ctx context.Context,
	tenant pagination.TenantInfo,
	c *agentwait.Condition,
	now int64,
) (*int64, error) {
	reading, err := f.reading(ctx, tenant, c.DocumentID)
	if err != nil {
		return nil, err
	}
	if !reading.stillBeingRead(now) {
		return nil, alreadyTrue("%s", reading.outcome())
	}
	due := now + documentReadRecheckSeconds

	return &due, nil
}

func (f *facts) recheckDocumentRead(
	ctx context.Context,
	tenant pagination.TenantInfo,
	c *agentwait.Condition,
	now int64,
) (*agentwaitcontract.Check, error) {
	reading, err := f.reading(ctx, tenant, c.DocumentID)
	if err != nil {
		return goneOr(err, "The file was removed before it was read.")
	}
	if reading.stillBeingRead(now) {
		due := now + documentReadRecheckSeconds
		return &agentwaitcontract.Check{DueAt: &due}, nil
	}

	return met("%s", reading.outcome()), nil
}
