package catalog

import "github.com/emoss08/trenova/internal/core/domain/platformcatalog"

//nolint:exhaustive // only features that own GraphQL surface appear here
var graphQLSourceOwners = map[platformcatalog.FeatureKey][]platformcatalog.GraphQLSource{
	platformcatalog.FeatureCoreTMS: {
		"carrier.graphqls",
		"commodity.graphqls",
		"customer.graphqls",
		"hazardous_material.graphqls",
		"hazmat_segregation_rule.graphqls",
		"service_type.graphqls",
	},
	platformcatalog.FeatureAdministration: {
		"audit_log.graphqls",
		"custom_field_definition.graphqls",
		"email_profile.graphqls",
		"role.graphqls",
		"scim_group_role_mapping.graphqls",
		"table_configuration.graphqls",
		"tenant.graphqls",
		"user.graphqls",
	},
	platformcatalog.FeatureDispatch: {
		"dispatch_console.graphqls",
		"carrier_intelligence.graphqls",
		"detention.graphqls",
		"distance_override.graphqls",
		"distance_profile.graphqls",
		"hold_reason.graphqls",
		"location.graphqls",
		"location_category.graphqls",
		"order.graphqls",
		"recurring_shipment.graphqls",
		"routing_guide.graphqls",
		"service_failure.graphqls",
		"service_failure_reason_code.graphqls",
		"shipment.graphqls",
		"shipment_type.graphqls",
		"stored_mileage.graphqls",
		"tender.graphqls",
	},
	platformcatalog.FeatureBilling: {
		"accessorial_charge.graphqls",
		"billing_queue.graphqls",
		"billing_transfer.graphqls",
		"costing.graphqls",
		"customer_payment.graphqls",
		"formula_template.graphqls",
		"fuel_surcharge.graphqls",
		"invoice.graphqls",
		"invoice_adjustment.graphqls",
		"invoice_dispute.graphqls",
		"late_charge.graphqls",
		"rate.graphqls",
	},
	platformcatalog.FeatureAccounting: {
		"account_type.graphqls",
		"accounting_sync.graphqls",
		"accounts_receivable.graphqls",
		"fiscal_period.graphqls",
		"fiscal_year.graphqls",
		"ifta.graphqls",
		"journal_entry.graphqls",
		"journal_reversal.graphqls",
		"jurisdiction_rule.graphqls",
		"manual_journal.graphqls",
	},
	platformcatalog.FeatureFleetMaintenance: {
		"equipment_continuity.graphqls",
		"equipment_manufacturer.graphqls",
		"equipment_type.graphqls",
		"fleet_code.graphqls",
		"fuel_purchase.graphqls",
		"tractor.graphqls",
		"trailer.graphqls",
	},
	platformcatalog.FeatureSettlement: {
		"carrier_settlement.graphqls",
		"driver_settlement.graphqls",
	},
	platformcatalog.FeatureDriverPortal: {
		"driver_portal.graphqls",
	},
	platformcatalog.FeatureWorkforceCore: {
		"org_holiday.graphqls",
		"org_structure.graphqls",
		"worker.graphqls",
		"worker_checklist.graphqls",
		"worker_employment.graphqls",
		"worker_overview.graphqls",
	},
	platformcatalog.FeatureWorkforceCompliance: {
		"worker_credential.graphqls",
		"worker_dqf.graphqls",
		"worker_drug_alcohol.graphqls",
	},
	platformcatalog.FeatureWorkforceTimeOff: {
		"pto_policy.graphqls",
		"worker_leave.graphqls",
	},
	platformcatalog.FeatureWorkforceTimeTracking: {
		"scheduling.graphqls",
		"timesheet.graphqls",
	},
	platformcatalog.FeatureWorkforceTalent: {
		"performance_review.graphqls",
		"worker_training.graphqls",
	},
	platformcatalog.FeatureWorkforceSafety: {
		"fleet_safety.graphqls",
		"worker_injury.graphqls",
		"worker_safety.graphqls",
	},
	platformcatalog.FeatureWorkforceBenefits: {
		"benefits.graphqls",
	},
	platformcatalog.FeatureWorkforceSelfService: {
		"self_service.graphqls",
	},
	platformcatalog.FeatureDocumentManagement: {
		"capture.graphqls",
		"document_packet_rule.graphqls",
		"document_template.graphqls",
		"document_type.graphqls",
	},
	platformcatalog.FeatureAnalytics: {
		"report.graphqls",
	},
	platformcatalog.FeatureAPIKeys: {
		"api_key.graphqls",
	},
	platformcatalog.FeatureTableChangeAlerts: {
		"table_change_alert.graphqls",
	},
	platformcatalog.FeatureEDIIntegration: {
		"edi.graphqls",
	},
	platformcatalog.FeatureSamsaraIntegration: {
		"telematics.graphqls",
	},
	platformcatalog.FeatureAgentAutomation: {
		"agent.graphqls",
		"agentdefinition.graphqls",
		"agentpreview.graphqls",
		"agentquality.graphqls",
		"agentrunevent.graphqls",
		"agentsafety.graphqls",
		"agentscorecard.graphqls",
		"aiaudit.graphqls",
		"aifeedback.graphqls",
		"aiprovider.graphqls",
		"airetrieval.graphqls",
		"aitraining.graphqls",
		"aiusage.graphqls",
		"decisions.graphqls",
		"desk_memory.graphqls",
		"extractioneval.graphqls",
		"extractionrollout.graphqls",
		"extractionshadow.graphqls",
		"watchtower.graphqls",
		"briefing.graphqls",
		"inboundmessage.graphqls",
	},
}

