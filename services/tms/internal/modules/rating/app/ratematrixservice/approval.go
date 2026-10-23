package ratematrixservice

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
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

type matrixTransition = approvalworkflow.Transition[*ratematrix.RateMatrix, formulatemplate.Status]

type SubmitRequest struct {
	ApprovalActionRequest
	ReviewerID   pulid.ID
	ReviewerNote string
}

func (s *Service) approvals(
	ctx context.Context,
) approvalworkflow.Engine[*ratematrix.RateMatrix, formulatemplate.Status] {
	return approvalworkflow.Engine[*ratematrix.RateMatrix, formulatemplate.Status]{
		Label: "rate matrix",
		Load: func(
			ctx context.Context,
			id pulid.ID,
			tenant pagination.TenantInfo,
		) (*ratematrix.RateMatrix, error) {
			return s.repo.GetByID(ctx, &ratematrixports.GetRateMatrixByIDRequest{
				RateMatrixID:      id,
				TenantInfo:        tenant,
				IncludeDimensions: true,
				ForUpdate:         true,
			})
		},
		Save:     s.repo.UpdateHeader,
		StatusOf: func(m *ratematrix.RateMatrix) formulatemplate.Status { return m.Status },
		SetStatus: func(m *ratematrix.RateMatrix, status formulatemplate.Status) {
			m.Status = status
		},
		CanTransition: formulatemplate.CanTransition,
		Snapshot: func(m *ratematrix.RateMatrix) *ratematrix.RateMatrix {
			snapshot := *m
			return &snapshot
		},
		Audit: func(
			updated, original *ratematrix.RateMatrix,
			operation permission.Operation,
			req *ApprovalActionRequest,
			comment string,
		) {
			s.audit(ctx, &auditRecord{
				entity:    updated,
				original:  original,
				operation: operation,
				userID:    req.TenantInfo.UserID,
				comment:   comment,
				critical:  operation == permission.OpApprove || operation == permission.OpReject,
			})
		},
	}
}

