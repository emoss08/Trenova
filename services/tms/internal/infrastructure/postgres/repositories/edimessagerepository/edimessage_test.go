package edimessagerepository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"
)

func newEDIMessageTestRepository(t *testing.T) (*repository, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	mock.MatchExpectationsInOrder(false)

	bunDB := bun.NewDB(db, pgdialect.New())
	t.Cleanup(func() {
		require.NoError(t, mock.ExpectationsWereMet())
	})

	return &repository{
		db: postgres.NewTestConnection(bunDB),
		l:  zap.NewNop(),
	}, mock
}

func TestGetServiceFailure214LifecycleMessageMatchesLifecycleIdentity(t *testing.T) {
	t.Parallel()

	repo, mock := newEDIMessageTestRepository(t)
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	serviceFailureID := pulid.MustNew("sf_")
	messageID := pulid.MustNew("edimsg_")
	partnerID := pulid.MustNew("edip_")
	profileID := pulid.MustNew("edidp_")

	mock.ExpectQuery(
		`SELECT .*FROM "edi_messages" AS "emsg".*transaction_set = .*direction = .*payload_snapshot->'shipmentStatus'->>'serviceFailureId' = .*payload_snapshot->'shipmentStatus'->'references'->>'serviceFailureId' = .*payload_snapshot->'shipmentStatus'->'references'->>'serviceFailure214Trigger' = .*organization_id = .*business_unit_id = .*ORDER BY "emsg"."generated_at" DESC LIMIT 1`,
	).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"organization_id",
			"business_unit_id",
			"edi_partner_id",
			"partner_document_profile_id",
			"direction",
			"standard",
			"transaction_set",
			"status",
			"x12_version",
			"validation_mode",
			"interchange_control_number",
			"group_control_number",
			"transaction_control_number",
			"segment_count",
			"raw_x12",
			"payload_snapshot",
			"generated_at",
			"version",
			"created_at",
			"updated_at",
		}).AddRow(
			messageID,
			tenantInfo.OrgID,
			tenantInfo.BuID,
			partnerID,
			profileID,
			edi.DocumentDirectionOutbound,
			edi.EDIStandardX12,
			edi.TransactionSet214,
			edi.MessageStatusGenerated,
			"004010",
			edi.ValidationModeStrict,
			"1",
			"1",
			"1",
			1,
			"ST*214~",
			`{"transactionSet":"214","shipmentStatus":{"serviceFailureId":"`+serviceFailureID.String()+`","references":{"serviceFailureId":"`+serviceFailureID.String()+`","serviceFailure214Trigger":"Reviewed"}}}`,
			1,
			0,
			1,
			1,
		))

	message, err := repo.GetServiceFailure214LifecycleMessage(
		t.Context(),
		repositories.GetServiceFailure214LifecycleMessageRequest{
			TenantInfo:       tenantInfo,
			ServiceFailureID: serviceFailureID,
			Trigger:          "Reviewed",
		},
	)

	require.NoError(t, err)
	require.Equal(t, messageID, message.ID)
	require.Equal(t, partnerID, message.EDIPartnerID)
	require.Equal(t, profileID, message.PartnerDocumentProfileID)
}

func TestGetServiceFailure214LifecycleMessageReturnsNotFoundForNonMatchingLifecycleIdentity(t *testing.T) {
	t.Parallel()

	repo, mock := newEDIMessageTestRepository(t)
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	serviceFailureID := pulid.MustNew("sf_")

	mock.ExpectQuery(
		`SELECT .*payload_snapshot->'shipmentStatus'->>'serviceFailureId' = .*payload_snapshot->'shipmentStatus'->'references'->>'serviceFailureId' = .*payload_snapshot->'shipmentStatus'->'references'->>'serviceFailure214Trigger' = .*organization_id = .*business_unit_id = .*LIMIT 1`,
	).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	message, err := repo.GetServiceFailure214LifecycleMessage(
		t.Context(),
		repositories.GetServiceFailure214LifecycleMessageRequest{
			TenantInfo:       tenantInfo,
			ServiceFailureID: serviceFailureID,
			Trigger:          "Resolved",
		},
	)

	require.Error(t, err)
	require.Nil(t, message)
}
