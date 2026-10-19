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
	watchtowerItemID       = "watchtowerItemId"
	reserveMovementInput   = "FactoringReserveMovementInput"
	remittanceVersionInput = "FactoringRemittanceVersionInput"
	exceptionItemArg       = "the mutation takes the exception as its id argument, beside the input"
	resolutionCode         = "one value under two names: the input's code, and resolutionCode, " +
		"the name the read tools hand it out under and the tool takes"
)

const (
	inboundMessageID   = "messageId"
	inboundMessageArg  = "the mutation takes the message as its own argument, beside the input"
	triageMessageIDs   = "messageIds"
	triageSelectionIDs = "one list under two names: the selection's ids, and messageIds, the " +
		"name the inbox read tools hand them out under; the tool never takes a folder"
)

const (
	watchtowerItemIDParam = "watchtowerItemId"
	noaIDParam            = "noaId"
	exceptionIDArg        = "the mutation takes the exception as its id argument, beside the input"
)

func stateCode(field string) string {
	return "the tool takes the two-letter state code and resolves it to the " + field +
		" the input takes; no read tool hands out state ids"
}

func decimalOf(field string) string {
	return "the tool takes the amount as a decimal string and converts it to the " + field +
		" the input takes in minor units"
}

func namedID(param string) string {
	return "one id under two names: the input's, and " + param + ", the name the read " +
		"tools hand it out under and the tool takes"
}

