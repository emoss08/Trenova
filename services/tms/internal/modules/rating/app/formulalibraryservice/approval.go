package formulalibraryservice

import (
	"context"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulalibraryports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/approvalworkflow"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type ApprovalActionRequest = approvalworkflow.Request

type libraryTransition = approvalworkflow.Transition[*formulalibrary.FormulaLibrary, formulatemplate.Status]

type SubmitRequest struct {
	ApprovalActionRequest
	ReviewerID   pulid.ID
	ReviewerNote string
}

func (s *Service) approvals(
	ctx context.Context,
) approvalworkflow.Engine[*formulalibrary.FormulaLibrary, formulatemplate.Status] {
	return approvalworkflow.Engine[*formulalibrary.FormulaLibrary, formulatemplate.Status]{
		Label: "library",
		Load: func(
			ctx context.Context,
			id pulid.ID,
			tenant pagination.TenantInfo,
		) (*formulalibrary.FormulaLibrary, error) {
			return s.repo.GetByID(ctx, &formulalibraryports.GetLibraryByIDRequest{
				TenantInfo: tenant,
				LibraryID:  id,
			})
		},
		Save:     s.repo.Update,
		StatusOf: func(l *formulalibrary.FormulaLibrary) formulatemplate.Status { return l.Status },
		SetStatus: func(l *formulalibrary.FormulaLibrary, status formulatemplate.Status) {
			l.Status = status
		},
		CanTransition: formulatemplate.CanTransition,
		Snapshot: func(l *formulalibrary.FormulaLibrary) *formulalibrary.FormulaLibrary {
			snapshot := *l
			return &snapshot
		},
		Audit: func(
			updated, original *formulalibrary.FormulaLibrary,
			operation permission.Operation,
			req *ApprovalActionRequest,
			comment string,
		) {
			opts := make([]auditports.LogOption, 0, 1)
			if operation == permission.OpApprove || operation == permission.OpReject {
				opts = append(opts, auditservice.WithCritical())
			}
			s.logAuditAction(
				ctx,
				updated,
				operation,
				req.TenantInfo.UserID,
				original,
				comment,
				opts...)
		},
	}
}

func (s *Service) Submit(
	ctx context.Context,
	req *SubmitRequest,
) (*formulalibrary.FormulaLibrary, error) {
	var submitted *formulalibrary.FormulaLibrary
	err := s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		result, txErr := s.approvals(ctx).
			Apply(txCtx, &req.ApprovalActionRequest, libraryTransition{
				Operation:    "Submit",
				From:         formulatemplate.StatusDraft,
				AlsoFrom:     []formulatemplate.Status{formulatemplate.StatusInactive},
				To:           formulatemplate.StatusInReview,
				PermissionOp: permission.OpSubmit,
				AuditComment: titleSubmitted,
				Validate: func(
					vCtx context.Context,
					library *formulalibrary.FormulaLibrary,
					_ *ApprovalActionRequest,
				) error {
					return s.validateSubmission(vCtx, library, req)
				},
				Apply: func(library *formulalibrary.FormulaLibrary, r *ApprovalActionRequest, now int64) {
					userID := r.TenantInfo.UserID
					library.SubmittedByID = &userID
					library.SubmittedAt = &now
					library.ReviewComment = r.Comment
				},
				AfterSave: func(
					aCtx context.Context,
					updated *formulalibrary.FormulaLibrary,
					r *ApprovalActionRequest,
				) error {
					return s.recordReview(aCtx, updated, r.TenantInfo, &reviewRecord{
						Comment:             r.Comment,
						Decision:            formulatemplate.ReviewDecisionSubmitted,
						RequestedReviewerID: req.ReviewerID,
						ReviewerNote:        req.ReviewerNote,
					})
				},
			})
		submitted = result
		return txErr
	})
	if err != nil {
		return nil, err
	}

	s.notifySubmitted(ctx, req.TenantInfo, submitted)
	if req.ReviewerID.IsNotNil() {
		s.notifyRequestedReviewer(ctx, req.TenantInfo, submitted, req.ReviewerID)
	}
	return submitted, nil
}

