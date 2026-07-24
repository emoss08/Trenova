package samsaraprovider

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/ports/services"
	sharedsamsara "github.com/emoss08/trenova/shared/samsara"
	"github.com/emoss08/trenova/shared/samsara/assets"
	"github.com/emoss08/trenova/shared/samsara/compliance"
	"github.com/emoss08/trenova/shared/samsara/drivers"
	"github.com/emoss08/trenova/shared/samsara/dvirs"
	"github.com/emoss08/trenova/shared/samsara/forms"
	"github.com/emoss08/trenova/shared/samsara/vehicles"
	"github.com/emoss08/trenova/shared/samsara/webhooks"
)

const pageLimit = 512

var vehicleStatsTypes = []string{"gps", "engineStates", "fuelPercents"}

type Provider struct {
	client *sharedsamsara.Client
}

func New(client *sharedsamsara.Client) *Provider {
	return &Provider{client: client}
}

func (p *Provider) Type() integration.Type {
	return integration.TypeSamsara
}

func (p *Provider) ListVehicles(ctx context.Context) ([]services.ProviderVehicle, error) {
	return p.listAssets(ctx, assets.TypeVehicle)
}

func (p *Provider) ListTrailers(ctx context.Context) ([]services.ProviderVehicle, error) {
	return p.listAssets(ctx, assets.TypeTrailer)
}

func (p *Provider) listAssets(
	ctx context.Context,
	assetType assets.Type,
) ([]services.ProviderVehicle, error) {
	out := make([]services.ProviderVehicle, 0)
	after := ""
	for {
		page, err := p.client.Assets.List(ctx, assets.ListParams{
			Type:  assetType,
			After: after,
			Limit: pageLimit,
		})
		if err != nil {
			return nil, fmt.Errorf("list samsara %s assets: %w", assetType, err)
		}

		for i := range page.Data {
			asset := &page.Data[i]
			vehicle := services.ProviderVehicle{ID: asset.Id}
			if asset.Name != nil {
				vehicle.Name = *asset.Name
			}
			if asset.Vin != nil {
				vehicle.VIN = *asset.Vin
			}
			if asset.LicensePlate != nil {
				vehicle.LicensePlate = *asset.LicensePlate
			}
			out = append(out, vehicle)
		}

		if !page.Pagination.HasNextPage || page.Pagination.EndCursor == "" {
			break
		}
		after = page.Pagination.EndCursor
	}
	return out, nil
}