func (s *Service) Submit(ctx context.Context, req *SubmitRequest) (*ratematrix.RateMatrix, error) {
	var submitted *ratematrix.RateMatrix
	err := s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		result, txErr := s.approvals(ctx).Apply(txCtx, &req.ApprovalActionRequest, matrixTransition{
			Operation:    "Submit",
			From:         formulatemplate.StatusDraft,
			AlsoFrom:     []formulatemplate.Status{formulatemplate.StatusInactive},
			To:           formulatemplate.StatusInReview,
			PermissionOp: permission.OpSubmit,
			AuditComment: titleSubmitted,
			Validate: func(
				vCtx context.Context,
				matrix *ratematrix.RateMatrix,
				_ *ApprovalActionRequest,
			) error {
				return s.validateSubmission(vCtx, matrix, req)
			},
			Apply: func(matrix *ratematrix.RateMatrix, r *ApprovalActionRequest, now int64) {
				userID := r.TenantInfo.UserID
				matrix.SubmittedByID = &userID
				matrix.SubmittedAt = &now
				matrix.ApprovedByID = nil
				matrix.ApprovedAt = nil
				matrix.ReviewComment = r.Comment
			},
			AfterSave: func(
				aCtx context.Context,
				updated *ratematrix.RateMatrix,
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
	matrix *ratematrix.RateMatrix,
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
				"Choose an active reviewer, other than you, who can publish rate matrices",
			)
		}
	}
	if err := s.publishGate(ctx, matrix, multiErr); err != nil {
		return err
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (s *Service) publishGate(
	ctx context.Context,
	matrix *ratematrix.RateMatrix,
	multiErr *errortypes.MultiError,
) error {
	if !matrix.HasDraftChanges {
		if matrix.Status != formulatemplate.StatusInactive {
			multiErr.Add(
				"hasDraftChanges",
				errortypes.ErrInvalid,
				"The draft is the same as published version {0}; change it before submitting",
				matrix.PublishedVersionNumber,
			)
		}
		return nil
	}

	matrix.Validate(multiErr)
	if matrix.DraftCellCount == 0 {
		multiErr.Add(
			"cells",
			errortypes.ErrRequired,
			"Add at least one cell before submitting the draft",
		)
		return nil
	}
	misfits, err := s.repo.CountMisfitCells(ctx, &ratematrixports.CountMisfitCellsRequest{
		TenantInfo:    matrixTenant(matrix),
		RateMatrixID:  matrix.ID,
		VersionNumber: ratematrix.DraftVersion,
		Dimensions:    matrix.OrderedDimensions(),
	})
	if err != nil {
		return err
	}
	if misfits > 0 {
		multiErr.Add(
			"cells",
			errortypes.ErrInvalid,
			"{0, plural, one {# cell does} other {# cells do}} not fit the axes as they are now; load the grid again",
			misfits,
		)
	}
	return nil
}

func (s *Service) Approve(
	ctx context.Context,
	req *ApprovalActionRequest,
) (*ratematrix.RateMatrix, error) {
	var approved *ratematrix.RateMatrix
	var publishes bool
	tenant := req.TenantInfo
	err := s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		result, txErr := s.approvals(ctx).Apply(txCtx, req, matrixTransition{
			Operation:    "Approve",
			From:         formulatemplate.StatusInReview,
			To:           formulatemplate.StatusActive,
			PermissionOp: permission.OpApprove,
			AuditComment: "Rate matrix version published",
			Validate: func(
				vCtx context.Context,
				matrix *ratematrix.RateMatrix,
				r *ApprovalActionRequest,
			) error {
				if err := refuseOwnReview(matrix, r.TenantInfo.UserID); err != nil {
					return err
				}
				if formulatemplate.SubmissionIsStale(matrix.SubmittedAt, timeutils.NowUnix()) {
					return staleSubmissionError()
				}
				multiErr := errortypes.NewMultiError()
				if err := s.publishGate(vCtx, matrix, multiErr); err != nil {
					return err
				}
				if multiErr.HasErrors() {
					return multiErr
				}
				publishes = matrix.HasDraftChanges
				return nil
			},
			Apply: func(matrix *ratematrix.RateMatrix, r *ApprovalActionRequest, now int64) {
				userID := r.TenantInfo.UserID
				matrix.ApprovedByID = &userID
				matrix.ApprovedAt = &now
				matrix.ReviewComment = r.Comment
				if !publishes {
					return
				}
				matrix.PublishedVersionNumber++
				matrix.PublishedAt = &now
				matrix.HasDraftChanges = false
				matrix.DraftEditorIDs = []pulid.ID{}
			},
			AfterSave: func(
				aCtx context.Context,
				updated *ratematrix.RateMatrix,
				r *ApprovalActionRequest,
			) error {
				if publishes {
					if err := s.publish(aCtx, updated, r.TenantInfo.UserID); err != nil {
						return err
					}
				}
				return s.recordReview(aCtx, updated, r.TenantInfo, &reviewRecord{
					Comment:  r.Comment,
					Decision: formulatemplate.ReviewDecisionApproved,
				})
			},
		})
		if txErr != nil {
			return txErr
		}
		approved = result
		s.invalidateServed(txCtx, tenant)
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.notifyReviewOutcome(ctx, req.TenantInfo, &reviewOutcome{
		Matrix:      approved,
		SubmitterID: approved.SubmittedByID,
		Decision:    formulatemplate.ReviewDecisionApproved,
		Comment:     req.Comment,
	})
	return approved, nil
}

func (s *Service) publish(
	ctx context.Context,
	matrix *ratematrix.RateMatrix,
	publisherID pulid.ID,
) error {
	version := ratematrix.NewVersionFromMatrix(matrix, matrix.PublishedVersionNumber, publisherID)
	if _, err := s.versionRepo.Create(ctx, version); err != nil {
		return err
	}
	copied, err := s.repo.CopyVersion(ctx, &ratematrixports.CopyVersionRequest{
		TenantInfo:   matrixTenant(matrix),
		RateMatrixID: matrix.ID,
		From:         ratematrix.DraftVersion,
		To:           version.VersionNumber,
	})
	if err != nil {
		return err
	}
	if copied != matrix.DraftCellCount {
		s.l.Warn("published cell count differs from the draft's recorded count",
			zap.String("rateMatrixID", matrix.ID.String()),
			zap.Int64("copied", copied),
			zap.Int64("recorded", matrix.DraftCellCount),
		)
	}
	return nil
}

func refuseOwnReview(matrix *ratematrix.RateMatrix, userID pulid.ID) error {
	if matrix.SubmittedByID != nil && *matrix.SubmittedByID == userID {
		return errortypes.NewValidationError(
			"approvedById",
			errortypes.ErrInvalid,
			"A rate matrix cannot be published by its submitter",
		)
	}
	if matrix.EditedBy(userID) {
		return errortypes.NewValidationError(
			"approvedById",
			errortypes.ErrInvalid,
			"You edited this draft, so another reviewer must publish it",
		)
	}
	return nil
}

func (s *Service) Reject(
	ctx context.Context,
	req *ApprovalActionRequest,
) (*ratematrix.RateMatrix, error) {
	if err := approvalworkflow.RequireComment(req, "rejecting a rate matrix draft"); err != nil {
		return nil, err
	}

	var submitterID *pulid.ID
	var publishedVersion int64
	var rejected *ratematrix.RateMatrix
	err := s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		current, err := s.repo.GetByID(txCtx, &ratematrixports.GetRateMatrixByIDRequest{
			RateMatrixID: req.EntityID,
			TenantInfo:   req.TenantInfo,
		})
		if err != nil {
			return err
		}
		publishedVersion = current.PublishedVersionNumber
		destination := formulatemplate.StatusInactive
		if publishedVersion > 0 {
			destination = formulatemplate.StatusDraft
		}

		result, txErr := s.approvals(ctx).Apply(txCtx, req, matrixTransition{
			Operation:    "Reject",
			From:         formulatemplate.StatusInReview,
			To:           destination,
			PermissionOp: permission.OpReject,
			AuditComment: "Rate matrix draft rejected",
			Validate: func(
				_ context.Context,
				matrix *ratematrix.RateMatrix,
				r *ApprovalActionRequest,
			) error {
				return refuseOwnDecision(matrix, r.TenantInfo.UserID)
			},
			Apply: func(matrix *ratematrix.RateMatrix, r *ApprovalActionRequest, _ int64) {
				submitterID = pulid.ClonePointer(matrix.SubmittedByID)
				matrix.SubmittedByID = nil
				matrix.SubmittedAt = nil
				matrix.ReviewComment = r.Comment
			},
			AfterSave: func(
				aCtx context.Context,
				updated *ratematrix.RateMatrix,
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
		Matrix:           rejected,
		SubmitterID:      submitterID,
		Decision:         formulatemplate.ReviewDecisionRejected,
		Comment:          req.Comment,
		PublishedVersion: publishedVersion,
	})
	return rejected, nil
}

func refuseOwnDecision(matrix *ratematrix.RateMatrix, userID pulid.ID) error {
	if matrix.SubmittedByID != nil && *matrix.SubmittedByID == userID {
		return errortypes.NewValidationError(
			"approvedById",
			errortypes.ErrInvalid,
			"A rate matrix's submitter cannot review it; ask another reviewer",
		)
	}
	return nil
}

func (s *Service) RequestChanges(
	ctx context.Context,
	req *ApprovalActionRequest,
) (*ratematrix.RateMatrix, error) {
	if err := approvalworkflow.RequireComment(req, "requesting changes"); err != nil {
		return nil, err
	}

	var submitterID *pulid.ID
	var returned *ratematrix.RateMatrix
	err := s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		result, txErr := s.approvals(ctx).Apply(txCtx, req, matrixTransition{
			Operation:    "RequestChanges",
			From:         formulatemplate.StatusInReview,
			To:           formulatemplate.StatusDraft,
			PermissionOp: permission.OpReject,
			AuditComment: "Changes requested on rate matrix draft",
			Validate: func(
				_ context.Context,
				matrix *ratematrix.RateMatrix,
				r *ApprovalActionRequest,
			) error {
				return refuseOwnDecision(matrix, r.TenantInfo.UserID)
			},
			Apply: func(matrix *ratematrix.RateMatrix, r *ApprovalActionRequest, _ int64) {
				submitterID = pulid.ClonePointer(matrix.SubmittedByID)
				matrix.ReviewComment = r.Comment
			},
			AfterSave: func(
				aCtx context.Context,
				updated *ratematrix.RateMatrix,
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
		Matrix:      returned,
		SubmitterID: submitterID,
		Decision:    formulatemplate.ReviewDecisionChangesRequested,
		Comment:     req.Comment,
	})
	return returned, nil
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
		&ratematrixports.ListStaleSubmissionsRequest{
			SubmittedBefore: now - formulatemplate.SubmissionExpiry,
			Limit:           req.Limit,
		},
	)
	if err != nil {
		return nil, err
	}

	result := &ExpireStaleSubmissionsResult{Expired: make([]pulid.ID, 0, len(stale))}
	for _, matrix := range stale {
		if matrix == nil || !formulatemplate.SubmissionIsStale(matrix.SubmittedAt, now) {
			continue
		}
		tenantInfo := matrixTenant(matrix)
		tenantCtx := dbscope.WithTenant(ctx, tenantInfo.DBTenant())
		if expireErr := s.expireSubmission(tenantCtx, matrix, tenantInfo); expireErr != nil {
			s.l.Error("failed to expire stale rate matrix submission",
				zap.String("rateMatrixID", matrix.ID.String()),
				zap.Error(expireErr),
			)
			continue
		}
		result.Expired = append(result.Expired, matrix.ID)
	}
	return result, nil
}

func (s *Service) expireSubmission(
	ctx context.Context,
	matrix *ratematrix.RateMatrix,
	tenantInfo pagination.TenantInfo,
) error {
	submitterID := pulid.ClonePointer(matrix.SubmittedByID)
	original := *matrix
	matrix.Status = formulatemplate.StatusDraft
	matrix.SubmittedByID = nil
	matrix.SubmittedAt = nil
	matrix.ReviewComment = "Submission expired after waiting for review"

	var updated *ratematrix.RateMatrix
	err := s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		saved, txErr := s.repo.UpdateHeader(txCtx, matrix)
		if txErr != nil {
			return txErr
		}
		if txErr = s.recordReview(txCtx, saved, tenantInfo, &reviewRecord{
			Comment:  matrix.ReviewComment,
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

	s.audit(ctx, &auditRecord{
		entity:    updated,
		original:  &original,
		operation: permission.OpUpdate,
		comment:   "Rate matrix submission expired",
	})
	s.notifyReviewOutcome(ctx, tenantInfo, &reviewOutcome{
		Matrix:      updated,
		SubmitterID: submitterID,
		Decision:    formulatemplate.ReviewDecisionExpired,
		Comment:     matrix.ReviewComment,
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
	matrix *ratematrix.RateMatrix,
	tenantInfo pagination.TenantInfo,
	record *reviewRecord,
) error {
	latest, err := s.reviewRepo.Latest(ctx, matrix.ID, tenantInfo)
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

	review := &ratematrix.RateMatrixReview{
		RateMatrixID:      matrix.ID,
		OrganizationID:    matrix.OrganizationID,
		BusinessUnitID:    matrix.BusinessUnitID,
		Round:             round,
		Decision:          record.Decision,
		Comment:           record.Comment,
		ReviewerNote:      strings.TrimSpace(record.ReviewerNote),
		BaseVersionNumber: matrix.PublishedVersionNumber,
	}
	if record.Decision == formulatemplate.ReviewDecisionApproved && latest != nil {
		review.BaseVersionNumber = latest.BaseVersionNumber
	}
	if record.RequestedReviewerID.IsNotNil() {
		reviewerID := record.RequestedReviewerID
		review.RequestedReviewerID = &reviewerID
	}
	if record.Decision != formulatemplate.ReviewDecisionExpired && tenantInfo.UserID.IsNotNil() {
		actor := tenantInfo.UserID
		review.ActorID = &actor
	}

	multiErr := errortypes.NewMultiError()
	review.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	_, err = s.reviewRepo.Create(ctx, review)
	return err
}

func nextRound(latest *ratematrix.RateMatrixReview) int32 {
	if latest == nil {
		return 1
	}
	if latest.Decision.ClosesRound() {
		return latest.Round + 1
	}
	return latest.Round
}

func staleSubmissionError() error {
	return errortypes.NewValidationError(
		"submittedAt",
		errortypes.ErrInvalid,
		"This submission is older than 14 days; ask the author to resubmit",
	)
}

type Reviewer struct {
	ID            pulid.ID `json:"id"`
	Name          string   `json:"name"`
	Username      string   `json:"username"`
	ProfilePicURL string   `json:"profilePicUrl,omitempty"`
}

func (s *Service) ListReviewers(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]Reviewer, error) {
	if s.roles == nil {
		return nil, errortypes.NewBusinessError(
			"Reviewers for rate matrices cannot be listed on this server",
		)
	}

	permitted, err := s.roles.ListUsersWithPermission(
		ctx,
		permissionports.ListUsersWithPermissionRequest{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			Resource:       permission.ResourceRateMatrix,
			Operation:      permission.OpApprove,
			Now:            timeutils.NowUnix(),
		},
	)
	if err != nil {
		return nil, err
	}

	reviewers := make([]Reviewer, 0, len(permitted))
	for _, user := range permitted {
		if user.UserID.IsNil() || user.UserID == tenantInfo.UserID {
			continue
		}
		reviewers = append(reviewers, Reviewer{
			ID:            user.UserID,
			Name:          user.Name,
			Username:      user.Username,
			ProfilePicURL: user.ProfilePicURL,
		})
	}
	return reviewers, nil
}

func (s *Service) isEligibleReviewer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	reviewerID pulid.ID,
) (bool, error) {
	reviewers, err := s.ListReviewers(ctx, tenantInfo)
	if err != nil {
		return false, err
	}
	for _, reviewer := range reviewers {
		if reviewer.ID == reviewerID {
			return true, nil
		}
	}
	return false, nil
}