func (s *Service) validateSubmission(
	ctx context.Context,
	library *formulalibrary.FormulaLibrary,
	req *SubmitRequest,
) error {
	multiErr := errortypes.NewMultiError()
	if utf8.RuneCountInString(strings.TrimSpace(req.ReviewerNote)) >
		formulatemplate.MaxReviewerNoteLength {
		multiErr.Add(
			"reviewerNote",
			errortypes.ErrInvalid,
			"A note for the reviewer cannot be longer than 4,000 characters",
		)
	}
	if req.ReviewerID.IsNotNil() {
		eligible, err := s.isEligibleReviewer(ctx, req.TenantInfo, req.ReviewerID)
		if err != nil {
			return err
		}
		if !eligible {
			multiErr.Add(
				"reviewerId",
				errortypes.ErrInvalid,
				"Choose an active reviewer, other than you, who can approve formula libraries",
			)
		}
	}

	gate, err := s.gate(ctx, library)
	if err != nil {
		return err
	}
	gate.addRefusals(multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (s *Service) Approve(
	ctx context.Context,
	req *ApprovalActionRequest,
) (*formulalibrary.FormulaLibrary, error) {
	var approved *formulalibrary.FormulaLibrary
	var content *contentCheck
	err := s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		result, txErr := s.approvals(ctx).Apply(txCtx, req, libraryTransition{
			Operation:    "Approve",
			From:         formulatemplate.StatusInReview,
			To:           formulatemplate.StatusActive,
			PermissionOp: permission.OpApprove,
			AuditComment: "Formula library approved",
			Validate: func(
				vCtx context.Context,
				library *formulalibrary.FormulaLibrary,
				r *ApprovalActionRequest,
			) error {
				if library.SubmittedByID != nil && *library.SubmittedByID == r.TenantInfo.UserID {
					return errortypes.NewValidationError(
						"approvedById",
						errortypes.ErrInvalid,
						"A library cannot be approved by its submitter",
					)
				}
				if formulatemplate.SubmissionIsStale(library.SubmittedAt, timeutils.NowUnix()) {
					return staleSubmissionError()
				}
				if err := s.refuseRevisionAuthor(vCtx, library, r.TenantInfo); err != nil {
					return err
				}

				gate, err := s.gate(vCtx, library)
				if err != nil {
					return err
				}
				multiErr := errortypes.NewMultiError()
				gate.addRefusals(multiErr)
				if multiErr.HasErrors() {
					return multiErr
				}
				content = gate.content
				return nil
			},
			Apply: func(library *formulalibrary.FormulaLibrary, r *ApprovalActionRequest, now int64) {
				userID := r.TenantInfo.UserID
				library.ApprovedByID = &userID
				library.ApprovedAt = &now
				library.ReviewComment = r.Comment
				applyContent(library, content)
				library.CurrentVersionNumber++
			},
			AfterSave: func(
				aCtx context.Context,
				updated *formulalibrary.FormulaLibrary,
				r *ApprovalActionRequest,
			) error {
				version := formulalibrary.NewVersionFromLibrary(
					updated,
					updated.CurrentVersionNumber,
					r.TenantInfo.UserID,
					"Approved",
					nil,
				)
				version.Status = formulatemplate.StatusActive
				if _, err := s.versionRepo.Create(aCtx, version); err != nil {
					return err
				}
				return s.recordReview(aCtx, updated, r.TenantInfo, &reviewRecord{
					Comment:  r.Comment,
					Decision: formulatemplate.ReviewDecisionApproved,
				})
			},
		})
		approved = result
		return txErr
	})
	if err != nil {
		return nil, err
	}

	s.notifyReviewOutcome(ctx, req.TenantInfo, &reviewOutcome{
		Library:     approved,
		SubmitterID: approved.SubmittedByID,
		Decision:    formulatemplate.ReviewDecisionApproved,
		Comment:     req.Comment,
	})
	return approved, nil
}

