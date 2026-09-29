package toolcontract

type Binding struct {
	Tool      string
	Input     string
	Param     string
	Nested    map[string]string
	Defaulted map[string]string
	Extra     map[string]string
	Narrowed  map[string]string
}

const (
	patchLeavesIt = "a patch leaves the saved value alone, so the tool asks only for what changes"
	shipmentIDArg = "the mutation takes the shipment as its own argument, beside the input"
	versionField  = "version"
	readVersion   = "the tool sends the version of the record it reads, which the approved " +
		"preview pins"
)

func namedID(param string) string {
	return "one id under two names: the input's, and " + param + ", the name the read " +
		"tools hand it out under and the tool takes"
}

var Bindings = []Binding{
	{
		Tool:  "create_shipment",
		Input: "ShipmentInput",
		Param: "shipment",
		Nested: map[string]string{
			"moves":             "ShipmentMoveInput",
			"moves.stops":       "ShipmentStopInput",
			"commodities":       "ShipmentCommodityInput",
			"additionalCharges": "ShipmentAdditionalChargeInput",
		},
	},
	{
		Tool:  "update_shipment",
		Input: "ShipmentInput",
		Defaulted: map[string]string{
			"serviceTypeId":     patchLeavesIt,
			"shipmentTypeId":    patchLeavesIt,
			"customerId":        patchLeavesIt,
			"formulaTemplateId": patchLeavesIt,
		},
		Extra: map[string]string{
			"shipmentId": "the mutation takes the shipment as its id argument, beside the input",
		},
	},
	{
		Tool:  "duplicate_shipment",
		Input: "ShipmentDuplicateInput",
		Extra: map[string]string{
			"firstPickupAt": "the service and the REST duplicate route take a new first " +
				"pickup; the GraphQL input offers only overrideDates, which re-anchors to today",
		},
	},
	{
		Tool:  "add_shipment_comment",
		Input: "ShipmentCommentInput",
		Extra: map[string]string{"shipmentId": shipmentIDArg},
	},
	{
		Tool:  "correct_charge_code",
		Input: "ShipmentAdditionalChargeInput",
		Param: "additionalCharges",
	},
	{
		Tool:      "attach_worker_credential_document",
		Input:     "AttachWorkerCredentialDocumentInput",
		Defaulted: map[string]string{"id": namedID("credentialId")},
		Extra:     map[string]string{"credentialId": namedID("credentialId")},
	},
	{
		Tool:      "attach_worker_training_document",
		Input:     "AttachWorkerTrainingDocumentInput",
		Defaulted: map[string]string{"id": namedID("trainingRecordId")},
		Extra:     map[string]string{"trainingRecordId": namedID("trainingRecordId")},
	},
	{
		Tool:   "draft_performance_review",
		Input:  "UpdatePerformanceReviewInput",
		Nested: map[string]string{"ratings": "ReviewRatingInput", "goals": "ReviewGoalInput"},
		Defaulted: map[string]string{
			"id": namedID("reviewId"),
			"ratings": "the tool merges the scores it is given into the review's current " +
				"ratings and sends every item, so an item left out keeps what it has",
			versionField: readVersion,
		},
		Extra: map[string]string{"reviewId": namedID("reviewId")},
	},
	{
		Tool:  "give_worker_recognition",
		Input: "WorkerRecognitionInput",
	},
	{
		Tool:  "open_leave_case",
		Input: "OpenLeaveCaseInput",
	},
	{
		Tool:  "open_worker_safety_event",
		Input: "WorkerSafetyEventInput",
	},
	{
		Tool:  "record_employment_verification",
		Input: "RecordEmploymentVerificationInput",
	},
	{
		Tool:      "record_leave_day",
		Input:     "RecordLeaveDayInput",
		Defaulted: map[string]string{"caseId": namedID("leaveCaseId")},
		Extra:     map[string]string{"leaveCaseId": namedID("leaveCaseId")},
	},
	{
		Tool:  "record_safety_violation",
		Input: "RecordSafetyViolationInput",
	},
	{
		Tool:  "record_training_completion",
		Input: "CompleteWorkerTrainingInput",
		Extra: map[string]string{"trainingRecordId": namedID("trainingRecordId")},
	},
	{
		Tool:  "record_worker_credential",
		Input: "WorkerCredentialInput",
	},
	{
		Tool:  "record_worker_injury",
		Input: "RecordWorkerInjuryInput",
	},
	{
		Tool:  "request_worker_pto",
		Input: "CreateWorkerPTOInput",
	},
	{
		Tool:  "schedule_dot_test",
		Input: "RecordDOTTestInput",
	},
	{
		Tool:  "start_performance_review",
		Input: "CreatePerformanceReviewInput",
	},
	{
		Tool:  "start_worker_checklist",
		Input: "StartWorkerChecklistInput",
	},
	{
		Tool:  "update_employment_verification",
		Input: "UpdateEmploymentVerificationInput",
	},
	{
		Tool:      "update_leave_case",
		Input:     "UpdateLeaveCaseInput",
		Defaulted: map[string]string{"caseId": namedID("leaveCaseId")},
		Extra:     map[string]string{"leaveCaseId": namedID("leaveCaseId")},
	},
	{
		Tool:      "update_leave_day",
		Input:     "UpdateLeaveDayInput",
		Defaulted: map[string]string{"entryId": namedID("leaveDayId")},
		Extra:     map[string]string{"leaveDayId": namedID("leaveDayId")},
	},
	{
		Tool:  "update_safety_violation",
		Input: "UpdateSafetyViolationInput",
		Defaulted: map[string]string{
			"id":          namedID("violationId"),
			"description": patchLeavesIt,
		},
		Extra: map[string]string{"violationId": namedID("violationId")},
	},
	{
		Tool:      "update_worker_checklist_item",
		Input:     "WorkerChecklistItemActionInput",
		Defaulted: map[string]string{"id": namedID("checklistItemId")},
		Extra: map[string]string{
			"checklistItemId": namedID("checklistItemId"),
			"action": "picks which item mutation runs: complete, skip and not applicable " +
				"take this input, and reopen takes the item's id",
		},
	},
	{
		Tool:  "update_worker_credential",
		Input: "UpdateWorkerCredentialInput",
		Defaulted: map[string]string{
			"id":         namedID("credentialId"),
			versionField: readVersion,
		},
		Extra: map[string]string{"credentialId": namedID("credentialId")},
	},
	{
		Tool:  "update_worker_injury",
		Input: "UpdateWorkerInjuryInput",
	},
	{
		Tool:  "update_worker_pto",
		Input: "UpdateWorkerPTOInput",
		Defaulted: map[string]string{
			"id":         namedID("ptoId"),
			versionField: readVersion,
			"type":       patchLeavesIt,
			"startDate":  patchLeavesIt,
			"endDate":    patchLeavesIt,
			"reason":     patchLeavesIt,
		},
		Extra: map[string]string{"ptoId": namedID("ptoId")},
	},
	{
		Tool:  "update_worker_safety_event",
		Input: "UpdateWorkerSafetyEventInput",
		Defaulted: map[string]string{
			"id":          namedID("safetyEventId"),
			versionField:  readVersion,
			"kind":        patchLeavesIt,
			"severity":    patchLeavesIt,
			"occurredAt":  patchLeavesIt,
			"description": patchLeavesIt,
			"points":      patchLeavesIt,
		},
		Extra: map[string]string{"safetyEventId": namedID("safetyEventId")},
	},
	{
		Tool:   "create_invoice_memo",
		Input:  "CreateMemoInput",
		Nested: map[string]string{"lines": "MemoLineInput"},
		Narrowed: map[string]string{
			"billType": "a memo is a credit or a debit; an invoice is issued by create_invoice",
		},
	},
}