//nolint:exhaustive // only features that override individual root fields appear here
var graphQLRootFieldOwners = map[platformcatalog.FeatureKey][]platformcatalog.GraphQLRootField{
	platformcatalog.FeatureSettlement: {
		{Operation: platformcatalog.GraphQLOperationQuery, Field: "settlementDisputes"},
		{Operation: platformcatalog.GraphQLOperationQuery, Field: "settlementDispute"},
		{Operation: platformcatalog.GraphQLOperationQuery, Field: "openSettlementDisputeCount"},
		{
			Operation: platformcatalog.GraphQLOperationMutation,
			Field:     "startSettlementDisputeReview",
		},
		{Operation: platformcatalog.GraphQLOperationMutation, Field: "resolveSettlementDispute"},
	},
	platformcatalog.FeatureWorkforceSelfService: {
		{Operation: platformcatalog.GraphQLOperationQuery, Field: "workerPortalStatus"},
		{Operation: platformcatalog.GraphQLOperationQuery, Field: "myPolicies"},
		{Operation: platformcatalog.GraphQLOperationQuery, Field: "myPolicyDocumentUrl"},
		{Operation: platformcatalog.GraphQLOperationQuery, Field: "myProfileChangeRequests"},
		{Operation: platformcatalog.GraphQLOperationMutation, Field: "inviteWorkerToPortal"},
		{Operation: platformcatalog.GraphQLOperationMutation, Field: "revokeWorkerPortalAccess"},
		{Operation: platformcatalog.GraphQLOperationQuery, Field: "myCredentials"},
		{Operation: platformcatalog.GraphQLOperationQuery, Field: "myComplianceProfile"},
		{Operation: platformcatalog.GraphQLOperationMutation, Field: "acknowledgeMyPolicy"},
		{Operation: platformcatalog.GraphQLOperationMutation, Field: "withdrawMyProfileChange"},
		{Operation: platformcatalog.GraphQLOperationMutation, Field: "updateMyContactInfo"},
	},
	platformcatalog.FeatureWorkforceTimeOff: {
		{Operation: platformcatalog.GraphQLOperationQuery, Field: "myPto"},
		{Operation: platformcatalog.GraphQLOperationQuery, Field: "myPtoBalances"},
		{Operation: platformcatalog.GraphQLOperationQuery, Field: "myLeave"},
		{Operation: platformcatalog.GraphQLOperationMutation, Field: "requestMyPto"},
		{Operation: platformcatalog.GraphQLOperationMutation, Field: "cancelMyPto"},
	},
	platformcatalog.FeatureWorkforceTimeTracking: {
		{Operation: platformcatalog.GraphQLOperationQuery, Field: "mySchedule"},
		{Operation: platformcatalog.GraphQLOperationQuery, Field: "myAvailability"},
		{Operation: platformcatalog.GraphQLOperationQuery, Field: "myShiftSwaps"},
		{Operation: platformcatalog.GraphQLOperationMutation, Field: "setMyAvailability"},
		{Operation: platformcatalog.GraphQLOperationMutation, Field: "proposeMyShiftSwap"},
		{Operation: platformcatalog.GraphQLOperationMutation, Field: "respondToMyShiftSwap"},
	},
}

func graphQLSourcesFor(key platformcatalog.FeatureKey) []platformcatalog.GraphQLSource {
	return graphQLSourceOwners[key]
}

func graphQLRootFieldsFor(key platformcatalog.FeatureKey) []platformcatalog.GraphQLRootField {
	return graphQLRootFieldOwners[key]
}