func (s *Service) refuseRevisionAuthor(
	ctx context.Context,
	library *formulalibrary.FormulaLibrary,
	tenantInfo pagination.TenantInfo,
) error {
	var baseVersion int64
	latest, err := s.reviewRepo.Latest(ctx, library.ID, tenantInfo)
	if err != nil {
		return err
	}
	if latest != nil {
		baseVersion = latest.BaseVersionNumber
	}

	authors, err := s.versionRepo.ListAuthorIDsSince(
		ctx,
		&formulalibraryports.ListLibraryAuthorsRequest{
			TenantInfo:   tenantInfo,
			LibraryID:    library.ID,
			AfterVersion: baseVersion,
		},
	)
	if err != nil {
		return err
	}
	if slices.Contains(authors, tenantInfo.UserID) {
		return errortypes.NewValidationError(
			"approvedById",
			errortypes.ErrInvalid,
			"You authored part of this revision, so another reviewer must approve it",
		)
	}
	return nil
}

func (s *Service) Reject(
	ctx context.Context,
	req *ApprovalActionRequest,
) (*formulalibrary.FormulaLibrary, error) {
	if err := approvalworkflow.RequireComment(req, "rejecting a formula library"); err != nil {
		return nil, err
	}

	approvedVersion := s.approvedVersionNumber(ctx, req.EntityID, req.TenantInfo)
	destination := formulatemplate.StatusInactive
	if approvedVersion > 0 {
		destination = formulatemplate.StatusDraft
	}

	var submitterID *pulid.ID
	var rejected *formulalibrary.FormulaLibrary
	err := s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		result, txErr := s.approvals(ctx).Apply(txCtx, req, libraryTransition{
			Operation:    "Reject",
			From:         formulatemplate.StatusInReview,
			To:           destination,
			PermissionOp: permission.OpReject,
			AuditComment: "Formula library rejected",
			Apply: func(library *formulalibrary.FormulaLibrary, r *ApprovalActionRequest, _ int64) {
				if library.SubmittedByID != nil {
					captured := *library.SubmittedByID
					submitterID = &captured
				}
				library.SubmittedByID = nil
				library.SubmittedAt = nil
				library.ReviewComment = r.Comment
			},
			AfterSave: func(
				aCtx context.Context,
				updated *formulalibrary.FormulaLibrary,
				r *ApprovalActionRequest,
			) error {
				return s.recordReview(aCtx, updated, r.TenantInfo, &reviewRecord{
					Comment:  r.Comment,
					Decision: formulatemplate.ReviewDecisionRejected,
				})
			},
		})
		rejected = result
		return txErr
	})
	if err != nil {
		return nil, err
	}

	s.notifyReviewOutcome(ctx, req.TenantInfo, &reviewOutcome{
		Library:         rejected,
		SubmitterID:     submitterID,
		Decision:        formulatemplate.ReviewDecisionRejected,
		Comment:         req.Comment,
		ApprovedVersion: approvedVersion,
	})
	return rejected, nil
}

