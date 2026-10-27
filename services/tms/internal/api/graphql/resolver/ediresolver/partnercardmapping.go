package ediresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
)

func ediPartnerCardToModel(card *ediservice.EDIPartnerCard) *gqlmodel.EDIPartnerCard {
	model := &gqlmodel.EDIPartnerCard{
		Partner:    card.Partner,
		Lading:     &gqlmodel.EDIPartnerLading{},
		Transports: ediPartnerTransportsToModel(card.Transports),
		Readiness:  ediPartnerReadinessToModel(card.Readiness),
		Mappings:   ediPartnerMappingsToModel(card.Mappings),
		Activity:   ediPartnerActivityToModel(card.Activity),
		Attention:  ediPartnerAttentionToModel(card.Attention),
	}
	if card.Lading != nil {
		model.Lading = &gqlmodel.EDIPartnerLading{
			Project: card.Lading.Project,
			Active:  card.Lading.Active,
			Shadow:  card.Lading.Shadow,
		}
	}
	if customer := card.Partner.Customer; customer != nil && card.Partner.CustomerID.IsNotNil() {
		model.Customer = &gqlmodel.EDIPartnerCustomer{
			ID:   customer.ID.String(),
			Name: customer.Name,
			Code: customer.Code,
		}
	}
	if card.Carriers != nil {
		model.Carriers = ediPartnerCarriersToModel(card.Carriers)
	}

	return model
}

func ediPartnerTransportsToModel(
	transports []*ediservice.EDIPartnerTransport,
) []*gqlmodel.EDIPartnerTransport {
	models := make([]*gqlmodel.EDIPartnerTransport, 0, len(transports))
	for _, transport := range transports {
		profile := transport.Profile
		models = append(models, &gqlmodel.EDIPartnerTransport{
			Profile:           profile,
			LastPollAttemptAt: base.IntPtr(profile.LastPollAttemptAt),
			LastPollSuccessAt: base.IntPtr(profile.LastPollSuccessAt),
			LastPollError:     base.NonEmptyPtr(transport.LastPollError),
			Credentials:       ediTransportCredentialsToModel(transport.Credentials),
			Certificates:      ediTransportCertificatesToModel(transport.Certificates),
		})
	}

	return models
}

func ediTransportCredentialsToModel(
	requirements []ediports.CredentialRequirement,
) []*gqlmodel.EDITransportCredential {
	models := make([]*gqlmodel.EDITransportCredential, 0, len(requirements))
	for index := range requirements {
		requirement := &requirements[index]
		if len(requirement.Keys) == 0 {
			continue
		}
		models = append(models, &gqlmodel.EDITransportCredential{
			Key:      requirement.Keys[0],
			Secret:   requirement.Kind == ediports.CredentialKindSecret,
			Required: requirement.Required,
			OnFile:   !requirement.Missing,
		})
	}

	return models
}

func ediTransportCertificatesToModel(
	certificates []ediservice.EDIPartnerCertificate,
) []*gqlmodel.EDITransportCertificate {
	models := make([]*gqlmodel.EDITransportCertificate, 0, len(certificates))
	for index := range certificates {
		certificate := &certificates[index]
		model := &gqlmodel.EDITransportCertificate{
			Role:     gqlmodel.EDITransportCertificateRole(certificate.Role),
			Readable: !certificate.Unreadable && certificate.Summary != nil,
		}
		if summary := certificate.Summary; model.Readable {
			notBefore := int(summary.NotBefore)
			notAfter := int(summary.NotAfter)
			model.Subject = base.NonEmptyPtr(summary.Subject)
			model.Issuer = base.NonEmptyPtr(summary.Issuer)
			model.NotBefore = &notBefore
			model.NotAfter = &notAfter
			model.Expired = summary.Expired
		}
		models = append(models, model)
	}

	return models
}