func (p *Provider) ListPositions(ctx context.Context) ([]services.ProviderPosition, error) {
	stats, err := p.client.Vehicles.StatsAll(ctx, vehicles.StatsParams{
		Types: vehicleStatsTypes,
		Limit: pageLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch samsara vehicle stats: %w", err)
	}

	out := make([]services.ProviderPosition, 0, len(stats))
	for i := range stats {
		stat := &stats[i]
		if stat.Gps == nil {
			continue
		}

		position := services.ProviderPosition{
			VehicleID:  stat.Id,
			Latitude:   stat.Gps.Latitude,
			Longitude:  stat.Gps.Longitude,
			RecordedAt: parseTime(stat.Gps.Time),
		}
		if stat.Gps.HeadingDegrees != nil {
			position.HeadingDegrees = *stat.Gps.HeadingDegrees
		}
		if stat.Gps.SpeedMilesPerHour != nil {
			position.SpeedMph = *stat.Gps.SpeedMilesPerHour
		}
		if stat.Gps.ReverseGeo != nil && stat.Gps.ReverseGeo.FormattedLocation != nil {
			position.FormattedLocation = *stat.Gps.ReverseGeo.FormattedLocation
		}
		if stat.EngineState != nil {
			position.EngineState = telematics.EngineState(stat.EngineState.Value)
		}
		if stat.FuelPercent != nil {
			fuel := float64(stat.FuelPercent.Value)
			position.FuelPercent = &fuel
		}
		if stat.ObdOdometerMeters != nil {
			odometer := stat.ObdOdometerMeters.Value
			position.OdometerMeters = &odometer
		}
		out = append(out, position)
	}
	return out, nil
}

func (p *Provider) ListHOSClocks(ctx context.Context) ([]services.ProviderHOSClocks, error) {
	clocks, err := p.client.Compliance.HOSClocksAll(ctx, compliance.HOSClocksParams{
		Limit: pageLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch samsara hos clocks: %w", err)
	}

	out := make([]services.ProviderHOSClocks, 0, len(clocks))
	for i := range clocks {
		clock := &clocks[i]
		if clock.Driver == nil || clock.Driver.Id == nil {
			continue
		}

		record := services.ProviderHOSClocks{DriverID: *clock.Driver.Id}
		if clock.CurrentDutyStatus != nil && clock.CurrentDutyStatus.HosStatusType != nil {
			record.DutyStatus = telematics.DutyStatus(*clock.CurrentDutyStatus.HosStatusType)
		}
		if clock.CurrentVehicle != nil && clock.CurrentVehicle.Id != nil {
			record.CurrentVehicleID = *clock.CurrentVehicle.Id
		}
		applyClockDurations(&record, clock.Clocks)
		applyClockViolations(&record, clock.Violations)
		out = append(out, record)
	}
	return out, nil
}

func applyClockDurations(record *services.ProviderHOSClocks, clocks *compliance.HOSClockSet) {
	if clocks == nil {
		return
	}
	if clocks.Drive != nil && clocks.Drive.DriveRemainingDurationMs != nil {
		record.DriveRemainingMs = int64(*clocks.Drive.DriveRemainingDurationMs)
	}
	if clocks.Shift != nil && clocks.Shift.ShiftRemainingDurationMs != nil {
		record.ShiftRemainingMs = int64(*clocks.Shift.ShiftRemainingDurationMs)
	}
	if clocks.Break != nil && clocks.Break.TimeUntilBreakDurationMs != nil {
		record.BreakRemainingMs = int64(*clocks.Break.TimeUntilBreakDurationMs)
	}
	if clocks.Cycle == nil {
		return
	}
	if clocks.Cycle.CycleRemainingDurationMs != nil {
		record.CycleRemainingMs = int64(*clocks.Cycle.CycleRemainingDurationMs)
	}
	if clocks.Cycle.CycleTomorrowDurationMs != nil {
		record.CycleTomorrowMs = int64(*clocks.Cycle.CycleTomorrowDurationMs)
	}
	if clocks.Cycle.CycleStartedAtTime != nil {
		if startedAt := parseTime(*clocks.Cycle.CycleStartedAtTime); startedAt > 0 {
			record.CycleStartedAt = &startedAt
		}
	}
}

func applyClockViolations(
	record *services.ProviderHOSClocks,
	violations *compliance.HOSClockViolations,
) {
	if violations == nil {
		return
	}
	if violations.ShiftDrivingViolationDurationMs != nil {
		record.ShiftDrivingViolationMs = int64(*violations.ShiftDrivingViolationDurationMs)
	}
	if violations.CycleViolationDurationMs != nil {
		record.CycleViolationMs = int64(*violations.CycleViolationDurationMs)
	}
}

func (p *Provider) ListDriverProfiles(
	ctx context.Context,
) ([]services.ProviderDriverProfile, error) {
	remoteDrivers, err := p.client.Drivers.ListAll(ctx, drivers.ListParams{Limit: pageLimit})
	if err != nil {
		return nil, fmt.Errorf("list samsara drivers: %w", err)
	}

	out := make([]services.ProviderDriverProfile, 0, len(remoteDrivers))
	for i := range remoteDrivers {
		driver := &remoteDrivers[i]
		if driver.Id == nil {
			continue
		}
		profile := services.ProviderDriverProfile{DriverID: *driver.Id}
		if driver.Name != nil {
			profile.Name = *driver.Name
		}
		profile.Ruleset = mapDriverRuleset(driver)
		out = append(out, profile)
	}
	return out, nil
}

func mapDriverRuleset(driver *drivers.Driver) *services.ProviderRuleset {
	if driver.EldSettings == nil || driver.EldSettings.Rulesets == nil ||
		len(*driver.EldSettings.Rulesets) == 0 {
		return nil
	}
	first := (*driver.EldSettings.Rulesets)[0]
	ruleset := &services.ProviderRuleset{}
	if first.Cycle != nil {
		ruleset.Cycle = string(*first.Cycle)
	}
	if first.Shift != nil {
		ruleset.Shift = string(*first.Shift)
	}
	if first.Restart != nil {
		ruleset.Restart = string(*first.Restart)
	}
	if first.Break != nil {
		ruleset.Break = string(*first.Break)
	}
	if first.Jurisdiction != nil {
		ruleset.Jurisdiction = *first.Jurisdiction
	}
	return ruleset
}

func (p *Provider) ListHOSViolations(
	ctx context.Context,
	startAt int64,
	endAt int64,
) ([]services.ProviderViolation, error) {
	startTime := time.Unix(startAt, 0).UTC()
	endTime := time.Unix(endAt, 0).UTC()

	out := make([]services.ProviderViolation, 0)
	after := ""
	for {
		page, err := p.client.Compliance.HOSViolations(ctx, compliance.HOSViolationsParams{
			StartTime: &startTime,
			EndTime:   &endTime,
			After:     after,
		})
		if err != nil {
			return nil, fmt.Errorf("fetch samsara hos violations: %w", err)
		}

		for i := range page.Data {
			for j := range page.Data[i].Violations {
				violation := &page.Data[i].Violations[j]
				record, ok := mapViolation(violation)
				if !ok {
					continue
				}
				out = append(out, record)
			}
		}

		if !page.Pagination.HasNextPage || page.Pagination.EndCursor == "" {
			break
		}
		after = page.Pagination.EndCursor
	}
	return out, nil
}

func mapViolation(violation *compliance.HOSViolation) (services.ProviderViolation, bool) {
	violationType := string(violation.Type)
	if violationType == "" || violationType == "NONE" {
		return services.ProviderViolation{}, false
	}
	startAt := parseTime(violation.ViolationStartTime)
	if startAt == 0 {
		return services.ProviderViolation{}, false
	}

	record := services.ProviderViolation{
		DriverID:    violation.Driver.Id,
		Type:        violationType,
		Description: violation.Description,
		DurationMs:  violation.DurationMs,
		StartAt:     startAt,
	}
	if dayStart := parseTime(violation.Day.StartTime); dayStart > 0 {
		record.DayStartAt = &dayStart
	}
	if dayEnd := parseTime(violation.Day.EndTime); dayEnd > 0 {
		record.DayEndAt = &dayEnd
	}
	return record, true
}

func (p *Provider) ListHOSLogs(
	ctx context.Context,
	driverID string,
	startAt int64,
	endAt int64,
) ([]services.ProviderHOSLogEntry, error) {
	startTime := time.Unix(startAt, 0).UTC()
	endTime := time.Unix(endAt, 0).UTC()

	out := make([]services.ProviderHOSLogEntry, 0)
	after := ""
	for {
		page, err := p.client.Compliance.HOSLogs(ctx, compliance.HOSLogsParams{
			DriverIDs: []string{driverID},
			StartTime: &startTime,
			EndTime:   &endTime,
			After:     after,
		})
		if err != nil {
			return nil, fmt.Errorf("fetch samsara hos logs: %w", err)
		}

		for i := range page.Data {
			if page.Data[i].HosLogs == nil {
				continue
			}
			logs := *page.Data[i].HosLogs
			for j := range logs {
				out = append(out, mapLogEntry(&logs[j]))
			}
		}

		if !page.Pagination.HasNextPage || page.Pagination.EndCursor == "" {
			break
		}
		after = page.Pagination.EndCursor
	}
	return out, nil
}

func mapLogEntry(entry *compliance.HOSLogEntry) services.ProviderHOSLogEntry {
	out := services.ProviderHOSLogEntry{
		LogStartAt: parseTime(entry.LogStartTime),
	}
	if entry.HosStatusType != nil {
		out.HosStatusType = string(*entry.HosStatusType)
	}
	if entry.LogEndTime != nil {
		if endAt := parseTime(*entry.LogEndTime); endAt > 0 {
			out.LogEndAt = &endAt
		}
	}
	if entry.Remark != nil {
		out.Remark = *entry.Remark
	}
	if entry.Vehicle != nil {
		if entry.Vehicle.Id != nil {
			out.VehicleID = *entry.Vehicle.Id
		}
		if entry.Vehicle.Name != nil {
			out.VehicleName = *entry.Vehicle.Name
		}
	}
	if entry.LogRecordedLocation != nil {
		latitude := entry.LogRecordedLocation.Latitude
		longitude := entry.LogRecordedLocation.Longitude
		out.Latitude = &latitude
		out.Longitude = &longitude
	}
	if entry.Codrivers != nil {
		for _, codriver := range *entry.Codrivers {
			if codriver.Name != nil {
				out.Codrivers = append(out.Codrivers, *codriver.Name)
			}
		}
	}
	return out
}

func (p *Provider) ListHOSDailyLogs(
	ctx context.Context,
	driverID string,
	startDate string,
	endDate string,
) ([]services.ProviderHOSDailyLog, error) {
	out := make([]services.ProviderHOSDailyLog, 0)
	after := ""
	for {
		page, err := p.client.Compliance.HOSDailyLogs(ctx, compliance.HOSDailyLogsParams{
			DriverIDs: []string{driverID},
			StartDate: startDate,
			EndDate:   endDate,
			After:     after,
		})
		if err != nil {
			return nil, fmt.Errorf("fetch samsara hos daily logs: %w", err)
		}

		for i := range page.Data {
			out = append(out, mapDailyLog(&page.Data[i]))
		}

		if !page.Pagination.HasNextPage || page.Pagination.EndCursor == "" {
			break
		}
		after = page.Pagination.EndCursor
	}
	return out, nil
}

func mapDailyLog(day *compliance.HOSDailyLog) services.ProviderHOSDailyLog {
	out := services.ProviderHOSDailyLog{
		StartAt: parseTime(day.StartTime),
		EndAt:   parseTime(day.EndTime),
	}
	if day.DistanceTraveled != nil && day.DistanceTraveled.DriveDistanceMeters != nil {
		out.DriveDistanceMeters = *day.DistanceTraveled.DriveDistanceMeters
	}
	applyDailyDurations(&out, day.DutyStatusDurations)
	applyDailyMetadata(&out, day.LogMetaData)
	return out
}

func applyDailyDurations(
	out *services.ProviderHOSDailyLog,
	durations *compliance.HOSDailyLogDurations,
) {
	if durations == nil {
		return
	}
	out.ActiveDurationMs = int64Value(durations.ActiveDurationMs)
	out.DriveDurationMs = int64Value(durations.DriveDurationMs)
	out.OnDutyDurationMs = int64Value(durations.OnDutyDurationMs)
	out.OffDutyDurationMs = int64Value(durations.OffDutyDurationMs)
	out.SleeperBerthDurationMs = int64Value(durations.SleeperBerthDurationMs)
	out.PersonalConveyanceDurationMs = int64Value(durations.PersonalConveyanceDurationMs)
	out.YardMoveDurationMs = int64Value(durations.YardMoveDurationMs)
}

func applyDailyMetadata(
	out *services.ProviderHOSDailyLog,
	metadata *compliance.HOSDailyLogMetadata,
) {
	if metadata == nil {
		return
	}
	if metadata.IsCertified != nil {
		out.IsCertified = *metadata.IsCertified
	}
	if metadata.CertifiedAtTime != nil {
		if certifiedAt := parseTime(*metadata.CertifiedAtTime); certifiedAt > 0 {
			out.CertifiedAt = &certifiedAt
		}
	}
	if metadata.ShippingDocs != nil {
		out.ShippingDocs = *metadata.ShippingDocs
	}
	if metadata.Vehicles != nil {
		for _, vehicle := range *metadata.Vehicles {
			if vehicle.Name != nil {
				out.VehicleNames = append(out.VehicleNames, *vehicle.Name)
			}
		}
	}
}

func (p *Provider) ListDVIRs(
	ctx context.Context,
	startAt int64,
	endAt int64,
) ([]services.ProviderDVIR, error) {
	startTime := time.Unix(startAt, 0).UTC()
	endTime := time.Unix(endAt, 0).UTC()

	records, err := p.client.Dvirs.HistoryAll(ctx, dvirs.HistoryParams{
		StartTime: &startTime,
		EndTime:   &endTime,
		Limit:     pageLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch samsara dvirs: %w", err)
	}

	out := make([]services.ProviderDVIR, 0, len(records))
	for i := range records {
		out = append(out, mapDVIR(&records[i]))
	}
	return out, nil
}

func mapDVIR(record *dvirs.HistoryDVIR) services.ProviderDVIR {
	out := services.ProviderDVIR{ID: record.Id}
	if record.Type != nil {
		out.Type = string(*record.Type)
	}
	if record.SafetyStatus != nil {
		out.SafetyStatus = string(*record.SafetyStatus)
	}
	applyDVIRRefs(&out, record)
	if record.StartTime != nil {
		out.StartAt = parseTime(*record.StartTime)
	}
	if record.EndTime != nil {
		out.EndAt = parseTime(*record.EndTime)
	}
	if record.OdometerMeters != nil {
		odometer := int64(*record.OdometerMeters)
		out.OdometerMeters = &odometer
	}
	if record.Location != nil {
		out.Location = *record.Location
	}
	applyDVIRSignature(&out, record)
	out.Defects = mapDVIRDefects(record.VehicleDefects, record.TrailerDefects)
	return out
}

func applyDVIRRefs(out *services.ProviderDVIR, record *dvirs.HistoryDVIR) {
	if record.Vehicle != nil && record.Vehicle.Id != nil {
		out.VehicleID = *record.Vehicle.Id
	}
	if record.Trailer != nil && record.Trailer.Id != nil {
		out.TrailerID = *record.Trailer.Id
	}
	switch {
	case record.TrailerName != nil:
		out.TrailerName = *record.TrailerName
	case record.Trailer != nil && record.Trailer.Name != nil:
		out.TrailerName = *record.Trailer.Name
	}
}

func applyDVIRSignature(out *services.ProviderDVIR, record *dvirs.HistoryDVIR) {
	signature := record.AuthorSignature
	if signature == nil {
		return
	}
	if signature.SignatoryUser != nil {
		if signature.SignatoryUser.Id != nil {
			out.DriverID = *signature.SignatoryUser.Id
		}
		if signature.SignatoryUser.Name != nil {
			out.DriverName = *signature.SignatoryUser.Name
		}
	}
	if signature.SignedAtTime != nil && parseTime(*signature.SignedAtTime) > 0 {
		out.Signed = true
	}
}

func mapDVIRDefects(lists ...*[]dvirs.DVIRDefect) []services.ProviderDVIRDefect {
	total := 0
	for _, list := range lists {
		if list != nil {
			total += len(*list)
		}
	}
	if total == 0 {
		return nil
	}

	out := make([]services.ProviderDVIRDefect, 0, total)
	for _, list := range lists {
		if list == nil {
			continue
		}
		items := *list
		for i := range items {
			item := &items[i]
			defect := services.ProviderDVIRDefect{
				ID:       item.Id,
				Resolved: item.IsResolved,
			}
			if item.DefectType != nil {
				defect.DefectType = *item.DefectType
			}
			if item.Comment != nil {
				defect.Comment = *item.Comment
			}
			if item.ResolvedAtTime != nil {
				if resolvedAt := parseTime(*item.ResolvedAtTime); resolvedAt > 0 {
					defect.ResolvedAt = &resolvedAt
				}
			}
			out = append(out, defect)
		}
	}
	return out
}

func (p *Provider) ListFormSubmissions(
	ctx context.Context,
	driverID string,
	startAt int64,
	endAt int64,
) ([]services.ProviderFormSubmission, error) {
	startTime := time.Unix(startAt, 0).UTC()
	endTime := time.Unix(endAt, 0).UTC()

	submissions, err := p.client.Forms.StreamSubmissionsAll(ctx, forms.SubmissionStreamParams{
		StartTime: &startTime,
		EndTime:   &endTime,
		DriverIDs: []string{driverID},
	})
	if err != nil {
		return nil, fmt.Errorf("fetch samsara form submissions: %w", err)
	}
	if len(submissions) == 0 {
		return []services.ProviderFormSubmission{}, nil
	}

	templateNames, err := p.formTemplateNames(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]services.ProviderFormSubmission, 0, len(submissions))
	for i := range submissions {
		out = append(out, mapFormSubmission(&submissions[i], templateNames))
	}
	return out, nil
}

func (p *Provider) formTemplateNames(ctx context.Context) (map[string]string, error) {
	names := make(map[string]string)
	after := ""
	for {
		page, err := p.client.Forms.ListTemplates(ctx, forms.TemplateListParams{After: after})
		if err != nil {
			return nil, fmt.Errorf("list samsara form templates: %w", err)
		}
		for i := range page.Data {
			template := &page.Data[i]
			names[template.Id.String()] = template.Title
		}
		if !page.Pagination.HasNextPage || page.Pagination.EndCursor == "" {
			break
		}
		after = page.Pagination.EndCursor
	}
	return names, nil
}

func mapFormSubmission(
	submission *forms.FormSubmission,
	templateNames map[string]string,
) services.ProviderFormSubmission {
	templateID := submission.FormTemplate.Id.String()
	out := services.ProviderFormSubmission{
		ID:           submission.Id,
		TemplateID:   templateID,
		TemplateName: templateNames[templateID],
		DriverID:     submission.SubmittedBy.Id,
		RouteStopID:  forms.SubmissionRouteStopID(*submission),
		ExternalIDs:  forms.SubmissionExternalIDs(*submission),
		Fields:       make([]services.ProviderFormField, 0, len(submission.Fields)),
	}
	if !submission.SubmittedAtTime.IsZero() {
		out.SubmittedAt = submission.SubmittedAtTime.Unix()
	}
	if out.TemplateName == "" && submission.Title != nil {
		out.TemplateName = *submission.Title
	}
	for i := range submission.Fields {
		field := &submission.Fields[i]
		kind, value := forms.FieldTypedValue(*field)
		record := services.ProviderFormField{Type: kind, Value: value}
		if field.Label != nil {
			record.Label = *field.Label
		}
		out.Fields = append(out.Fields, record)
	}
	return out
}

func (p *Provider) VerifyWebhookSignature(
	secret string,
	timestamp string,
	body []byte,
	signature string,
	now time.Time,
	maxSkew time.Duration,
) error {
	return webhooks.VerifySignatureWithTolerance(secret, timestamp, body, signature, now, maxSkew)
}

func (p *Provider) ParseWebhookEvent(body []byte) (*services.ProviderWebhookEvent, error) {
	event, err := webhooks.ParseEvent(body)
	if err != nil {
		return nil, err
	}

	out := &services.ProviderWebhookEvent{
		EventID:    event.EventID,
		EventType:  string(event.EventType),
		OccurredAt: event.EventTime.Unix(),
		Payload:    event.Data,
	}

	out.Kind = services.ProviderEventKindOther

	switch event.EventType { //nolint:exhaustive // only entity-bearing event types enrich the record
	case webhooks.EventTypeGeofenceEntry:
		out.Kind = services.ProviderEventKindGeofenceEntry
		enrichGeofence(&event, out)
	case webhooks.EventTypeGeofenceExit:
		out.Kind = services.ProviderEventKindGeofenceExit
		enrichGeofence(&event, out)
	case webhooks.EventTypeRouteStopArrival:
		out.Kind = services.ProviderEventKindStopArrival
		enrichStop(&event, out)
	case webhooks.EventTypeRouteStopDeparture:
		out.Kind = services.ProviderEventKindStopDeparture
		enrichStop(&event, out)
	case webhooks.EventTypeFormSubmitted, webhooks.EventTypeFormUpdated,
		webhooks.EventTypeDocumentSubmitted:
		out.Kind = services.ProviderEventKindFormSubmission
		enrichForm(&event, out)
	case webhooks.EventTypeVehicleCreated, webhooks.EventTypeVehicleUpdated:
		if data, dataErr := event.VehicleData(); dataErr == nil {
			out.VehicleID = data.ID
		}
	case webhooks.EventTypeDriverCreated, webhooks.EventTypeDriverUpdated:
		if data, dataErr := event.DriverData(); dataErr == nil {
			out.DriverID = data.ID
		}
	default:
	}

	return out, nil
}

func enrichGeofence(event *webhooks.Event, out *services.ProviderWebhookEvent) {
	data, dataErr := event.GeofenceData()
	if dataErr != nil {
		return
	}
	geofence := &services.ProviderGeofenceEvent{}
	if data.Address != nil {
		geofence.AddressName = data.Address.Name
		geofence.AddressExternalIDs = data.Address.ExternalIDs
	}
	if data.Vehicle != nil {
		geofence.VehicleID = data.Vehicle.ID
		geofence.VehicleVIN = data.Vehicle.Vin
		out.VehicleID = data.Vehicle.ID
	}
	if data.Driver != nil {
		geofence.DriverID = data.Driver.ID
		out.DriverID = data.Driver.ID
	}
	out.Geofence = geofence
}

func enrichStop(event *webhooks.Event, out *services.ProviderWebhookEvent) {
	data, dataErr := event.RouteStopData()
	if dataErr != nil {
		return
	}
	stop := &services.ProviderStopEvent{}
	if data.Vehicle != nil {
		stop.VehicleID = data.Vehicle.ID
		stop.VehicleVIN = data.Vehicle.Vin
		stop.AddressExternalIDs = data.Vehicle.ExternalIDs
		out.VehicleID = data.Vehicle.ID
	}
	if data.Driver != nil {
		stop.DriverID = data.Driver.ID
		out.DriverID = data.Driver.ID
	}
	if data.RouteStop != nil {
		stop.RouteStopID = data.RouteStop.ID
		stop.StopExternalIDs = data.RouteStop.ExternalIDs
		if arrival := parseTime(data.RouteStop.ActualArrivalTime); arrival > 0 {
			stop.OccurredAt = arrival
		} else if departure := parseTime(data.RouteStop.ActualDepartureTime); departure > 0 {
			stop.OccurredAt = departure
		}
	}
	out.Stop = stop
}

func enrichForm(event *webhooks.Event, out *services.ProviderWebhookEvent) {
	data, dataErr := event.FormData()
	if dataErr != nil {
		return
	}
	form := &services.ProviderFormEvent{
		SubmissionID: data.FormID,
		TemplateID:   data.TemplateID,
		RouteStopID:  data.AssignedToRouteStopID,
		ExternalIDs:  data.ExternalIDs,
	}
	if data.SubmittedBy != nil && data.SubmittedBy.Type == "driver" {
		form.DriverID = data.SubmittedBy.ID
		out.DriverID = data.SubmittedBy.ID
	}
	if submittedAt := parseTime(data.SubmittedAtTime); submittedAt > 0 {
		form.SubmittedAt = submittedAt
	}
	form.Fields = make([]services.ProviderFormField, 0, len(data.Fields))
	for i := range data.Fields {
		field := &data.Fields[i]
		form.Fields = append(form.Fields, services.ProviderFormField{
			Label: field.Label,
			Type:  field.Type,
			Value: field.Value,
		})
	}
	out.Form = form
}

func int64Value(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func parseTime(value string) int64 {
	if value == "" {
		return 0
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return 0
	}
	return parsed.Unix()
}