func (s *Service) RequestChanges(
	ctx context.Context,
	req *ApprovalActionRequest,
) (*formulalibrary.FormulaLibrary, error) {
	if err := approvalworkflow.RequireComment(req, "requesting changes"); err != nil {
		return nil, err
	}

	var submitterID *pulid.ID
	var returned *formulalibrary.FormulaLibrary
	err := s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		result, txErr := s.approvals(ctx).Apply(txCtx, req, libraryTransition{
			Operation:    "RequestChanges",
			From:         formulatemplate.StatusInReview,
			To:           formulatemplate.StatusDraft,
			PermissionOp: permission.OpReject,
			AuditComment: "Changes requested on formula library",
			Validate: func(
				_ context.Context,
				library *formulalibrary.FormulaLibrary,
				r *ApprovalActionRequest,
			) error {
				if library.SubmittedByID != nil && *library.SubmittedByID == r.TenantInfo.UserID {
					return errortypes.NewValidationError(
						"approvedById",
						errortypes.ErrInvalid,
						"A library's submitter cannot review it; ask another reviewer",
					)
				}
				return nil
			},
			Apply: func(library *formulalibrary.FormulaLibrary, r *ApprovalActionRequest, _ int64) {
				if library.SubmittedByID != nil {
					captured := *library.SubmittedByID
					submitterID = &captured
				}
				library.ReviewComment = r.Comment
			},
			AfterSave: func(
				aCtx context.Context,
				updated *formulalibrary.FormulaLibrary,
				r *ApprovalActionRequest,
			) error {
				return s.recordReview(aCtx, updated, r.TenantInfo, &reviewRecord{
					Comment:  r.Comment,
					Decision: formulatemplate.ReviewDecisionChangesRequested,
				})
			},
		})
		returned = result
		return txErr
	})
	if err != nil {
		return nil, err
	}

	s.notifyReviewOutcome(ctx, req.TenantInfo, &reviewOutcome{
		Library:     returned,
		SubmitterID: submitterID,
		Decision:    formulatemplate.ReviewDecisionChangesRequested,
		Comment:     req.Comment,
	})
	return returned, nil
}

type ListReviewsRequest struct {
	TenantInfo pagination.TenantInfo
	LibraryID  pulid.ID
}

func (s *Service) ListReviews(
	ctx context.Context,
	req *ListReviewsRequest,
) ([]*formulalibrary.FormulaLibraryReview, error) {
	if _, err := s.repo.GetByID(ctx, &formulalibraryports.GetLibraryByIDRequest{
		TenantInfo: req.TenantInfo,
		LibraryID:  req.LibraryID,
	}); err != nil {
		return nil, err
	}

	reviews, err := s.reviewRepo.ListByLibrary(ctx, &formulalibraryports.ListLibraryReviewsRequest{
		TenantInfo: req.TenantInfo,
		LibraryID:  req.LibraryID,
	})
	if err != nil {
		return nil, err
	}
	if reviews == nil {
		reviews = []*formulalibrary.FormulaLibraryReview{}
	}
	return reviews, nil
}

type ExpireStaleSubmissionsRequest struct {
	Now   int64
	Limit int
}

type ExpireStaleSubmissionsResult struct {
	Expired []pulid.ID `json:"expired"`
}

func (s *Service) ExpireStaleSubmissions(
	ctx context.Context,
	req *ExpireStaleSubmissionsRequest,
) (*ExpireStaleSubmissionsResult, error) {
	now := req.Now
	if now == 0 {
		now = timeutils.NowUnix()
	}

	stale, err := s.reviewRepo.ListStaleSubmissions(
		ctx,
		&formulalibraryports.ListStaleLibrarySubmissionsRequest{
			SubmittedBefore: now - formulatemplate.SubmissionExpiry,
			Limit:           req.Limit,
		},
	)
	if err != nil {
		return nil, err
	}

	result := &ExpireStaleSubmissionsResult{Expired: make([]pulid.ID, 0, len(stale))}
	for _, library := range stale {
		if library == nil || !formulatemplate.SubmissionIsStale(library.SubmittedAt, now) {
			continue
		}
		tenantInfo := libraryTenant(library)
		tenantCtx := dbscope.WithTenant(ctx, tenantInfo.DBTenant())
		if expireErr := s.expireSubmission(tenantCtx, library, tenantInfo); expireErr != nil {
			s.l.Error("failed to expire stale library submission",
				zap.String("libraryID", library.ID.String()),
				zap.Error(expireErr),
			)
			continue
		}
		result.Expired = append(result.Expired, library.ID)
	}
	return result, nil
}

