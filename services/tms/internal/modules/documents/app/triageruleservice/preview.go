package triageruleservice

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/app/triagesupport"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/hashutils"
)

func (s *Service) Preview(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	input *triage.Rule,
) (*inboundmessageports.RulePreview, error) {
	rule := sentenceOf(input)
	multiErr := errortypes.NewMultiError()
	rule.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return s.preview(ctx, tenantInfo, rule)
}

func (s *Service) Matches(
	ctx context.Context,
	req triageports.GetRuleRequest,
) (*inboundmessageports.RulePreview, error) {
	rule, err := s.rules.GetByID(ctx, req)
	if err != nil {
		return nil, err
	}

	return s.preview(ctx, req.TenantInfo, rule)
}

func (s *Service) preview(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	rule *triage.Rule,
) (*inboundmessageports.RulePreview, error) {
	return triagesupport.GetOrCompute(
		ctx,
		s.cache,
		s.l,
		&triagesupport.Request{
			Section:    triageports.CachePreview,
			TenantInfo: tenantInfo,
			Variant:    SentenceHash(rule),
			TTL:        previewTTL,
		},
		func(ctx context.Context) (*inboundmessageports.RulePreview, error) {
			since := s.clock.DaysAgo(triagesupport.PreviewWindowDays)
			candidates, err := s.insights.RuleCandidates(ctx, &triageports.RuleCandidatesRequest{
				TenantInfo:     tenantInfo,
				Classification: rule.Classification,
				SenderScope:    rule.SenderScope,
				Sender:         rule.Sender,
				MailboxID:      rule.MailboxID,
				Since:          since,
				Limit:          maxPreviewCandidates,
			})
			if err != nil {
				return nil, err
			}

			return BuildPreview(rule, candidates, since), nil
		},
	)
}

func SentenceHash(rule *triage.Rule) string {
	return hashutils.SHA256Hex(strings.Join([]string{
		string(rule.Classification),
		string(rule.SenderScope),
		rule.Sender,
		rule.MailboxID.String(),
		string(rule.Action),
		rule.AssigneeID.String(),
	}, "|"))[:32]
}

func BuildPreview(
	rule *triage.Rule,
	candidates *triageports.RuleCandidates,
	since int64,
) *inboundmessageports.RulePreview {
	matches := make([]*inboundmessageports.RuleMatch, 0, len(candidates.Rows))
	rejected := 0
	off := 0
	for _, row := range candidates.Rows {
		if !rule.Covers(row.Message()) {
			rejected++

			continue
		}

		sample := triagesupport.SampleOf(row)
		match := &inboundmessageports.RuleMatch{
			MessageID:   sample.MessageID,
			FromName:    sample.FromName,
			FromAddress: sample.FromAddress,
			Subject:     sample.Subject,
			ReceivedAt:  sample.ReceivedAt,
			Reference:   sample.Reference,
		}
		if mismatch, found := MismatchOf(rule, row); found {
			match.Mismatch = &mismatch
			off++
		}
		matches = append(matches, match)
	}

	slices.SortStableFunc(matches, func(a, b *inboundmessageports.RuleMatch) int {
		if (a.Mismatch != nil) != (b.Mismatch != nil) {
			if a.Mismatch != nil {
				return -1
			}

			return 1
		}

		return cmp.Compare(b.ReceivedAt, a.ReceivedAt)
	})
	if len(matches) > maxPreviewMatches {
		matches = matches[:maxPreviewMatches]
	}

	return &inboundmessageports.RulePreview{
		Matched: max(candidates.Total-rejected, 0),
		Off:     off,
		Matches: matches,
		Since:   since,
	}
}

func MismatchOf(
	rule *triage.Rule,
	row *triageports.MessageRow,
) (inboundmessageports.RuleMismatch, bool) {
	if row.ClassificationCorrected {
		return inboundmessageports.MismatchKindCorrected, true
	}
	if lacksRecord(rule.Action, row) {
		return inboundmessageports.MismatchNoMatchedRecord, true
	}
	if personDisagreed(rule, row) {
		return inboundmessageports.MismatchPersonDisagreed, true
	}

	return "", false
}

func lacksRecord(action inboundmessage.NovaAction, row *triageports.MessageRow) bool {
	switch action {
	case inboundmessage.NovaActionFileOnShipment, inboundmessage.NovaActionFileAndQueuePay:
		return row.MatchedShipmentID.IsNil()
	case inboundmessage.NovaActionLinkBySender:
		return row.MatchedCustomerID.IsNil() && row.MatchedCarrierID.IsNil()
	case inboundmessage.NovaActionDraftReply, inboundmessage.NovaActionDraftLoad,
		inboundmessage.NovaActionIgnore, inboundmessage.NovaActionAssign:
		return false
	default:
		return false
	}
}

func personDisagreed(rule *triage.Rule, row *triageports.MessageRow) bool {
	if rule.Action == inboundmessage.NovaActionAssign {
		return row.OwnerID.IsNotNil() && row.OwnerID != rule.AssigneeID
	}
	if !row.SettledByPerson() {
		return false
	}

	ignored := row.Status == inboundmessage.StatusIgnored
	switch rule.Action {
	case inboundmessage.NovaActionIgnore:
		return !ignored
	case inboundmessage.NovaActionFileOnShipment, inboundmessage.NovaActionFileAndQueuePay:
		return ignored || !row.PersonFiled
	case inboundmessage.NovaActionDraftReply, inboundmessage.NovaActionDraftLoad,
		inboundmessage.NovaActionLinkBySender:
		return ignored
	case inboundmessage.NovaActionAssign:
		return false
	default:
		return false
	}
}
