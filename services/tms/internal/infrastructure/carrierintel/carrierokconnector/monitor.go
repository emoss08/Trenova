package carrierokconnector

import (
	"bytes"
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/carrierintel/intelkit"
	"github.com/emoss08/trenova/shared/carrierok"
	"github.com/emoss08/trenova/shared/jsonflex"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	defaultMonitoringPageSize = 100
	maxMonitoringPageSize     = 500
	changeDateType            = "last_changed_date"
	maxFailureMessageLength   = 200

	messageEnrollRejected   = "CarrierOk did not accept this profile for monitoring"
	messageUnenrollRejected = "CarrierOk did not remove this profile from monitoring"
)

func (c *Client) MonitoringRef(
	profile *carrierintel.Profile,
	dotNumber, docketNumber string,
) string {
	if profile != nil && profile.Identity != nil && profile.Identity.DocketNumber != "" {
		if ref := profile.Identity.ProviderRef(); ref != "" {
			return ref
		}
	}

	dot := stringutils.DigitsOnly(dotNumber)
	if dot == "" {
		dot = profile.DOTNumber()
	}
	if dot == "" {
		return ""
	}

	prefix, number := intelkit.SplitDocket(docketNumber)
	if number == "" {
		return dot
	}
	return carrierok.ProfileID(dot, intelkit.DocketPrefixOrDefault(prefix), number)
}

func (c *Client) Enroll(
	ctx context.Context,
	refs []string,
) (*services.CarrierIntelEnrollResult, error) {
	return c.writeMonitoring(ctx, &monitoringWrite{
		refs:     refs,
		endpoint: carrierintel.EndpointMonitorAdd,
		call:     c.sdk.AddToMonitoring,
		rejected: messageEnrollRejected,
		applied:  func(result *carrierok.MonitoringWriteResult) int64 { return result.Added },
	})
}

func (c *Client) Unenroll(
	ctx context.Context,
	refs []string,
) (*services.CarrierIntelEnrollResult, error) {
	return c.writeMonitoring(ctx, &monitoringWrite{
		refs:     refs,
		endpoint: carrierintel.EndpointMonitorRemove,
		call:     c.sdk.RemoveFromMonitoring,
		rejected: messageUnenrollRejected,
		applied:  func(result *carrierok.MonitoringWriteResult) int64 { return result.Removed },
	})
}

type monitoringWrite struct {
	refs     []string
	endpoint carrierintel.Endpoint
	call     func(context.Context, []string) (*carrierok.MonitoringWriteResult, error)
	rejected string
	applied  func(*carrierok.MonitoringWriteResult) int64
}

func (c *Client) writeMonitoring(
	ctx context.Context,
	write *monitoringWrite,
) (*services.CarrierIntelEnrollResult, error) {
	refs := sliceutils.DedupeStrings(write.refs)
	if len(refs) == 0 {
		return &services.CarrierIntelEnrollResult{}, nil
	}

	outcome := &intelkit.CallOutcome{
		Endpoint: write.endpoint,
		Started:  time.Now(),
		Units:    len(refs),
	}
	result, err := write.call(ctx, refs)
	if err != nil {
		outcome.RawErr = err
		outcome.Mapped = errorMapper.Map(err)
		c.recorder.Record(ctx, outcome)
		return nil, outcome.Mapped
	}

	enrollResult := splitWriteResult(refs, result, write)
	outcome.Status = result.StatusCode
	outcome.Found = len(enrollResult.Succeeded) > 0
	outcome.Units = len(enrollResult.Succeeded)
	c.recorder.Record(ctx, outcome)
	return enrollResult, nil
}

func splitWriteResult(
	refs []string,
	result *carrierok.MonitoringWriteResult,
	write *monitoringWrite,
) *services.CarrierIntelEnrollResult {
	failedMessages := make(map[string]string, len(result.Failed))
	for _, failure := range result.Failed {
		ref := strings.ToUpper(strings.TrimSpace(failure.ProfileID))
		if ref == "" {
			continue
		}
		failedMessages[ref] = failureMessage(failure.Error, write.rejected)
	}

	rejectAll := len(failedMessages) == 0 && write.applied(result) == 0 &&
		strings.TrimSpace(result.Error) != ""

	out := &services.CarrierIntelEnrollResult{
		Succeeded: make([]string, 0, len(refs)),
		Failed:    make([]services.CarrierIntelEnrollFailure, 0, len(failedMessages)),
	}
	for _, ref := range refs {
		if rejectAll {
			out.Failed = append(out.Failed, services.CarrierIntelEnrollFailure{
				ProviderRef: ref,
				Message:     failureMessage(result.Error, write.rejected),
			})
			continue
		}
		if message, failed := failedMessages[strings.ToUpper(ref)]; failed {
			out.Failed = append(out.Failed, services.CarrierIntelEnrollFailure{
				ProviderRef: ref,
				Message:     message,
			})
			continue
		}
		out.Succeeded = append(out.Succeeded, ref)
	}
	return out
}