var Bindings = []Binding{
	{Tool: "draft_factoring_noa", Input: "DraftFactoringNoaInput"},
	{
		Tool:      "send_factoring_noa",
		Input:     "SendFactoringNoaInput",
		Defaulted: map[string]string{"id": namedID(noaIDParam)},
		Extra:     map[string]string{noaIDParam: namedID(noaIDParam)},
	},
	{
		Tool:      "record_factoring_noa_acknowledgement",
		Input:     "AcknowledgeFactoringNoaInput",
		Defaulted: map[string]string{"id": namedID(noaIDParam)},
		Extra:     map[string]string{noaIDParam: namedID(noaIDParam)},
	},
	{
		Tool:      "release_factoring_noa",
		Input:     "ReleaseFactoringNoaInput",
		Defaulted: map[string]string{"id": namedID(noaIDParam)},
		Extra:     map[string]string{noaIDParam: namedID(noaIDParam)},
	},
	{Tool: "set_invoice_factoring_override", Input: "SetInvoiceFactoringOverrideInput"},
	{
		Tool:  "retry_factoring_submission",
		Input: "RetryFactoringSubmissionInput",
		Defaulted: map[string]string{
			"id":         namedID("submissionId"),
			versionField: readVersion,
		},
		Extra: map[string]string{"submissionId": namedID("submissionId")},
	},
	{Tool: "request_missing_factoring_document", Input: "RequestMissingFactoringDocumentsInput"},
	{
		Tool:   "record_factoring_funding",
		Input:  "RecordFactoringFundingInput",
		Nested: map[string]string{"items": "FactoringFundingItemInput"},
		Extra: map[string]string{
			"items.advance": decimalOf("advanceMinor"),
			"items.fee":     decimalOf("feeMinor"),
		},
	},
	{
		Tool:  "release_factoring_reserve",
		Input: reserveMovementInput,
		Extra: map[string]string{"cash": decimalOf("cashMinor")},
	},
	{
		Tool:  "record_factoring_chargeback",
		Input: reserveMovementInput,
		Extra: map[string]string{"cash": decimalOf("cashMinor")},
	},
	{Tool: "reverse_factoring_chargeback", Input: reserveMovementInput},
	{Tool: "resolve_factoring_remittance_line", Input: "UpdateFactoringRemittanceLineInput"},
	{Tool: "rematch_factoring_remittance", Input: remittanceVersionInput},
	{Tool: "apply_factoring_remittance", Input: remittanceVersionInput},
	{Tool: "discard_factoring_remittance", Input: "DiscardFactoringRemittanceInput"},
	{
		Tool:  "assign_exception",
		Input: "AssignWatchtowerItemInput",
		Extra: map[string]string{watchtowerItemID: exceptionItemArg},
	},
	{
		Tool:  "snooze_exception",
		Input: "SnoozeWatchtowerItemInput",
		Extra: map[string]string{watchtowerItemID: exceptionItemArg},
	},
	{
		Tool:  "set_exception_due",
		Input: "SetWatchtowerItemDueInput",
		Extra: map[string]string{watchtowerItemID: exceptionItemArg},
	},
	{
		Tool:      "resolve_exception",
		Input:     "ResolveWatchtowerItemInput",
		Defaulted: map[string]string{"code": resolutionCode},
		Extra: map[string]string{
			watchtowerItemID: exceptionItemArg,
			"resolutionCode": resolutionCode,
		},
	},
	{
		Tool:  "reopen_exception",
		Input: "ReopenWatchtowerItemInput",
		Extra: map[string]string{watchtowerItemID: exceptionItemArg},
	},
	{
		Tool:  "shipment_create",
		Input: "ShipmentInput",
		Param: "shipment",
		Nested: map[string]string{
			"moves":             "ShipmentMoveInput",
			"moves.stops":       "ShipmentStopInput",
			"commodities":       "ShipmentCommodityInput",
			"additionalCharges": "ShipmentAdditionalChargeInput",
		},
		Defaulted: map[string]string{
			"formulaTemplateId": "left out, the tool seats the rating method and base rate the " +
				"customer's rate agreement prices the lane with, and refuses when none does",
		},
	},
	{
		Tool:  "shipment_update",
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
		Tool:  "shipment_duplicate",
		Input: "ShipmentDuplicateInput",
		Extra: map[string]string{
			"firstPickupAt": "the service and the REST duplicate route take a new first " +
				"pickup; the GraphQL input offers only overrideDates, which re-anchors to today",
		},
	},
	{
		Tool:      "log_check_call",
		Input:     "LogCheckCallInput",
		Defaulted: map[string]string{"shipmentMoveId": namedID("moveId")},
		Extra:     map[string]string{"moveId": namedID("moveId")},
	},
	{
		Tool:  "assign_exception",
		Input: "AssignWatchtowerItemInput",
		Extra: map[string]string{watchtowerItemIDParam: exceptionIDArg},
	},
	{
		Tool:  "snooze_exception",
		Input: "SnoozeWatchtowerItemInput",
		Extra: map[string]string{watchtowerItemIDParam: exceptionIDArg},
	},
	{
		Tool:  "set_exception_due",
		Input: "SetWatchtowerItemDueInput",
		Extra: map[string]string{watchtowerItemIDParam: exceptionIDArg},
	},
	{
		Tool:  "resolve_exception",
		Input: "ResolveWatchtowerItemInput",
		Defaulted: map[string]string{
			"code": "one value under two names: the input's code, and resolutionCode, " +
				"which the tool takes so it is not mistaken for a shipment or charge code",
		},
		Extra: map[string]string{
			watchtowerItemIDParam: exceptionIDArg,
			"resolutionCode": "one value under two names: the input's code, and " +
				"resolutionCode, which the tool takes",
		},
	},
	{
		Tool:  "reopen_exception",
		Input: "ReopenWatchtowerItemInput",
		Extra: map[string]string{watchtowerItemIDParam: exceptionIDArg},
	},
	{
		Tool:  "shipment_share_tracking_link",
		Input: "IssueCustomerTrackingLinkInput",
	},
	{
		Tool:      "issue_carrier_tracking_link",
		Input:     "IssueCarrierTrackingLinkInput",
		Defaulted: map[string]string{"shipmentMoveId": namedID("moveId")},
		Extra:     map[string]string{"moveId": namedID("moveId")},
	},
	{
		Tool:  "shipment_add_comment",
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
		Defaulted: map[string]string{
			"reason": "left out, the kind of time off is the reason: \"put it in as vacation\" " +
				"says why, and asking for a reason only delayed the request",
		},
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
	{Tool: "record_fuel_purchase", Input: "FuelPurchaseInput", Defaulted: map[string]string{
		"totalAmount": "the tool works the total out from quantity and unitPrice when a receipt " +
			"is read out as gallons and a price, and refuses when it has neither",
	}},
	{
		Tool:  "correct_fuel_purchase",
		Input: "FuelPurchaseInput",
		Defaulted: map[string]string{
			"tractorId":      patchLeavesIt,
			"jurisdictionId": patchLeavesIt,
			"purchasedAt":    patchLeavesIt,
			"fuelType":       patchLeavesIt,
			"quantity":       patchLeavesIt,
			"totalAmount":    patchLeavesIt,
		},
		Extra: map[string]string{"fuelPurchaseId": namedID("fuelPurchaseId")},
	},
	{
		Tool:  "assign_fuel_card",
		Input: "AssignFuelCardInput",
		Defaulted: map[string]string{
			"id":         namedID("fuelCardId"),
			versionField: readVersion,
		},
		Extra: map[string]string{"fuelCardId": namedID("fuelCardId")},
	},
	{Tool: "record_fuel_index_price", Input: "FuelIndexPriceInput"},
	{
		Tool:      "correct_fuel_index_price",
		Input:     "UpdateFuelIndexPriceInput",
		Defaulted: map[string]string{"id": namedID("fuelIndexPriceId")},
		Extra:     map[string]string{"fuelIndexPriceId": namedID("fuelIndexPriceId")},
	},
	{Tool: "record_ifta_mileage_entry", Input: "IFTAMileageEntryInput"},
	{
		Tool:  "correct_ifta_mileage_entry",
		Input: "IFTAMileageEntryInput",
		Defaulted: map[string]string{
			"tractorId":      patchLeavesIt,
			"jurisdictionId": patchLeavesIt,
			"traveledAt":     patchLeavesIt,
			"miles":          patchLeavesIt,
		},
		Extra: map[string]string{"iftaMileageEntryId": namedID("iftaMileageEntryId")},
	},
	{Tool: "generate_ifta_return", Input: "IFTAPeriodInput"},
	{
		Tool:  "create_carrier_capacity_posting",
		Input: "CarrierCapacityPostingInput",
		Extra: map[string]string{
			"originState":      stateCode("originStateId"),
			"destinationState": stateCode("destinationStateId"),
		},
	},
	{
		Tool:  "update_carrier_capacity_posting",
		Input: "CarrierCapacityPostingInput",
		Defaulted: map[string]string{
			"carrierId":     patchLeavesIt,
			"availableFrom": patchLeavesIt,
			"availableTo":   patchLeavesIt,
		},
		Extra: map[string]string{
			"carrierCapacityPostingId": "the mutation takes the posting as its id argument, " +
				"beside the input",
			"originState":      stateCode("originStateId"),
			"destinationState": stateCode("destinationStateId"),
		},
	},
	{
		Tool:  "create_tractor",
		Input: "TractorInput",
		Extra: map[string]string{"state": stateCode("stateId")},
	},
	{
		Tool:  "update_tractor",
		Input: "TractorPatchInput",
		Extra: map[string]string{
			"tractorId": "the mutation takes the tractor as its id argument, beside the input",
			"state":     stateCode("stateId"),
		},
	},
	{Tool: "locate_tractor", Input: "LocateTractorInput"},
	{
		Tool:  "create_trailer",
		Input: "TrailerInput",
		Extra: map[string]string{"registrationState": stateCode("registrationStateId")},
	},
	{
		Tool:  "update_trailer",
		Input: "TrailerPatchInput",
		Extra: map[string]string{
			"trailerId":         "the mutation takes the trailer as its id argument, beside the input",
			"registrationState": stateCode("registrationStateId"),
		},
	},
	{Tool: "locate_trailer", Input: "LocateTrailerInput"},
	{
		Tool:      "file_capture_items",
		Input:     "FileCaptureItemsEntryInput",
		Param:     "items",
		Defaulted: map[string]string{"version": readVersion},
	},
	{
		Tool:  "assign_inbound_messages",
		Input: "TriageSelectionInput",
		Extra: map[string]string{
			triageMessageIDs: triageSelectionIDs,
			"ownerId": "the mutation takes the owner as its own argument, beside the " +
				"selection",
		},
	},
	{
		Tool:  "reopen_inbound_messages",
		Input: "TriageSelectionInput",
		Extra: map[string]string{triageMessageIDs: triageSelectionIDs},
	},
	{
		Tool:  "snooze_inbound_messages",
		Input: "SnoozeInboundMessagesInput",
		Extra: map[string]string{triageMessageIDs: triageSelectionIDs},
	},
	{Tool: "create_inbound_load", Input: "TriageLoadFieldInput", Param: "fields"},
	{
		Tool:   "file_inbound_attachment",
		Input:  "FileInboundAttachmentInput",
		Nested: map[string]string{"pages": "InboundAttachmentPagesInput"},
		Extra: map[string]string{
			inboundMessageID: "the message the attachment came on, which decides how far the " +
				"call may run and must hold the attachment; the mutation reads it from the " +
				"attachment",
		},
	},
	{
		Tool:  "log_inbound_exception",
		Input: "LogInboundExceptionInput",
		Extra: map[string]string{inboundMessageID: inboundMessageArg},
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
	bulkDecision = "approves a set of items by one status decision each; the mutation's " +
		"input decides one item, so the tool takes their ids instead"
	oneMoveTender = "tenders one move; tenderShipments takes a batch of shipment items, so " +
		"the tool takes its move instead"
	pendingBinding = "pending: written before the contract, with parameters named for the " +
		"model where the input names them differently; binding it is the write waves' work"
)

var Unbound = map[string]string{
	"add_carrier_settlement_adjustment": pendingBinding,
	"add_dashboard_tile":                layoutPart,
	"add_driver_settlement_adjustment":  pendingBinding,
	"add_home_widget":                   layoutPart,
	"order_add_charge": "addOrderCharge takes the charge as scalar arguments; only its payer sp" +
		"lit is an input object, which the tool does not take",
	"shipment_add_charge": "adds one accessorial to a saved shipment through the shipment " +
		"service's update; it takes the charge and its units, never the whole ShipmentInput",
	"adjust_escrow_account":           pendingBinding,
	"apply_carrier_intel_suggestions": pendingBinding,
	"apply_credit_memo":               pendingBinding,
	"apply_customer_payment":          pendingBinding,
	"approve_billing_queue_item":      statusDecision,
	"approve_billing_queue_items":     bulkDecision,
	"assign_billing_queue_billers": "assigns one biller to a set of items; the mutation's " +
		"input names one item, so the tool takes their ids instead",
	"arrange_home_layout":                  layoutPart,
	"assess_late_charges":                  pendingBinding,
	"assign_move":                          pendingBinding,
	"attach_pay_events_to_settlement":      pendingBinding,
	"cancel_billing_queue_item":            statusDecision,
	"confirm_accounting_mapping_proposals": pendingBinding,
	"create_accounting_reference_record":   pendingBinding,
	"create_carrier_invoice_match":         pendingBinding,
	"create_dashboard":                     pendingBinding,
	"order_create":                         pendingBinding,
	"create_recurring_deduction":           pendingBinding,
	"create_recurring_earning":             pendingBinding,
	"create_report":                        pendingBinding,
	"detach_pay_event_from_settlement":     pendingBinding,
	"dismiss_accounting_drift":             pendingBinding,
	"dispute_detention":                    pendingBinding,
	"shipment_update_comment":              pendingBinding,
	"email_customer": "composes any customer email; notifyShipmentDelay's input is one fixed " +
		"kind of it, so the tool keeps its own parameters",
	"fork_report":                       pendingBinding,
	"generate_carrier_settlement_batch": pendingBinding,
	"generate_driver_settlement":        pendingBinding,
	"generate_driver_settlement_batch":  pendingBinding,
	"guide_profile": "takes which profile the line is for as well as the line; each " +
		"mutation fixes the profile by its name, so AgentProfileGuidanceInput carries only " +
		"the line",
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
	"order_remove_charge":                  pendingBinding,
	"reply_to_inbound_message": "answers the sender under the original subject in the " +
		"organization's letterhead and settles the message; it takes the body and an email " +
		"profile, never the subject, address or template sendInboundReply's input lets a person set",
	"shipment_reschedule_stop": "moves one stop's window through the shipment service's update; it " +
		"takes the stop and its new window, never the whole ShipmentInput",
	"resolve_accounting_drift":           pendingBinding,
	"resolve_carrier_intel_event":        pendingBinding,
	"retry_accounting_sync":              pendingBinding,
	"reverse_customer_payment":           pendingBinding,
	"save_table_view":                    pendingBinding,
	"schedule_report":                    pendingBinding,
	"send_billing_item_back_to_ops":      statusDecision,
	"set_accounting_mapping":             pendingBinding,
	"order_set_charge_allocations":       pendingBinding,
	"set_worker_availability_preference": pendingBinding,
	"skip_accounting_sync":               pendingBinding,
	"tender_move_to_carriers":            oneMoveTender,
	"tender_move_to_routing_guide":       oneMoveTender,
	"shipment_transfer_ownership":        pendingBinding,
	"transfer_to_billing": "one tool for four mutations (one shipment, its items, a bulk transfer " +
		"and a transfer run), so no single input is its shape",
	"transition_item_to_in_review": statusDecision,
	"transition_items_to_in_review": "moves a set of items into review by one status decision " +
		"each; the mutation's input decides one item, so the tool takes their ids instead",
	"unapply_credit_memo":        pendingBinding,
	"update_escrow_account":      pendingBinding,
	"order_update":               pendingBinding,
	"order_update_charge":        pendingBinding,
	"update_recurring_deduction": pendingBinding,
	"update_recurring_earning":   pendingBinding,
	"update_report":              pendingBinding,
	"update_tractor_status":      pendingBinding,
	"update_trailer_status":      pendingBinding,
	"verify_carrier_equipment":   pendingBinding,
	"waive_detention":            pendingBinding,
}