func ediPartnerReadinessToModel(
	readiness *ediservice.EDIPartnerReadiness,
) *gqlmodel.EDIPartnerCardReadiness {
	model := &gqlmodel.EDIPartnerCardReadiness{
		Ready:          readiness.Ready,
		CompletedCount: readiness.CompletedCount,
		TotalCount:     readiness.TotalCount,
		Items:          make([]*gqlmodel.EDIPartnerReadinessItem, 0, len(readiness.Items)),
	}
	for _, item := range readiness.Items {
		model.Items = append(model.Items, &gqlmodel.EDIPartnerReadinessItem{
			Key:      item.Key,
			Label:    item.Label,
			Complete: item.Complete,
		})
	}

	return model
}

func ediPartnerMappingsToModel(
	summary *ediports.EDIMappingSummary,
) *gqlmodel.EDIPartnerMappingSummary {
	model := &gqlmodel.EDIPartnerMappingSummary{
		Counts: make([]*gqlmodel.EDIMappingCount, 0, len(summary.Counts)),
	}
	if summary.ProfileID.IsNotNil() {
		model.ProfileID = new(summary.ProfileID.String())
	}
	if summary.UpdatedAt > 0 {
		model.UpdatedAt = new(int(summary.UpdatedAt))
	}
	for _, count := range summary.Counts {
		model.Counts = append(model.Counts, &gqlmodel.EDIMappingCount{
			EntityType: count.EntityType,
			Count:      int(count.Count),
		})
	}

	return model
}

func ediPartnerActivityToModel(
	activity *ediservice.EDIPartnerActivity,
) *gqlmodel.EDIPartnerActivity {
	model := &gqlmodel.EDIPartnerActivity{
		Since:           int(activity.Since),
		SeriesSince:     int(activity.SeriesSince),
		Scorecard:       ediPartnerScorecardsToModel([]*ediports.EDIPartnerScorecardRow{activity.Scorecard})[0],
		AckStatusCounts: summaryStatusCounts(activity.AckCounts),
		Sets:            make([]*gqlmodel.EDIPartnerSetActivity, 0, len(activity.Sets)),
		Series: ediVolumeSeriesToModel(&ediservice.EDIVolumeSeries{
			BucketSeconds: activity.BucketSeconds,
			Points:        activity.Series,
		}),
	}
	model.Scorecard.OverdueAckCount = activity.OverdueAckCount
	for _, set := range activity.Sets {
		row := &gqlmodel.EDIPartnerSetActivity{
			TransactionSet:    string(set.TransactionSet),
			Direction:         set.Direction,
			Count:             int(set.Count),
			FailedCount:       int(set.FailedCount),
			DeadLetteredCount: int(set.DeadLetteredCount),
		}
		if set.LastAt > 0 {
			row.LastAt = new(int(set.LastAt))
		}
		model.Sets = append(model.Sets, row)
	}

	return model
}

func ediPartnerAttentionToModel(
	attention *ediservice.EDIPartnerAttention,
) *gqlmodel.EDIPartnerAttention {
	model := &gqlmodel.EDIPartnerAttention{
		OverdueAckAgeSeconds: int(attention.OverdueAckAge),
		OverdueAckCount:      attention.OverdueAckCount,
		OverdueAcks:          nonNilSlice(attention.OverdueAcks),
		DeadLettered:         nonNilSlice(attention.DeadLettered),
		QuarantinedFiles:     nonNilSlice(attention.Quarantined),
	}
	if attention.OldestOverdueAt > 0 {
		model.OldestOverdueAt = new(int(attention.OldestOverdueAt))
	}

	return model
}

func ediPartnerCarriersToModel(
	channels []*carrier.CarrierEDIChannel,
) []*gqlmodel.EDIPartnerCarrier {
	models := make([]*gqlmodel.EDIPartnerCarrier, 0, len(channels))
	for _, channel := range channels {
		if channel.Carrier == nil {
			continue
		}
		scac := channel.SCACOverride
		if scac == "" {
			scac = channel.Carrier.SCAC
		}
		models = append(models, &gqlmodel.EDIPartnerCarrier{
			CarrierID:   channel.CarrierID.String(),
			CarrierName: channel.Carrier.Name,
			CarrierCode: base.NonEmptyPtr(channel.Carrier.Code),
			SCAC:        base.NonEmptyPtr(scac),
			IsDefault:   channel.IsDefault,
		})
	}

	return models
}

func nonNilSlice[T any](values []T) []T {
	if values == nil {
		return []T{}
	}
	return values
}
