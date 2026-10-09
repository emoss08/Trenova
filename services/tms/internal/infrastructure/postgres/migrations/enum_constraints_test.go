package migrations

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/agentextension"
	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/domain/agentwait"
	"github.com/emoss08/trenova/internal/core/domain/aiaudit"
	"github.com/emoss08/trenova/internal/core/domain/aifeedback"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/domain/carriercapacity"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/deskcase"
	"github.com/emoss08/trenova/internal/core/domain/extractionrollout"
	"github.com/emoss08/trenova/internal/core/domain/extractionshadow"
	"github.com/emoss08/trenova/internal/core/domain/onboarding"
	"github.com/emoss08/trenova/internal/core/domain/shipmentbrief"
	"github.com/emoss08/trenova/internal/core/domain/shipmentsuggestion"
	"github.com/emoss08/trenova/internal/core/domain/subscription"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/stretchr/testify/require"
)

type enumConstraint struct {
	name   string
	values []string
}

func stringsOf[T ~string](values []T) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, string(value))
	}

	return out
}

/*
Every value a Go enum declares is one the column's CHECK constraint accepts.

A template added to the domain and forgotten here is a row the database
refuses. The seed for the intake desk hit exactly that: the check still listed
the original eight templates, so a fresh install could not seed at all, and an
administrator could not create any of the desk agents.

The constraint is read from the newest up migration that adds it, which is the
one a migrated database is left with.
*/
func TestEnumCheckConstraintsAcceptEveryDeclaredValue(t *testing.T) {
	t.Parallel()

	constraints := []enumConstraint{
		{
			name:   "ck_ai_retraining_cycles_status",
			values: stringsOf(aitraining.AllRetrainingStatuses()),
		},
		{
			name:   "ck_ai_retraining_cycles_trigger",
			values: stringsOf(aitraining.AllRetrainingTriggers()),
		},
		{
			name:   "ck_ai_retraining_cycles_skip_reason",
			values: stringsOf(aitraining.AllRetrainingSkipReasons()),
		},
		{
			name:   "ck_ai_providers_reasoning_effort",
			values: stringsOf(aiprovider.AllReasoningEfforts()),
		},
		{
			name:   "ck_ai_retraining_cycles_structured_output_mode",
			values: stringsOf(aiprovider.AllStructuredOutputModes()),
		},
		{
			name:   "ck_extraction_rollouts_halt_reason",
			values: stringsOf(extractionrollout.AllHaltReasons()),
		},
		{
			name:   "ck_extraction_rollout_assignments_arm",
			values: stringsOf(extractionrollout.AllArms()),
		},
		{
			name:   "ck_extraction_rollout_assignments_outcome",
			values: stringsOf(extractionrollout.AllOutcomes()),
		},
		{
			name:   "ck_extraction_shadow_results_status",
			values: stringsOf(extractionshadow.AllResultStatuses()),
		},
		{
			name:   "ck_extraction_shadow_results_verdict",
			values: stringsOf(extractionshadow.AllVerdicts()),
		},
		{
			name:   "ck_accounting_connections_status",
			values: stringsOf(accountingsync.AllConnectionStatuses()),
		},
		{
			name:   "ck_accounting_connections_last_error_category",
			values: stringsOf(accountingsync.AllErrorCategories()),
		},
		{
			name:   "ck_accounting_connections_setup_step",
			values: stringsOf(accountingsync.AllSetupSteps()),
		},
		{
			name:   "ck_accounting_reference_objects_kind",
			values: stringsOf(accountingsync.AllReferenceKinds()),
		},
		{
			name:   "ck_accounting_reference_objects_account_class",
			values: stringsOf(accountingsync.AllAccountClasses()),
		},
		{
			name:   "ck_accounting_connections_integration_type",
			values: stringsOf(accountingsync.AccountingSystems()),
		},
		{
			name:   "ck_accounting_app_credentials_integration_type",
			values: stringsOf(accountingsync.AccountingSystems()),
		},
		{
			name:   "ck_accounting_mappings_target_type",
			values: stringsOf(accountingsync.AllMappingTargetTypes()),
		},
		{
			name:   "ck_accounting_mappings_provider_kind",
			values: stringsOf(accountingsync.AllReferenceKinds()),
		},
		{
			name:   "ck_accounting_mappings_state",
			values: stringsOf(accountingsync.AllMappingStates()),
		},
		{
			name:   "ck_accounting_mappings_source",
			values: stringsOf(accountingsync.AllMappingSources()),
		},
		{
			name:   "ck_accounting_sync_records_object_type",
			values: stringsOf(accountingsync.AllSyncObjectTypes()),
		},
		{
			name:   "ck_accounting_sync_records_operation",
			values: stringsOf(accountingsync.AllSyncOperations()),
		},
		{
			name:   "ck_accounting_sync_records_source_event",
			values: stringsOf(accountingsync.AllSyncSourceEvents()),
		},
		{
			name:   "ck_accounting_sync_records_status",
			values: stringsOf(accountingsync.AllSyncStatuses()),
		},
		{
			name:   "ck_accounting_sync_records_error_category",
			values: stringsOf(accountingsync.AllSyncErrorCategories()),
		},
		{
			name:   "ck_accounting_sync_attempts_outcome",
			values: stringsOf(accountingsync.AllSyncAttemptOutcomes()),
		},
		{
			name:   "ck_accounting_sync_attempts_error_category",
			values: stringsOf(accountingsync.AllSyncErrorCategories()),
		},
		{
			name:   "ck_accounting_backfills_status",
			values: stringsOf(accountingsync.AllBackfillStatuses()),
		},
		{
			name:   "ck_accounting_connections_inbound_payment_policy",
			values: stringsOf(accountingsync.AllInboundPaymentPolicies()),
		},
		{
			name:   "ck_accounting_connections_changes_error_category",
			values: stringsOf(accountingsync.AllSyncErrorCategories()),
		},
		{
			name:   "ck_accounting_inbound_changes_kind",
			values: stringsOf(accountingsync.AllInboundChangeKinds()),
		},
		{
			name:   "ck_accounting_inbound_changes_status",
			values: stringsOf(accountingsync.AllInboundChangeStatuses()),
		},
		{
			name:   "ck_accounting_inbound_changes_reason",
			values: stringsOf(accountingsync.AllInboundChangeReasons()),
		},
		{
			name:   "ck_accounting_connections_drift_error_category",
			values: stringsOf(accountingsync.AllSyncErrorCategories()),
		},
		{
			name:   "ck_accounting_drift_findings_object_type",
			values: stringsOf(accountingsync.AllSyncObjectTypes()),
		},
		{
			name:   "ck_accounting_drift_findings_kind",
			values: stringsOf(accountingsync.AllDriftKinds()),
		},
		{
			name:   "ck_accounting_drift_findings_status",
			values: stringsOf(accountingsync.AllDriftStatuses()),
		},
		{
			name:   "ck_accounting_drift_findings_resolution",
			values: stringsOf(accountingsync.AllDriftResolutions()),
		},
		{
			name:   "ck_agent_definitions_template",
			values: stringsOf(agentdefinition.AllTemplates()),
		},
		{
			name:   "ck_agent_extensions_type",
			values: stringsOf(agentextension.AllTypes()),
		},
		{
			name:   "ck_agent_extensions_availability",
			values: stringsOf(agentextension.AllAvailabilities()),
		},
		{
			name:   "ck_agent_definitions_access_mode",
			values: stringsOf(agentdefinition.AllAccessModes()),
		},
		{
			name:   "ck_agent_definitions_data_access_ceiling",
			values: stringsOf(agentdefinition.AllDataAccessCeilings()),
		},
		{
			name:   "ck_assistant_artifacts_kind",
			values: stringsOf(assistantartifact.AllKinds()),
		},
		{
			name:   "ck_assistant_threads_origin",
			values: stringsOf(conversation.AllThreadOrigins()),
		},
		{
			name:   "ck_case_checklist_templates_kind",
			values: stringsOf(deskcase.AllChecklistKinds()),
		},
		{
			name:   "ck_assistant_threads_snooze",
			values: stringsOf(deskcase.AllSnoozeAnchors()),
		},
		{
			name:   "ck_assistant_messages_kind",
			values: stringsOf(conversation.AllMessageKinds()),
		},
		{
			name:   "ck_assistant_turns_origin",
			values: stringsOf(conversation.AllAssistantTurnOrigins()),
		},
		{
			name:   "ck_agent_waits_kind",
			values: stringsOf(agentwait.AllKinds()),
		},
		{
			name:   "ck_agent_waits_status",
			values: stringsOf(agentwait.AllStatuses()),
		},
		{
			name:   "ck_agent_eval_cases_source",
			values: stringsOf(agentquality.AllCaseSources()),
		},
		{
			name:   "ck_agent_eval_cases_status",
			values: stringsOf(agentquality.AllCaseStatuses()),
		},
		{
			name:   "chk_agent_evaluations_status",
			values: stringsOf(agent.AllEvaluationStatuses()),
		},
		{
			name:   "chk_agent_memories_kind",
			values: stringsOf(agent.AllMemoryKinds()),
		},
		{
			name:   "chk_agent_memories_source",
			values: stringsOf(agent.AllMemorySources()),
		},
		{
			name:   "chk_agent_memories_status",
			values: stringsOf(agent.AllMemoryStatuses()),
		},
		{
			name:   "chk_agent_memories_scope",
			values: stringsOf(agent.AllMemoryScopes()),
		},
		{
			name:   "chk_agent_memory_preferences_saving_mode",
			values: stringsOf(agent.AllMemorySavingModes()),
		},
		{
			name:   "ck_agent_reflections_subject_type",
			values: stringsOf(agent.AllReflectionSubjects()),
		},
		{
			name:   "ck_agent_reflections_status",
			values: stringsOf(agent.AllReflectionStatuses()),
		},
		{
			name:   "ck_agent_reflections_skip_reason",
			values: stringsOf(agent.AllReflectionSkips()),
		},
		{
			name:   "chk_agent_proposals_egress_class",
			values: stringsOf(agent.EgressClasses()),
		},
		{
			name:   "ck_ai_feedback_target_type",
			values: stringsOf(aifeedback.AllTargetTypes()),
		},
		{
			name:   "ck_ai_feedback_fingerprint_source",
			values: stringsOf(aifeedback.AllFingerprintSources()),
		},
		{
			name:   "ck_agent_suite_runs_status",
			values: stringsOf(agentquality.AllSuiteRunStatuses()),
		},
		{
			name:   "ck_agent_suite_runs_trigger",
			values: stringsOf(agentquality.AllSuiteRunTriggers()),
		},
		{
			name:   "ck_ai_providers_embedding_input_style",
			values: stringsOf(aiprovider.AllEmbeddingInputStyles()),
		},
		{
			name:   "ck_ai_providers_thinking_style",
			values: stringsOf(aiprovider.AllThinkingStyles()),
		},
		{
			name:   "ck_ai_providers_on_cap",
			values: stringsOf(aiprovider.AllCapActions()),
		},
		{
			name:   "ck_ai_index_entries_source_type",
			values: stringsOf(airetrieval.AllSourceTypes()),
		},
		{
			name:   "ck_ai_index_entries_status",
			values: stringsOf(airetrieval.AllIndexStatuses()),
		},
		{
			name:   "ck_ai_retrieval_settings_paused_reason",
			values: stringsOf(airetrieval.AllPauseReasons()),
		},
		{
			name:   "ck_ai_embeddings_source_type",
			values: stringsOf(airetrieval.AllSourceTypes()),
		},
		{
			name:   "ck_ai_catalog_embeddings_corpus",
			values: stringsOf(airetrieval.AllCatalogCorpora()),
		},
		{
			name:   "ck_ai_usage_records_feature",
			values: stringsOf(aiusage.AllFeatures()),
		},
		{
			name:   "ck_ai_usage_records_subject_type",
			values: stringsOf(aiusage.AllSubjectTypes()),
		},
		{
			name:   "ck_ai_usage_records_owner_kind",
			values: stringsOf(agent.AllRunOwnerKinds()),
		},
		{
			name:   "ck_agent_runs_parent_owner_kind",
			values: stringsOf(agent.AllRunOwnerKinds()),
		},
		{
			name:   "ck_ai_audit_events_kind",
			values: stringsOf(aiaudit.AllKinds()),
		},
		{
			name:   "ck_ai_audit_events_outcome",
			values: stringsOf(aiaudit.AllOutcomes()),
		},
		{
			name:   "ck_ai_audit_events_principal_type",
			values: stringsOf(aiaudit.AllPrincipalTypes()),
		},
		{
			name:   "ck_ai_audit_events_purpose",
			values: stringsOf(aiaudit.AllPurposes()),
		},
		{
			name:   "ck_ai_audit_events_owner_kind",
			values: stringsOf(agent.AllRunOwnerKinds()),
		},
		{
			name:   "ck_ai_audit_chain_heads_last_verification_status",
			values: stringsOf(aiaudit.AllVerificationStatuses()),
		},
		{
			name:   "ck_ai_audit_exports_format",
			values: stringsOf(aiaudit.AllExportFormats()),
		},
		{
			name:   "ck_ai_audit_exports_status",
			values: stringsOf(aiaudit.AllExportStatuses()),
		},
		{
			name:   "ck_ai_audit_projector_state_source",
			values: stringsOf(aiaudit.AllSources()),
		},
		{
			name:   "ck_capture_devices_architecture",
			values: stringsOf(capture.AllArchitectures()),
		},
		{
			name:   "ck_capture_devices_status",
			values: stringsOf(capture.AllDeviceStatuses()),
		},
		{
			name:   "ck_capture_pairings_architecture",
			values: stringsOf(capture.AllArchitectures()),
		},
		{
			name:   "ck_capture_pairings_status",
			values: stringsOf(capture.AllPairingStatuses()),
		},
		{
			name:   "ck_capture_profiles_status",
			values: stringsOf(capture.AllProfileStatuses()),
		},
		{
			name:   "ck_capture_profiles_pixel_type",
			values: stringsOf(capture.AllPixelTypes()),
		},
		{
			name:   "ck_capture_profiles_separator_strategies",
			values: stringsOf(capture.AllSeparatorStrategies()),
		},
		{
			name:   "ck_capture_requests_mode",
			values: stringsOf(capture.AllRequestModes()),
		},
		{
			name:   "ck_capture_requests_status",
			values: stringsOf(capture.AllRequestStatuses()),
		},
		{
			name:   "ck_capture_requests_failure_code",
			values: stringsOf(capture.AllRequestFailureCodes()),
		},
		{
			name:   "ck_capture_batches_source",
			values: stringsOf(capture.AllSources()),
		},
		{
			name:   "ck_capture_batches_status",
			values: stringsOf(capture.AllBatchStatuses()),
		},
		{
			name:   "ck_capture_pages_status",
			values: stringsOf(capture.AllPageStatuses()),
		},
		{
			name:   "ck_capture_items_status",
			values: stringsOf(capture.AllItemStatuses()),
		},
		{
			name:   "ck_capture_items_suggestion_source",
			values: stringsOf(capture.AllSuggestionSources()),
		},
		{
			name:   "ck_organization_subscriptions_status",
			values: stringsOf(subscription.AllStatuses()),
		},
		{
			name:   "ck_organization_onboarding_status",
			values: stringsOf(onboarding.AllStatuses()),
		},
		{
			name:   "ck_organization_onboarding_operation_type",
			values: stringsOf(tenant.AllOperationTypes()),
		},
		{
			name:   "ck_carrier_capacity_postings_rate_method",
			values: stringsOf(carriercapacity.RateMethods()),
		},
		{
			name:   "ck_carrier_capacity_postings_source",
			values: stringsOf(carriercapacity.Sources()),
		},
		{
			name:   "ck_shipment_suggestion_decisions_decision",
			values: stringsOf(shipmentsuggestion.Decisions()),
		},
		{
			name:   "ck_shipment_board_briefs_trigger",
			values: stringsOf(shipmentbrief.Triggers()),
		},
	}

	for _, constraint := range constraints {
		latest, err := LatestConstraintDefinition(constraint.name)
		require.NoError(t, err)

		for _, value := range constraint.values {
			require.Contains(t, latest, "'"+value+"'",
				"%s does not accept %q; add a migration that widens it", constraint.name, value)
		}
	}
}