func failureMessage(vendorMessage, fallback string) string {
	if trimmed := strings.TrimSpace(vendorMessage); trimmed != "" {
		return stringutils.TruncateRunes(trimmed, maxFailureMessageLength)
	}
	return fallback
}

func (c *Client) ListChanges(
	ctx context.Context,
	req *services.CarrierIntelChangeFeedRequest,
) (*services.CarrierIntelChangeFeedPage, error) {
	if req == nil {
		req = &services.CarrierIntelChangeFeedRequest{}
	}
	page, pageSize := normalizePage(req.Page, req.PageSize)
	viewChanges := true
	params := carrierok.MonitoringListParams{
		Page:        page,
		PageSize:    pageSize,
		ViewChanges: &viewChanges,
		DateType:    changeDateType,
	}
	if req.Since > 0 {
		params.DateMin = time.Unix(req.Since, 0).UTC()
	}
	if req.Until > 0 {
		params.DateMax = time.Unix(req.Until, 0).UTC()
	}
	if params.DateMin.IsZero() && params.DateMax.IsZero() {
		params.DateType = ""
	}

	result, err := c.listMonitoring(ctx, &params)
	if err != nil {
		return nil, err
	}

	items := make([]services.CarrierIntelChangedProfile, 0, len(result.Profiles))
	for idx := range result.Profiles {
		items = append(items, changedProfile(&result.Profiles[idx]))
	}
	total := int(max(result.TotalCount, int64(len(items))))
	return &services.CarrierIntelChangeFeedPage{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		HasMore:  page*pageSize < total,
	}, nil
}

func (c *Client) ListWatchlist(
	ctx context.Context,
	page, pageSize int,
) (*services.CarrierIntelWatchlistPage, error) {
	page, pageSize = normalizePage(page, pageSize)
	result, err := c.listMonitoring(ctx, &carrierok.MonitoringListParams{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, err
	}

	refs := make([]string, 0, len(result.Profiles))
	for idx := range result.Profiles {
		refs = append(refs, result.Profiles[idx].ProfileID)
	}
	total := int(max(result.TotalCount, int64(len(refs))))
	return &services.CarrierIntelWatchlistPage{
		ProviderRefs: refs,
		Total:        total,
		HasMore:      page*pageSize < total,
	}, nil
}

func (c *Client) listMonitoring(
	ctx context.Context,
	params *carrierok.MonitoringListParams,
) (*carrierok.MonitoringListResult, error) {
	outcome := &intelkit.CallOutcome{
		Endpoint: carrierintel.EndpointMonitorList,
		Started:  time.Now(),
		Units:    1,
	}
	result, err := c.sdk.ListMonitoring(ctx, *params)
	if err != nil {
		outcome.RawErr = err
		outcome.Mapped = errorMapper.Map(err)
		c.recorder.Record(ctx, outcome)
		return nil, outcome.Mapped
	}
	outcome.Found = true
	c.recorder.Record(ctx, outcome)
	return result, nil
}

func normalizePage(page, pageSize int) (normalizedPage, normalizedSize int) {
	normalizedPage = max(page, 1)
	normalizedSize = pageSize
	if normalizedSize <= 0 {
		normalizedSize = defaultMonitoringPageSize
	}
	return normalizedPage, min(normalizedSize, maxMonitoringPageSize)
}

func changedProfile(monitored *carrierok.MonitoredProfile) services.CarrierIntelChangedProfile {
	changes := make([]services.CarrierIntelVendorChange, 0, len(monitored.Changes))
	for idx := range monitored.Changes {
		change := &monitored.Changes[idx]
		if !change.Changed && (change.Prior == nil || bytes.Equal(change.Prior, change.Current)) {
			continue
		}
		path := vendorFieldPath(change.Field)
		changes = append(changes, services.CarrierIntelVendorChange{
			VendorField: change.Field,
			Path:        path,
			Section:     carrierintel.SectionForPath(path),
			Prior:       decodeChangeValue(change.Prior),
			Current:     decodeChangeValue(change.Current),
		})
	}

	return services.CarrierIntelChangedProfile{
		ProviderRef: monitored.ProfileID,
		DOTNumber:   monitored.DOTNumber,
		ChangedAt:   parseChangedAt(monitored.Meta[changeDateType]),
		Changes:     changes,
	}
}

func decodeChangeValue(raw []byte) any {
	if len(raw) == 0 {
		return nil
	}
	var value any
	if err := sonic.Unmarshal(raw, &value); err != nil {
		return string(raw)
	}
	return value
}

func parseChangedAt(value string) *int64 {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	parsed, ok := jsonflex.ParseTime([]byte(strconv.Quote(trimmed)))
	if !ok {
		return nil
	}
	return parsed.Unix()
}