const (
	statusDecision = "one of several decisions a single status mutation takes: the tool " +
		"fixes the status the input leaves open, and names the item it decides on"
	layoutPart = "changes one part of a layout the input replaces whole, so the input's " +
		"required fields are the layout the tool reads, not what the model sends"
	pendingBinding = "pending: written before the contract, with parameters named for the " +
		"model where the input names them differently; binding it is the write waves' work"
)

var Unbound = map[string]string{
	"add_carrier_settlement_adjustment": pendingBinding,
	"add_dashboard_tile":                layoutPart,
	"add_driver_settlement_adjustment":  pendingBinding,
	"add_home_widget":                   layoutPart,
	"add_order_charge": "addOrderCharge takes the charge as scalar arguments; only its payer sp" +
		"lit is an input object, which the tool does not take",
	"adjust_escrow_account":                pendingBinding,
	"apply_carrier_intel_suggestions":      pendingBinding,
	"apply_credit_memo":                    pendingBinding,
	"apply_customer_payment":               pendingBinding,
	"approve_billing_queue_item":           statusDecision,
	"arrange_home_layout":                  layoutPart,
	"assess_late_charges":                  pendingBinding,
	"assign_move":                          pendingBinding,
	"attach_pay_events_to_settlement":      pendingBinding,
	"cancel_billing_queue_item":            statusDecision,
	"confirm_accounting_mapping_proposals": pendingBinding,
	"create_accounting_reference_record":   pendingBinding,
	"create_carrier_invoice_match":         pendingBinding,
	"create_dashboard":                     pendingBinding,
	"create_order":                         pendingBinding,
	"create_recurring_deduction":           pendingBinding,
	"create_recurring_earning":             pendingBinding,
	"create_report":                        pendingBinding,
	"detach_pay_event_from_settlement":     pendingBinding,
	"dismiss_accounting_drift":             pendingBinding,
	"dispute_detention":                    pendingBinding,
	"edit_shipment_comment":                pendingBinding,
	"fork_report":                          pendingBinding,
	"generate_carrier_settlement_batch":    pendingBinding,
	"generate_driver_settlement":           pendingBinding,
	"generate_driver_settlement_batch":     pendingBinding,
	"hold_billing_queue_item":              statusDecision,
	"hold_driver_pay_event":                pendingBinding,
	"ignore_accounting_inbound_change":     pendingBinding,
	"issue_pay_advance":                    pendingBinding,
	"link_inbound_message":                 pendingBinding,
	"mark_inbound_message":                 pendingBinding,
	"move_billing_item_to_exception":       statusDecision,
	"open_escrow_account":                  pendingBinding,
	"open_invoice_dispute":                 pendingBinding,
	"pause_accounting_sync":                pendingBinding,
	"post_customer_payment":                pendingBinding,
	"propose_shift_swap":                   pendingBinding,
	"recalculate_carrier_settlement":       pendingBinding,
	"recalculate_driver_settlement":        pendingBinding,
	"record_carrier_settlement_payment":    pendingBinding,
	"record_driver_settlement_payment":     pendingBinding,
	"release_accounting_sync":              pendingBinding,
	"remember":                             pendingBinding,
	"remove_carrier_settlement_adjustment": pendingBinding,
	"remove_driver_settlement_adjustment":  pendingBinding,
	"remove_home_widget":                   layoutPart,
	"remove_order_charge":                  pendingBinding,
	"resolve_accounting_drift":             pendingBinding,
	"resolve_carrier_intel_event":          pendingBinding,
	"retry_accounting_sync":                pendingBinding,
	"reverse_customer_payment":             pendingBinding,
	"save_table_view":                      pendingBinding,
	"schedule_report":                      pendingBinding,
	"send_billing_item_back_to_ops":        statusDecision,
	"set_accounting_mapping":               pendingBinding,
	"set_order_charge_allocations":         pendingBinding,
	"set_worker_availability_preference":   pendingBinding,
	"skip_accounting_sync":                 pendingBinding,
	"transfer_shipment_ownership":          pendingBinding,
	"transfer_to_billing": "one tool for four mutations (one shipment, its items, a bulk transfer " +
		"and a transfer run), so no single input is its shape",
	"transition_item_to_in_review": statusDecision,
	"unapply_credit_memo":          pendingBinding,
	"update_escrow_account":        pendingBinding,
	"update_order":                 pendingBinding,
	"update_order_charge":          pendingBinding,
	"update_recurring_deduction":   pendingBinding,
	"update_recurring_earning":     pendingBinding,
	"update_report":                pendingBinding,
	"update_tractor_status":        pendingBinding,
	"update_trailer_status":        pendingBinding,
	"verify_carrier_equipment":     pendingBinding,
	"waive_detention":              pendingBinding,
}
