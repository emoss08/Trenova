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
	noteParam             = "note"
	keptNotes             = "one text under two names: the input's notes, and note, the name every " +
		"tool gives text kept on the record"
	exceptionIDArg = "the mutation takes the exception as its id argument, beside the input"
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
	{
		Tool:  "factoring_create_noa",
		Input: "DraftFactoringNoaInput",
		Extra: map[string]string{noteParam: keptNotes},
	},
	{
		Tool:      "factoring_send_noa",
		Input:     "SendFactoringNoaInput",
		Defaulted: map[string]string{"id": namedID(noaIDParam)},
		Extra:     map[string]string{noaIDParam: namedID(noaIDParam)},
	},
	{
		Tool:      "factoring_record_noa_acknowledgement",
		Input:     "AcknowledgeFactoringNoaInput",
		Defaulted: map[string]string{"id": namedID(noaIDParam)},
		Extra:     map[string]string{noaIDParam: namedID(noaIDParam)},
	},
	{
		Tool:      "factoring_release_noa",
		Input:     "ReleaseFactoringNoaInput",
		Defaulted: map[string]string{"id": namedID(noaIDParam)},
		Extra:     map[string]string{noaIDParam: namedID(noaIDParam)},
	},
	{Tool: "factoring_set_override", Input: "SetInvoiceFactoringOverrideInput"},
	{
		Tool:  "factoring_retry_submission",
		Input: "RetryFactoringSubmissionInput",
		Defaulted: map[string]string{
			"id":         namedID("submissionId"),
			versionField: readVersion,
		},
		Extra: map[string]string{"submissionId": namedID("submissionId")},
	},
	{Tool: "factoring_request_document", Input: "RequestMissingFactoringDocumentsInput"},
	{
		Tool:   "factoring_record_funding",
		Input:  "RecordFactoringFundingInput",
		Nested: map[string]string{"items": "FactoringFundingItemInput"},
		Extra: map[string]string{
			"items.advance": decimalOf("advanceMinor"),
			"items.fee":     decimalOf("feeMinor"),
		},
	},
	{
		Tool:  "factoring_release_reserve",
		Input: reserveMovementInput,
		Extra: map[string]string{"cash": decimalOf("cashMinor")},
	},
	{
		Tool:  "factoring_record_chargeback",
		Input: reserveMovementInput,
		Extra: map[string]string{"cash": decimalOf("cashMinor")},
	},
	{Tool: "factoring_reverse_chargeback", Input: reserveMovementInput},
	{Tool: "factoring_resolve_remittance_line", Input: "UpdateFactoringRemittanceLineInput"},
	{Tool: "factoring_rematch_remittance", Input: remittanceVersionInput},
	{Tool: "factoring_apply_remittance", Input: remittanceVersionInput},
	{Tool: "factoring_discard_remittance", Input: "DiscardFactoringRemittanceInput"},
	{
		Tool:  "exception_assign",
		Input: "AssignWatchtowerItemInput",
		Extra: map[string]string{watchtowerItemID: exceptionItemArg},
	},
	{
		Tool:  "exception_snooze",
		Input: "SnoozeWatchtowerItemInput",
		Extra: map[string]string{watchtowerItemID: exceptionItemArg},
	},
	{
		Tool:  "exception_set_due",
		Input: "SetWatchtowerItemDueInput",
		Extra: map[string]string{watchtowerItemID: exceptionItemArg},
	},
	{
		Tool:      "exception_resolve",
		Input:     "ResolveWatchtowerItemInput",
		Defaulted: map[string]string{"code": resolutionCode},
		Extra: map[string]string{
			watchtowerItemID: exceptionItemArg,
			"resolutionCode": resolutionCode,
		},
	},
	{
		Tool:  "exception_reopen",
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
	},
	{
		Tool:  "shipment_update",
		Input: "ShipmentInput",
		Defaulted: map[string]string{
			"serviceTypeId":  patchLeavesIt,
			"shipmentTypeId": patchLeavesIt,
			"customerId":     patchLeavesIt,
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
		Tool:      "dispatch_record_check_call",
		Input:     "LogCheckCallInput",
		Defaulted: map[string]string{"shipmentMoveId": namedID("moveId")},
		Extra:     map[string]string{"moveId": namedID("moveId")},
	},
	{
		Tool:  "exception_assign",
		Input: "AssignWatchtowerItemInput",
		Extra: map[string]string{watchtowerItemIDParam: exceptionIDArg},
	},
	{
		Tool:  "exception_snooze",
		Input: "SnoozeWatchtowerItemInput",
		Extra: map[string]string{watchtowerItemIDParam: exceptionIDArg},
	},
	{
		Tool:  "exception_set_due",
		Input: "SetWatchtowerItemDueInput",
		Extra: map[string]string{watchtowerItemIDParam: exceptionIDArg},
	},
	{
		Tool:  "exception_resolve",
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
		Tool:  "exception_reopen",
		Input: "ReopenWatchtowerItemInput",
		Extra: map[string]string{watchtowerItemIDParam: exceptionIDArg},
	},
	{
		Tool:  "shipment_share_tracking_link",
		Input: "IssueCustomerTrackingLinkInput",
	},
	{
		Tool:      "dispatch_share_tracking_link",
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
		Tool:  "billing_correct_charge_codes",
		Input: "ShipmentAdditionalChargeInput",
		Param: "additionalCharges",
	},
	{
		Tool:      "worker_attach_credential_document",
		Input:     "AttachWorkerCredentialDocumentInput",
		Defaulted: map[string]string{"id": namedID("credentialId")},
		Extra:     map[string]string{"credentialId": namedID("credentialId")},
	},
	{
		Tool:      "worker_attach_training_document",
		Input:     "AttachWorkerTrainingDocumentInput",
		Defaulted: map[string]string{"id": namedID("trainingRecordId")},
		Extra:     map[string]string{"trainingRecordId": namedID("trainingRecordId")},
	},
	{
		Tool:   "worker_update_performance_review",
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
		Tool:  "worker_record_recognition",
		Input: "WorkerRecognitionInput",
	},
	{
		Tool:  "worker_open_leave_case",
		Input: "OpenLeaveCaseInput",
		Extra: map[string]string{noteParam: keptNotes},
	},
	{
		Tool:  "worker_open_safety_event",
		Input: "WorkerSafetyEventInput",
	},
	{
		Tool:  "worker_record_employment_verification",
		Input: "RecordEmploymentVerificationInput",
		Extra: map[string]string{noteParam: keptNotes},
	},
	{
		Tool:      "worker_record_leave_day",
		Input:     "RecordLeaveDayInput",
		Defaulted: map[string]string{"caseId": namedID("leaveCaseId")},
		Extra: map[string]string{
			"leaveCaseId": namedID("leaveCaseId"),
			noteParam:     keptNotes,
		},
	},
	{
		Tool:  "worker_record_safety_violation",
		Input: "RecordSafetyViolationInput",
	},
	{
		Tool:  "worker_record_training_completion",
		Input: "CompleteWorkerTrainingInput",
		Extra: map[string]string{
			noteParam:          keptNotes,
			"trainingRecordId": namedID("trainingRecordId"),
		},
	},
	{
		Tool:  "worker_record_credential",
		Input: "WorkerCredentialInput",
		Extra: map[string]string{noteParam: keptNotes},
	},
	{
		Tool:  "worker_record_injury",
		Input: "RecordWorkerInjuryInput",
		Extra: map[string]string{noteParam: keptNotes},
	},
	{
		Tool:  "worker_request_pto",
		Input: "CreateWorkerPTOInput",
		Defaulted: map[string]string{
			"reason": "left out, the kind of time off is the reason: \"put it in as vacation\" " +
				"says why, and asking for a reason only delayed the request",
		},
	},
	{
		Tool:  "worker_schedule_dot_test",
		Input: "RecordDOTTestInput",
		Extra: map[string]string{noteParam: keptNotes},
	},
	{
		Tool:  "worker_start_performance_review",
		Input: "CreatePerformanceReviewInput",
	},
	{
		Tool:  "worker_start_checklist",
		Input: "StartWorkerChecklistInput",
	},
	{
		Tool:  "worker_update_employment_verification",
		Input: "UpdateEmploymentVerificationInput",
		Extra: map[string]string{noteParam: keptNotes},
	},
	{
		Tool:      "worker_update_leave_case",
		Input:     "UpdateLeaveCaseInput",
		Defaulted: map[string]string{"caseId": namedID("leaveCaseId")},
		Extra: map[string]string{
			"leaveCaseId": namedID("leaveCaseId"),
			noteParam:     keptNotes,
		},
	},
	{
		Tool:      "worker_update_leave_day",
		Input:     "UpdateLeaveDayInput",
		Defaulted: map[string]string{"entryId": namedID("leaveDayId")},
		Extra: map[string]string{
			"leaveDayId": namedID("leaveDayId"),
			noteParam:    keptNotes,
		},
	},
	{
		Tool:  "worker_update_safety_violation",
		Input: "UpdateSafetyViolationInput",
		Defaulted: map[string]string{
			"id":          namedID("violationId"),
			"description": patchLeavesIt,
		},
		Extra: map[string]string{"violationId": namedID("violationId")},
	},
	{
		Tool:      "worker_update_checklist_item",
		Input:     "WorkerChecklistItemActionInput",
		Defaulted: map[string]string{"id": namedID("checklistItemId")},
		Extra: map[string]string{
			"checklistItemId": namedID("checklistItemId"),
			"action": "picks which item mutation runs: complete, skip and not applicable " +
				"take this input, and reopen takes the item's id",
		},
	},
	{
		Tool:  "worker_update_credential",
		Input: "UpdateWorkerCredentialInput",
		Defaulted: map[string]string{
			"id":         namedID("credentialId"),
			versionField: readVersion,
		},
		Extra: map[string]string{
			"credentialId": namedID("credentialId"),
			noteParam:      keptNotes,
		},
	},
	{
		Tool:  "worker_update_injury",
		Input: "UpdateWorkerInjuryInput",
		Extra: map[string]string{noteParam: keptNotes},
	},
	{
		Tool:  "worker_update_pto",
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
		Tool:  "worker_update_safety_event",
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
		Tool:  "fuel_record_purchase",
		Input: "FuelPurchaseInput",
		Defaulted: map[string]string{
			"totalAmount": "the tool works the total out from quantity and unitPrice when a " +
				"receipt is read out as gallons and a price, and refuses when it has neither",
		},
		Extra: map[string]string{noteParam: keptNotes},
	},
	{
		Tool:  "fuel_correct_purchase",
		Input: "FuelPurchaseInput",
		Defaulted: map[string]string{
			"tractorId":      patchLeavesIt,
			"jurisdictionId": patchLeavesIt,
			"purchasedAt":    patchLeavesIt,
			"fuelType":       patchLeavesIt,
			"quantity":       patchLeavesIt,
			"totalAmount":    patchLeavesIt,
		},
		Extra: map[string]string{
			"fuelPurchaseId": namedID("fuelPurchaseId"),
			noteParam:        keptNotes,
		},
	},
	{
		Tool:  "fuel_assign_card",
		Input: "AssignFuelCardInput",
		Defaulted: map[string]string{
			"id":         namedID("fuelCardId"),
			versionField: readVersion,
		},
		Extra: map[string]string{"fuelCardId": namedID("fuelCardId")},
	},
	{Tool: "rate_record_fuel_index_price", Input: "FuelIndexPriceInput"},
	{
		Tool:      "rate_correct_fuel_index_price",
		Input:     "UpdateFuelIndexPriceInput",
		Defaulted: map[string]string{"id": namedID("fuelIndexPriceId")},
		Extra:     map[string]string{"fuelIndexPriceId": namedID("fuelIndexPriceId")},
	},
	{
		Tool:  "fuel_record_ifta_mileage_entry",
		Input: "IFTAMileageEntryInput",
		Extra: map[string]string{noteParam: keptNotes},
	},
	{
		Tool:  "fuel_correct_ifta_mileage_entry",
		Input: "IFTAMileageEntryInput",
		Defaulted: map[string]string{
			"tractorId":      patchLeavesIt,
			"jurisdictionId": patchLeavesIt,
			"traveledAt":     patchLeavesIt,
			"miles":          patchLeavesIt,
		},
		Extra: map[string]string{
			"iftaMileageEntryId": namedID("iftaMileageEntryId"),
			noteParam:            keptNotes,
		},
	},
	{Tool: "fuel_generate_ifta_return", Input: "IFTAPeriodInput"},
	{
		Tool:  "carrier_create_capacity_posting",
		Input: "CarrierCapacityPostingInput",
		Extra: map[string]string{
			noteParam:          keptNotes,
			"originState":      stateCode("originStateId"),
			"destinationState": stateCode("destinationStateId"),
		},
	},
	{
		Tool:  "carrier_update_capacity_posting",
		Input: "CarrierCapacityPostingInput",
		Defaulted: map[string]string{
			"carrierId":     patchLeavesIt,
			"availableFrom": patchLeavesIt,
			"availableTo":   patchLeavesIt,
		},
		Extra: map[string]string{
			"carrierCapacityPostingId": "the mutation takes the posting as its id argument, " +
				"beside the input",
			noteParam:          keptNotes,
			"originState":      stateCode("originStateId"),
			"destinationState": stateCode("destinationStateId"),
		},
	},
	{
		Tool:  "equipment_create_tractor",
		Input: "TractorInput",
		Extra: map[string]string{"state": stateCode("stateId")},
	},
	{
		Tool:  "equipment_update_tractor",
		Input: "TractorPatchInput",
		Extra: map[string]string{
			"tractorId": "the mutation takes the tractor as its id argument, beside the input",
			"state":     stateCode("stateId"),
		},
	},
	{Tool: "equipment_set_tractor_location", Input: "LocateTractorInput"},
	{
		Tool:  "equipment_create_trailer",
		Input: "TrailerInput",
		Extra: map[string]string{"registrationState": stateCode("registrationStateId")},
	},
	{
		Tool:  "equipment_update_trailer",
		Input: "TrailerPatchInput",
		Extra: map[string]string{
			"trailerId":         "the mutation takes the trailer as its id argument, beside the input",
			"registrationState": stateCode("registrationStateId"),
		},
	},
	{Tool: "equipment_set_trailer_location", Input: "LocateTrailerInput"},
	{
		Tool:      "inbox_file_capture_item",
		Input:     "FileCaptureItemsEntryInput",
		Param:     "items",
		Defaulted: map[string]string{"version": readVersion},
	},
	{
		Tool:  "inbox_assign_inbound_message",
		Input: "TriageSelectionInput",
		Extra: map[string]string{
			triageMessageIDs: triageSelectionIDs,
			"ownerId": "the mutation takes the owner as its own argument, beside the " +
				"selection",
		},
	},
	{
		Tool:  "inbox_reopen_inbound_message",
		Input: "TriageSelectionInput",
		Extra: map[string]string{triageMessageIDs: triageSelectionIDs},
	},
	{
		Tool:  "inbox_snooze_inbound_message",
		Input: "SnoozeInboundMessagesInput",
		Extra: map[string]string{triageMessageIDs: triageSelectionIDs},
	},
	{Tool: "inbox_create_load", Input: "TriageLoadFieldInput", Param: "fields"},
	{
		Tool:   "inbox_file_attachment",
		Input:  "FileInboundAttachmentInput",
		Nested: map[string]string{"pages": "InboundAttachmentPagesInput"},
		Extra: map[string]string{
			inboundMessageID: "the message the attachment came on, which decides how far the " +
				"call may run and must hold the attachment; the mutation reads it from the " +
				"attachment",
		},
	},
	{
		Tool:  "inbox_record_discrepancy",
		Input: "LogInboundExceptionInput",
		Extra: map[string]string{inboundMessageID: inboundMessageArg},
	},
	{
		Tool:   "invoice_create_memo",
		Input:  "CreateMemoInput",
		Nested: map[string]string{"lines": "MemoLineInput"},
		Narrowed: map[string]string{
			"billType": "a memo is a credit or a debit; an invoice is issued by invoice_create",
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
	"settlement_add_carrier_adjustment": pendingBinding,
	"report_add_dashboard_tile":         layoutPart,
	"settlement_add_driver_adjustment":  pendingBinding,
	"home_add_widget":                   layoutPart,
	"order_add_charge": "addOrderCharge takes the charge as scalar arguments; only its payer sp" +
		"lit is an input object, which the tool does not take",
	"shipment_add_charge": "adds one accessorial to a saved shipment through the shipment " +
		"service's update; it takes the charge and its units, never the whole ShipmentInput",
	"settlement_adjust_escrow_account":       pendingBinding,
	"carrier_apply_intelligence_suggestions": pendingBinding,
	"receivable_apply_credit_memo":           pendingBinding,
	"receivable_apply_customer_payment":      pendingBinding,
	"billing_approve_queue_item":             bulkDecision,
	"billing_assign_biller": "assigns one biller to a set of items; the mutation's " +
		"input names one item, so the tool takes their ids instead",
	"home_arrange_layout":                   layoutPart,
	"receivable_assess_late_charges":        pendingBinding,
	"dispatch_assign_move":                  pendingBinding,
	"settlement_attach_pay_events":          pendingBinding,
	"billing_cancel_queue_item":             statusDecision,
	"accounting_confirm_mapping":            pendingBinding,
	"accounting_create_reference_record":    pendingBinding,
	"carrier_create_invoice_match":          pendingBinding,
	"report_create_dashboard":               pendingBinding,
	"order_create":                          pendingBinding,
	"settlement_create_recurring_deduction": pendingBinding,
	"settlement_create_recurring_earning":   pendingBinding,
	"report_create":                         pendingBinding,
	"settlement_detach_pay_event":           pendingBinding,
	"accounting_dismiss_drift_finding":      pendingBinding,
	"detention_open_dispute":                pendingBinding,
	"shipment_update_comment":               pendingBinding,
	"customer_send_email": "composes any customer email; notifyShipmentDelay's input is one fixed " +
		"kind of it, so the tool keeps its own parameters",
	"report_fork":                       pendingBinding,
	"settlement_generate_carrier_batch": pendingBinding,
	"settlement_generate_driver":        pendingBinding,
	"settlement_generate_driver_batch":  pendingBinding,
	"agent_guide_profile": "takes which profile the line is for as well as the line; each " +
		"mutation fixes the profile by its name, so AgentProfileGuidanceInput carries only " +
		"the line",
	"billing_hold_queue_item":              statusDecision,
	"settlement_hold_pay_event":            pendingBinding,
	"accounting_ignore_inbound_change":     pendingBinding,
	"settlement_issue_advance":             pendingBinding,
	"inbox_link_inbound_message":           pendingBinding,
	"inbox_mark_inbound_message":           pendingBinding,
	"billing_raise_exception":              statusDecision,
	"settlement_open_escrow_account":       pendingBinding,
	"invoice_open_dispute":                 pendingBinding,
	"accounting_pause_sync":                pendingBinding,
	"receivable_post_customer_payment":     pendingBinding,
	"worker_request_shift_swap":            pendingBinding,
	"settlement_recalculate_carrier":       pendingBinding,
	"settlement_recalculate_driver":        pendingBinding,
	"settlement_record_carrier_payment":    pendingBinding,
	"settlement_record_driver_payment":     pendingBinding,
	"accounting_release_sync_record":       pendingBinding,
	"agent_create_memory":                  pendingBinding,
	"settlement_remove_carrier_adjustment": pendingBinding,
	"settlement_remove_driver_adjustment":  pendingBinding,
	"home_remove_widget":                   layoutPart,
	"order_remove_charge":                  pendingBinding,
	"inbox_send_reply": "answers the sender under the original subject in the " +
		"organization's letterhead and settles the message; it takes the body and an email " +
		"profile, never the subject, address or template sendInboundReply's input lets a person set",
	"shipment_reschedule_stop": "moves one stop's window through the shipment service's update; it " +
		"takes the stop and its new window, never the whole ShipmentInput",
	"accounting_resolve_drift_finding":          pendingBinding,
	"carrier_resolve_intelligence_event":        pendingBinding,
	"accounting_retry_sync_record":              pendingBinding,
	"receivable_reverse_customer_payment":       pendingBinding,
	"report_save_table_view":                    pendingBinding,
	"report_create_schedule":                    pendingBinding,
	"billing_return_to_operations":              statusDecision,
	"accounting_set_mapping":                    pendingBinding,
	"order_set_charge_allocations":              pendingBinding,
	"worker_set_availability_preference":        pendingBinding,
	"accounting_skip_sync_record":               pendingBinding,
	"dispatch_create_tender":                    oneMoveTender,
	"dispatch_create_tender_from_routing_guide": oneMoveTender,
	"shipment_transfer_ownership":               pendingBinding,
	"billing_transfer": "one tool for four mutations (one shipment, its items, a bulk transfer " +
		"and a transfer run), so no single input is its shape",
	"billing_start_review": "moves a set of items into review by one status decision " +
		"each; the mutation's input decides one item, so the tool takes their ids instead",
	"receivable_unapply_credit_memo":        pendingBinding,
	"settlement_update_escrow_account":      pendingBinding,
	"order_update":                          pendingBinding,
	"order_update_charge":                   pendingBinding,
	"settlement_update_recurring_deduction": pendingBinding,
	"settlement_update_recurring_earning":   pendingBinding,
	"report_update":                         pendingBinding,
	"equipment_set_tractor_status":          pendingBinding,
	"equipment_set_trailer_status":          pendingBinding,
	"carrier_verify_equipment":              pendingBinding,
	"detention_waive":                       pendingBinding,
}