func (s *Service) expireSubmission(
	ctx context.Context,
	library *formulalibrary.FormulaLibrary,
	tenantInfo pagination.TenantInfo,
) error {
	var submitterID *pulid.ID
	if library.SubmittedByID != nil {
		captured := *library.SubmittedByID
		submitterID = &captured
	}

	original := *library
	library.Status = formulatemplate.StatusDraft
	library.SubmittedByID = nil
	library.SubmittedAt = nil
	library.ReviewComment = "Submission expired after waiting for review"

	var updated *formulalibrary.FormulaLibrary
	err := s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		saved, txErr := s.repo.Update(txCtx, library)
		if txErr != nil {
			return txErr
		}
		if txErr = s.recordReview(txCtx, saved, tenantInfo, &reviewRecord{
			Comment:  library.ReviewComment,
			Decision: formulatemplate.ReviewDecisionExpired,
		}); txErr != nil {
			return txErr
		}
		updated = saved
		return nil
	})
	if err != nil {
		return err
	}

	s.logAuditAction(ctx, updated, permission.OpUpdate, pulid.Nil, &original,
		"Formula library submission expired")
	s.notifyReviewOutcome(ctx, tenantInfo, &reviewOutcome{
		Library:     updated,
		SubmitterID: submitterID,
		Decision:    formulatemplate.ReviewDecisionExpired,
		Comment:     library.ReviewComment,
	})
	return nil
}

type reviewRecord struct {
	Comment             string
	Decision            formulatemplate.ReviewDecision
	RequestedReviewerID pulid.ID
	ReviewerNote        string
}

func (s *Service) recordReview(
	ctx context.Context,
	library *formulalibrary.FormulaLibrary,
	tenantInfo pagination.TenantInfo,
	record *reviewRecord,
) error {
	latest, err := s.reviewRepo.Latest(ctx, library.ID, tenantInfo)
	if err != nil {
		return err
	}

	round := int32(1)
	if latest != nil {
		round = latest.Round
	}
	if record.Decision == formulatemplate.ReviewDecisionSubmitted {
		round = nextRound(latest)
	}

	review := &formulalibrary.FormulaLibraryReview{
		LibraryID:      library.ID,
		OrganizationID: library.OrganizationID,
		BusinessUnitID: library.BusinessUnitID,
		Round:          round,
		Decision:       record.Decision,
		Comment:        record.Comment,
		ReviewerNote:   strings.TrimSpace(record.ReviewerNote),
	}
	if record.RequestedReviewerID.IsNotNil() {
		reviewerID := record.RequestedReviewerID
		review.RequestedReviewerID = &reviewerID
	}
	if record.Decision != formulatemplate.ReviewDecisionExpired && tenantInfo.UserID.IsNotNil() {
		actor := tenantInfo.UserID
		review.ActorID = &actor
	}
	switch {
	case record.Decision == formulatemplate.ReviewDecisionSubmitted:
		review.BaseVersionNumber = s.approvedVersionNumber(ctx, library.ID, tenantInfo)
	case latest != nil:
		review.BaseVersionNumber = latest.BaseVersionNumber
	}

	multiErr := errortypes.NewMultiError()
	review.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	_, err = s.reviewRepo.Create(ctx, review)
	return err
}

func nextRound(latest *formulalibrary.FormulaLibraryReview) int32 {
	if latest == nil {
		return 1
	}
	if latest.Decision.ClosesRound() {
		return latest.Round + 1
	}
	return latest.Round
}

func (s *Service) approvedVersionNumber(
	ctx context.Context,
	libraryID pulid.ID,
	tenantInfo pagination.TenantInfo,
) int64 {
	version, err := s.versionRepo.GetLatestByStatus(
		ctx,
		&formulalibraryports.GetLatestLibraryVersionByStatusRequest{
			TenantInfo: tenantInfo,
			LibraryID:  libraryID,
			Status:     formulatemplate.StatusActive,
		},
	)
	if err != nil || version == nil {
		return 0
	}
	return version.VersionNumber
}

func staleSubmissionError() error {
	return errortypes.NewValidationError(
		"submittedAt",
		errortypes.ErrInvalid,
		"This submission is older than 14 days; ask the author to resubmit",
	)
}
