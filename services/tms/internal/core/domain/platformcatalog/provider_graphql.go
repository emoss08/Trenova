package platformcatalog

//nolint:exhaustive // only features that own GraphQL surface appear here
var graphQLSourceOwners = map[FeatureKey][]GraphQLSource{
	FeatureCoreTMS: {
		"carrier.graphqls",
		"commodity.graphqls",
		"customer.graphqls",
		"hazardous_material.graphqls",
		"hazmat_segregation_rule.graphqls",
		"service_type.graphqls",
	},
	FeatureAdministration: {
		"audit_log.graphqls",
		"custom_field_definition.graphqls",
		"email_profile.graphqls",
		"role.graphqls",
		"scim_group_role_mapping.graphqls",
		"table_configuration.graphqls",
		"tenant.graphqls",
		"user.graphqls",
	},
	FeatureDispatch: {
		"dispatch_console.graphqls",
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
	FeatureBilling: {
		"accessorial_charge.graphqls",
		"billing_queue.graphqls",
		"costing.graphqls",
		"customer_payment.graphqls",
		"formula_template.graphqls",
		"fuel_surcharge.graphqls",
		"invoice.graphqls",
		"rate.graphqls",
	},
	FeatureAccounting: {
		"account_type.graphqls",
		"accounts_receivable.graphqls",
		"fiscal_period.graphqls",
		"fiscal_year.graphqls",
		"ifta.graphqls",
		"journal_entry.graphqls",
		"journal_reversal.graphqls",
		"jurisdiction_rule.graphqls",
		"manual_journal.graphqls",
	},
	FeatureFleetMaintenance: {
		"equipment_continuity.graphqls",
		"equipment_manufacturer.graphqls",
		"equipment_type.graphqls",
		"fleet_code.graphqls",
		"fuel_purchase.graphqls",
		"tractor.graphqls",
		"trailer.graphqls",
	},
	FeatureSettlement: {
		"carrier_settlement.graphqls",
		"driver_settlement.graphqls",
	},
	FeatureDriverPortal: {
		"driver_portal.graphqls",
	},
	FeatureWorkforceCore: {
		"org_holiday.graphqls",
		"org_structure.graphqls",
		"worker.graphqls",
		"worker_checklist.graphqls",
		"worker_employment.graphqls",
		"worker_overview.graphqls",
	},
	FeatureWorkforceCompliance: {
		"worker_credential.graphqls",
		"worker_dqf.graphqls",
		"worker_drug_alcohol.graphqls",
	},
	FeatureWorkforceTimeOff: {
		"pto_policy.graphqls",
		"worker_leave.graphqls",
	},
	FeatureWorkforceTimeTracking: {
		"scheduling.graphqls",
		"timesheet.graphqls",
	},
	FeatureWorkforceTalent: {
		"performance_review.graphqls",
		"worker_training.graphqls",
	},
	FeatureWorkforceSafety: {
		"fleet_safety.graphqls",
		"worker_injury.graphqls",
		"worker_safety.graphqls",
	},
	FeatureWorkforceBenefits: {
		"benefits.graphqls",
	},
	FeatureWorkforceSelfService: {
		"self_service.graphqls",
	},
	FeatureDocumentManagement: {
		"document_packet_rule.graphqls",
		"document_template.graphqls",
		"document_type.graphqls",
	},
	FeatureAnalytics: {
		"report.graphqls",
	},
	FeatureAPIKeys: {
		"api_key.graphqls",
	},
	FeatureTableChangeAlerts: {
		"table_change_alert.graphqls",
	},
	FeatureEDIIntegration: {
		"edi.graphqls",
	},
	FeatureSamsaraIntegration: {
		"telematics.graphqls",
	},
	FeatureAgentAutomation: {
		"agent.graphqls",
	},
}

//nolint:exhaustive // only features that override individual root fields appear here
var graphQLRootFieldOwners = map[FeatureKey][]GraphQLRootField{
	FeatureSettlement: {
		{Operation: GraphQLOperationQuery, Field: "settlementDisputes"},
		{Operation: GraphQLOperationQuery, Field: "settlementDispute"},
		{Operation: GraphQLOperationQuery, Field: "openSettlementDisputeCount"},
		{Operation: GraphQLOperationMutation, Field: "startSettlementDisputeReview"},
		{Operation: GraphQLOperationMutation, Field: "resolveSettlementDispute"},
	},
	FeatureWorkforceSelfService: {
		{Operation: GraphQLOperationQuery, Field: "workerPortalStatus"},
		{Operation: GraphQLOperationQuery, Field: "myPolicies"},
		{Operation: GraphQLOperationQuery, Field: "myPolicyDocumentUrl"},
		{Operation: GraphQLOperationQuery, Field: "myProfileChangeRequests"},
		{Operation: GraphQLOperationMutation, Field: "inviteWorkerToPortal"},
		{Operation: GraphQLOperationMutation, Field: "revokeWorkerPortalAccess"},
		{Operation: GraphQLOperationQuery, Field: "myCredentials"},
		{Operation: GraphQLOperationQuery, Field: "myComplianceProfile"},
		{Operation: GraphQLOperationMutation, Field: "acknowledgeMyPolicy"},
		{Operation: GraphQLOperationMutation, Field: "withdrawMyProfileChange"},
		{Operation: GraphQLOperationMutation, Field: "updateMyContactInfo"},
	},
	FeatureWorkforceTimeOff: {
		{Operation: GraphQLOperationQuery, Field: "myPto"},
		{Operation: GraphQLOperationQuery, Field: "myPtoBalances"},
		{Operation: GraphQLOperationQuery, Field: "myLeave"},
		{Operation: GraphQLOperationMutation, Field: "requestMyPto"},
		{Operation: GraphQLOperationMutation, Field: "cancelMyPto"},
	},
	FeatureWorkforceTimeTracking: {
		{Operation: GraphQLOperationQuery, Field: "mySchedule"},
		{Operation: GraphQLOperationQuery, Field: "myAvailability"},
		{Operation: GraphQLOperationQuery, Field: "myShiftSwaps"},
		{Operation: GraphQLOperationMutation, Field: "setMyAvailability"},
		{Operation: GraphQLOperationMutation, Field: "proposeMyShiftSwap"},
		{Operation: GraphQLOperationMutation, Field: "respondToMyShiftSwap"},
	},
}

func graphQLSourcesFor(key FeatureKey) []GraphQLSource {
	return graphQLSourceOwners[key]
}

func graphQLRootFieldsFor(key FeatureKey) []GraphQLRootField {
	return graphQLRootFieldOwners[key]
}
