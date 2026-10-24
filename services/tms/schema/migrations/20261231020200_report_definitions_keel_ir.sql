-- Converts stored report definitions, their revisions and dashboard layouts from Trenova IR
-- version 1 to keel analytics IR version 2, renaming entities, fields and edges to the keys the
-- keel catalog derives from the models. A cleared catalog_version makes the needs-attention
-- sweep recheck every converted definition against the current catalog.
CREATE TEMPORARY TABLE "rpt_entity"("old" text PRIMARY KEY, "new" text NOT NULL) ON COMMIT DROP;
CREATE TEMPORARY TABLE "rpt_field"("entity" text, "old" text, "new" text NOT NULL, PRIMARY KEY ("entity", "old")) ON COMMIT DROP;
CREATE TEMPORARY TABLE "rpt_edge"("entity" text, "old" text, "new" text NOT NULL, "target" text NOT NULL, PRIMARY KEY ("entity", "old")) ON COMMIT DROP;

INSERT INTO "rpt_entity"("old", "new") VALUES
('accessorial_charge', 'accessorial_charge'),
('account_type', 'account_type'),
('accounting_drift_finding', 'accounting_drift_finding'),
('accounting_inbound_change', 'accounting_inbound_change'),
('accounting_sync_record', 'accounting_sync_record'),
('assignment', 'assignment'),
('bank_receipt', 'bank_receipt'),
('carrier', 'carrier'),
('carrier_cost_event', 'cost_event'),
('carrier_settlement', 'carrier_settlement'),
('commodity', 'commodity'),
('customer', 'customer'),
('customer_ledger_entry', 'customer_ledger_entry'),
('customer_payment', 'payment'),
('customer_payment_application', 'application'),
('driver_pay_event', 'pay_event'),
('driver_settlement', 'settlement'),
('driver_settlement_batch', 'settlement_batch'),
('driver_settlement_line', 'settlement_line'),
('equipment_manufacturer', 'equipment_manufacturer'),
('equipment_type', 'equipment_type'),
('escrow_account', 'escrow_account'),
('escrow_transaction', 'escrow_transaction'),
('factoring_agreement', 'agreement'),
('factoring_company', 'factoring_company'),
('factoring_invoice_state', 'invoice_state'),
('factoring_noa', 'noa'),
('factoring_remittance', 'remittance'),
('factoring_remittance_line', 'remittance_line'),
('factoring_reserve_entry', 'reserve_entry'),
('factoring_submission', 'submission'),
('factoring_submission_item', 'submission_item'),
('fiscal_period', 'fiscal_period'),
('fiscal_year', 'fiscal_year'),
('fleet_code', 'fleet_code'),
('fuel_card', 'fuel_card'),
('fuel_purchase', 'fuel_purchase'),
('gl_account', 'gl_account'),
('gl_account_balance', 'gl_account_period_balance'),
('hazardous_material', 'hazardous_material'),
('ifta_jurisdiction_mileage_entry', 'jurisdiction_mileage_entry'),
('ifta_return', 'return'),
('ifta_return_line', 'return_line'),
('invoice', 'invoice'),
('invoice_adjustment', 'invoice_adjustment'),
('invoice_adjustment_line', 'invoice_adjustment_line'),
('invoice_line', 'invoice_line'),
('job_position', 'job_position'),
('journal_batch', 'journal_batch'),
('journal_entry', 'journal_entry'),
('journal_entry_line', 'journal_entry_line'),
('journal_source', 'source'),
('location', 'location'),
('location_category', 'location_category'),
('order', 'order'),
('pay_advance', 'pay_advance'),
('pay_code', 'pay_code'),
('rate_agreement', 'rate_agreement'),
('rate_agreement_accessorial', 'rate_agreement_accessorial'),
('rate_agreement_rule', 'rate_agreement_rule'),
('rate_agreement_version', 'rate_agreement_version'),
('rate_quote', 'rate_quote'),
('recurring_deduction', 'recurring_deduction'),
('recurring_earning', 'recurring_earning'),
('report_definition', 'report_definition'),
('report_run', 'report_run'),
('service_type', 'service_type'),
('shipment', 'shipment'),
('shipment_commodity', 'shipment_commodity'),
('shipment_move', 'shipment_move'),
('shipment_type', 'shipment_type'),
('stop', 'stop'),
('tractor', 'tractor'),
('trailer', 'trailer'),
('worker', 'worker'),
('worker_credential', 'worker_credential'),
('worker_credential_type', 'worker_credential_type'),
('worker_employment_event', 'worker_employment_event'),
('worker_profile', 'worker_profile'),
('worker_pto', 'worker_pto'),
('worker_pto_balance', 'worker_pto_balance');

INSERT INTO "rpt_field"("entity", "old", "new") VALUES
('accounting_drift_finding', 'connectionId', 'connectionID'),
('accounting_drift_finding', 'externalId', 'externalID'),
('accounting_drift_finding', 'externalUrl', 'externalURL'),
('accounting_drift_finding', 'fixObjectId', 'fixObjectID'),
('accounting_drift_finding', 'objectId', 'objectID'),
('accounting_drift_finding', 'partyId', 'partyID'),
('accounting_drift_finding', 'resolvedById', 'resolvedByID'),
('accounting_inbound_change', 'connectionId', 'connectionID'),
('accounting_inbound_change', 'decidedById', 'decidedByID'),
('accounting_inbound_change', 'externalId', 'externalID'),
('accounting_inbound_change', 'externalUrl', 'externalURL'),
('accounting_inbound_change', 'partyObjectId', 'partyObjectID'),
('accounting_sync_record', 'connectionId', 'connectionID'),
('accounting_sync_record', 'externalId', 'externalID'),
('accounting_sync_record', 'externalUrl', 'externalURL'),
('accounting_sync_record', 'objectId', 'objectID'),
('accounting_sync_record', 'redatedById', 'redatedByID'),
('accounting_sync_record', 'releasedById', 'releasedByID'),
('accounting_sync_record', 'skippedById', 'skippedByID'),
('assignment', 'primaryWorkerId', 'primaryWorker.id'),
('assignment', 'secondaryWorkerId', 'secondaryWorker.id'),
('assignment', 'shipmentMoveId', 'shipmentMove.id'),
('assignment', 'tractorId', 'tractor.id'),
('assignment', 'trailerId', 'trailer.id'),
('bank_receipt', 'createdById', 'createdByID'),
('bank_receipt', 'importBatchId', 'importBatchID'),
('bank_receipt', 'matchedById', 'matchedByID'),
('bank_receipt', 'matchedCustomerPaymentId', 'matchedCustomerPayment.id'),
('bank_receipt', 'updatedById', 'updatedByID'),
('carrier', 'externalId', 'externalID'),
('carrier', 'remitStateId', 'remitStateID'),
('carrier', 'stateId', 'stateID'),
('carrier', 'taxId', 'taxID'),
('carrier', 'taxIdType', 'taxIDType'),
('carrier_cost_event', 'carrierAssignmentId', 'carrierAssignmentID'),
('carrier_cost_event', 'carrierId', 'carrier.id'),
('carrier_cost_event', 'moveId', 'moveID'),
('carrier_cost_event', 'settlementId', 'settlementID'),
('carrier_cost_event', 'shipmentId', 'shipmentID'),
('carrier_settlement', 'approvedById', 'approvedByID'),
('carrier_settlement', 'batchId', 'batchID'),
('carrier_settlement', 'carrierId', 'carrier.id'),
('carrier_settlement', 'paidById', 'paidByID'),
('carrier_settlement', 'paidJournalBatchId', 'paidJournalBatchID'),
('carrier_settlement', 'postedApAccountId', 'postedAPAccountID'),
('carrier_settlement', 'postedById', 'postedByID'),
('carrier_settlement', 'postedExpenseAccountId', 'postedExpenseAccountID'),
('carrier_settlement', 'postedJournalBatchId', 'postedJournalBatchID'),
('carrier_settlement', 'submittedById', 'submittedByID'),
('carrier_settlement', 'voidJournalBatchId', 'voidJournalBatchID'),
('carrier_settlement', 'voidedById', 'voidedByID'),
('commodity', 'hazardousMaterialId', 'hazardousMaterial.id'),
('customer', 'externalId', 'externalID'),
('customer', 'placeId', 'placeID'),
('customer', 'stateId', 'stateID'),
('customer_ledger_entry', 'createdById', 'createdByID'),
('customer_ledger_entry', 'customerId', 'customer.id'),
('customer_ledger_entry', 'relatedInvoiceId', 'relatedInvoice.id'),
('customer_ledger_entry', 'sourceObjectId', 'sourceObjectID'),
('customer_payment', 'createdById', 'createdByID'),
('customer_payment', 'customerId', 'customer.id'),
('customer_payment', 'postedBatchId', 'postedBatchID'),
('customer_payment', 'reversalBatchId', 'reversalBatchID'),
('customer_payment', 'reversedById', 'reversedByID'),
('customer_payment', 'updatedById', 'updatedByID'),
('customer_payment_application', 'customerPaymentId', 'payment.id'),
('customer_payment_application', 'invoiceId', 'invoice.id'),
('driver_pay_event', 'assignmentId', 'assignmentID'),
('driver_pay_event', 'moveId', 'moveID'),
('driver_pay_event', 'payProfileId', 'payProfileID'),
('driver_pay_event', 'settlementId', 'settlement.id'),
('driver_pay_event', 'settlementLineId', 'settlementLineID'),
('driver_pay_event', 'shipmentId', 'shipment.id'),
('driver_pay_event', 'workerId', 'worker.id'),
('driver_settlement', 'approvedById', 'approvedByID'),
('driver_settlement', 'batchId', 'batch.id'),
('driver_settlement', 'paidById', 'paidByID'),
('driver_settlement', 'paidJournalBatchId', 'paidJournalBatchID'),
('driver_settlement', 'payProfileId', 'payProfileID'),
('driver_settlement', 'postedById', 'postedByID'),
('driver_settlement', 'postedJournalBatchId', 'postedJournalBatchID'),
('driver_settlement', 'postedPayableAccountId', 'postedPayableAccountID'),
('driver_settlement', 'submittedById', 'submittedByID'),
('driver_settlement', 'voidJournalBatchId', 'voidJournalBatchID'),
('driver_settlement', 'voidedById', 'voidedByID'),
('driver_settlement', 'workerId', 'worker.id'),
('driver_settlement_batch', 'canceledById', 'canceledByID'),
('driver_settlement_batch', 'generatedById', 'generatedByID'),
('driver_settlement_line', 'advanceId', 'advanceID'),
('driver_settlement_line', 'escrowAccountId', 'escrowAccountID'),
('driver_settlement_line', 'moveId', 'moveID'),
('driver_settlement_line', 'payCodeId', 'payCode.id'),
('driver_settlement_line', 'payEventId', 'payEventID'),
('driver_settlement_line', 'recurringDeductionId', 'recurringDeductionID'),
('driver_settlement_line', 'recurringEarningId', 'recurringEarningID'),
('driver_settlement_line', 'settlementId', 'settlement.id'),
('driver_settlement_line', 'shipmentId', 'shipment.id'),
('escrow_account', 'workerId', 'worker.id'),
('escrow_transaction', 'createdById', 'createdByID'),
('escrow_transaction', 'escrowAccountId', 'escrowAccount.id'),
('escrow_transaction', 'settlementId', 'settlementID'),
('factoring_agreement', 'advanceRateBp', 'advanceRateBP'),
('factoring_agreement', 'cashAccountId', 'cashAccountID'),
('factoring_agreement', 'dueFromFactorAccountId', 'dueFromFactorAccountID'),
('factoring_agreement', 'factoringCompanyId', 'factoringCompany.id'),
('factoring_agreement', 'feeExpenseAccountId', 'feeExpenseAccountID'),
('factoring_agreement', 'liabilityAccountId', 'liabilityAccountID'),
('factoring_agreement', 'reserveRateBp', 'reserveRateBP'),
('factoring_company', 'stateId', 'stateID'),
('factoring_invoice_state', 'customerId', 'customer.id'),
('factoring_invoice_state', 'factoringCompanyId', 'factoringCompany.id'),
('factoring_invoice_state', 'id', 'invoice.id'),
('factoring_noa', 'customerId', 'customer.id'),
('factoring_noa', 'factoringCompanyId', 'factoringCompany.id'),
('factoring_noa', 'signedDocumentId', 'signedDocumentID'),
('factoring_remittance', 'appliedById', 'appliedByID'),
('factoring_remittance', 'createdById', 'createdByID'),
('factoring_remittance', 'documentId', 'documentID'),
('factoring_remittance', 'factoringCompanyId', 'factoringCompany.id'),
('factoring_remittance', 'inboundMessageId', 'inboundMessageID'),
('factoring_remittance_line', 'remittanceId', 'remittanceID'),
('factoring_remittance_line', 'reserveEntryId', 'reserveEntryID'),
('factoring_remittance_line', 'submissionItemId', 'submissionItem.id'),
('factoring_reserve_entry', 'agreementId', 'agreementID'),
('factoring_reserve_entry', 'createdById', 'createdByID'),
('factoring_reserve_entry', 'customerId', 'customer.id'),
('factoring_reserve_entry', 'factoringCompanyId', 'factoringCompany.id'),
('factoring_reserve_entry', 'invoiceId', 'invoice.id'),
('factoring_reserve_entry', 'journalBatchId', 'journalBatchID'),
('factoring_reserve_entry', 'journalEntryId', 'journalEntryID'),
('factoring_reserve_entry', 'remittanceLineId', 'remittanceLineID'),
('factoring_reserve_entry', 'reversesEntryId', 'reversesEntryID'),
('factoring_reserve_entry', 'submissionId', 'submission.id'),
('factoring_reserve_entry', 'submissionItemId', 'submissionItemID'),
('factoring_submission', 'agentRunId', 'agentRunID'),
('factoring_submission', 'agreementId', 'agreement.id'),
('factoring_submission', 'factoringCompanyId', 'factoringCompany.id'),
('factoring_submission', 'scheduleCsvDocumentId', 'scheduleCSVDocumentID'),
('factoring_submission', 'schedulePdfDocumentId', 'schedulePDFDocumentID'),
('factoring_submission', 'submittedById', 'submittedByID'),
('factoring_submission_item', 'customerId', 'customer.id'),
('factoring_submission_item', 'invoiceId', 'invoice.id'),
('factoring_submission_item', 'packetDocumentId', 'packetDocumentID'),
('factoring_submission_item', 'submissionId', 'submissionID'),
('fiscal_period', 'closedById', 'closedByID'),
('fiscal_period', 'fiscalYearId', 'fiscalYearID'),
('fiscal_period', 'lockedById', 'lockedByID'),
('fiscal_period', 'reopenedById', 'reopenedByID'),
('fiscal_year', 'closedById', 'closedByID'),
('fiscal_year', 'lockedById', 'lockedByID'),
('fiscal_year', 'reopenedById', 'reopenedByID'),
('fleet_code', 'managerId', 'managerID'),
('fuel_card', 'assignedTractorId', 'assignedTractor.id'),
('fuel_card', 'assignedWorkerId', 'assignedWorker.id'),
('fuel_card', 'externalCardId', 'externalCardID'),
('fuel_purchase', 'createdById', 'createdByID'),
('fuel_purchase', 'fuelCardId', 'fuelCard.id'),
('fuel_purchase', 'importBatchId', 'importBatchID'),
('fuel_purchase', 'jurisdictionId', 'jurisdictionID'),
('fuel_purchase', 'tractorId', 'tractor.id'),
('fuel_purchase', 'workerId', 'worker.id'),
('gl_account', 'accountTypeId', 'accountType.id'),
('gl_account', 'allowManualJe', 'allowManualJE'),
('gl_account', 'parentId', 'parent.id'),
('gl_account_balance', 'fiscalPeriodId', 'fiscalPeriod.id'),
('gl_account_balance', 'fiscalYearId', 'fiscalYear.id'),
('gl_account_balance', 'glAccountId', 'glAccount.id'),
('gl_account_balance', 'lastJournalEntryId', 'lastJournalEntry.id'),
('ifta_jurisdiction_mileage_entry', 'createdById', 'createdByID'),
('ifta_jurisdiction_mileage_entry', 'jurisdictionId', 'jurisdictionID'),
('ifta_jurisdiction_mileage_entry', 'shipmentMoveId', 'shipmentMoveID'),
('ifta_jurisdiction_mileage_entry', 'tractorId', 'tractor.id'),
('ifta_return', 'amendsReturnId', 'amendsReturn.id'),
('ifta_return', 'filedById', 'filedByID'),
('ifta_return', 'finalizedById', 'finalizedByID'),
('ifta_return', 'fleetMpgByFuelType', 'fleetMPGByFuelType'),
('ifta_return', 'reopenedById', 'reopenedByID'),
('ifta_return_line', 'jurisdictionId', 'jurisdictionID'),
('ifta_return_line', 'returnId', 'return.id'),
('invoice', 'billingQueueItemId', 'billingQueueItemID'),
('invoice', 'correctionGroupId', 'correctionGroupID'),
('invoice', 'customerId', 'customer.id'),
('invoice', 'emailBccSnapshot', 'emailBCCSnapshot'),
('invoice', 'emailCcSnapshot', 'emailCCSnapshot'),
('invoice', 'factoringAgreementId', 'factoringAgreementID'),
('invoice', 'factoringCompanyId', 'factoringCompany.id'),
('invoice', 'invoiceRunId', 'invoiceRunID'),
('invoice', 'lastEdiError', 'lastEDIError'),
('invoice', 'lastEdiMessageId', 'lastEDIMessageID'),
('invoice', 'orderId', 'order.id'),
('invoice', 'pdfDocumentId', 'pdfDocumentID'),
('invoice', 'referenceInvoiceId', 'referenceInvoice.id'),
('invoice', 'sentById', 'sentByID'),
('invoice', 'shipmentBol', 'shipmentBOL'),
('invoice', 'shipmentId', 'shipment.id'),
('invoice', 'shipperCustomerId', 'shipperCustomer.id'),
('invoice', 'sourceInvoiceAdjustmentId', 'sourceInvoiceAdjustmentID'),
('invoice', 'supersededByInvoiceId', 'supersededByInvoiceID'),
('invoice', 'supersedesInvoiceId', 'supersedesInvoiceID'),
('invoice', 'voidedByAdjustmentId', 'voidedByAdjustmentID'),
('invoice', 'voidedById', 'voidedByID'),
('invoice_adjustment', 'approvedById', 'approvedByID'),
('invoice_adjustment', 'batchId', 'batchID'),
('invoice_adjustment', 'correctionGroupId', 'correctionGroupID'),
('invoice_adjustment', 'creditMemoInvoiceId', 'creditMemoInvoiceID'),
('invoice_adjustment', 'originalInvoiceId', 'originalInvoiceID'),
('invoice_adjustment', 'rebillQueueItemId', 'rebillQueueItemID'),
('invoice_adjustment', 'rejectedById', 'rejectedByID'),
('invoice_adjustment', 'replacementInvoiceId', 'replacementInvoiceID'),
('invoice_adjustment', 'submittedById', 'submittedByID'),
('invoice_adjustment_line', 'adjustmentId', 'adjustmentID'),
('invoice_adjustment_line', 'creditMemoLineId', 'creditMemoLineID'),
('invoice_adjustment_line', 'originalInvoiceId', 'originalInvoiceID'),
('invoice_adjustment_line', 'originalLineId', 'originalLineID'),
('invoice_adjustment_line', 'replacementLineId', 'replacementLineID'),
('invoice_line', 'accessorialChargeId', 'accessorialChargeID'),
('invoice_line', 'additionalChargeId', 'additionalChargeID'),
('invoice_line', 'chargeAllocationId', 'chargeAllocationID'),
('invoice_line', 'formulaTemplateId', 'formulaTemplateID'),
('invoice_line', 'invoiceId', 'invoice.id'),
('invoice_line', 'shipmentBol', 'shipmentBOL'),
('invoice_line', 'shipmentId', 'shipmentID'),
('job_position', 'reportsToPositionId', 'reportsTo.id'),
('journal_batch', 'createdById', 'createdByID'),
('journal_batch', 'fiscalPeriodId', 'fiscalPeriod.id'),
('journal_batch', 'fiscalYearId', 'fiscalYear.id'),
('journal_batch', 'postedById', 'postedByID'),
('journal_batch', 'updatedById', 'updatedByID'),
('journal_entry', 'approvedById', 'approvedByID'),
('journal_entry', 'batchId', 'batch.id'),
('journal_entry', 'createdById', 'createdByID'),
('journal_entry', 'fiscalPeriodId', 'fiscalPeriod.id'),
('journal_entry', 'fiscalYearId', 'fiscalYear.id'),
('journal_entry', 'postedById', 'postedByID'),
('journal_entry', 'referenceId', 'referenceID'),
('journal_entry', 'rejectedById', 'rejectedByID'),
('journal_entry', 'reversalOfId', 'reversalOf.id'),
('journal_entry', 'reversedById', 'reversedBy.id'),
('journal_entry', 'updatedById', 'updatedByID'),
('journal_entry_line', 'customerId', 'customer.id'),
('journal_entry_line', 'departmentId', 'departmentID'),
('journal_entry_line', 'glAccountId', 'glAccount.id'),
('journal_entry_line', 'journalEntryId', 'journalEntry.id'),
('journal_entry_line', 'locationId', 'location.id'),
('journal_entry_line', 'projectId', 'projectID'),
('journal_entry_line', 'reconciledById', 'reconciledByID'),
('journal_entry_line', 'vendorId', 'vendorID'),
('journal_source', 'journalBatchId', 'journalBatch.id'),
('journal_source', 'journalEntryId', 'journalEntry.id'),
('journal_source', 'sourceObjectId', 'sourceObjectID'),
('location', 'locationCategoryId', 'locationCategory.id'),
('location', 'placeId', 'placeID'),
('location', 'stateId', 'stateID'),
('order', 'customerId', 'customer.id'),
('order', 'enteredById', 'enteredByID'),
('order', 'ownerId', 'ownerID'),
('pay_advance', 'createdById', 'createdByID'),
('pay_advance', 'workerId', 'worker.id'),
('pay_advance', 'writtenOffById', 'writtenOffByID'),
('pay_code', 'glAccountId', 'glAccount.id'),
('rate_agreement', 'approvedById', 'approvedByID'),
('rate_agreement', 'billToCustomerId', 'billToCustomerID'),
('rate_agreement', 'carrierId', 'carrier.id'),
('rate_agreement', 'customerId', 'customer.id'),
('rate_agreement', 'documentId', 'documentID'),
('rate_agreement', 'submittedById', 'submittedByID'),
('rate_agreement_accessorial', 'accessorialChargeId', 'accessorialCharge.id'),
('rate_agreement_accessorial', 'rateAgreementId', 'agreement.id'),
('rate_agreement_accessorial', 'serviceTypeIds', 'serviceTypeIDs'),
('rate_agreement_accessorial', 'shipmentTypeIds', 'shipmentTypeIDs'),
('rate_agreement_rule', 'commodityIds', 'commodityIDs'),
('rate_agreement_rule', 'formulaTemplateId', 'formulaTemplateID'),
('rate_agreement_rule', 'partyId', 'partyID'),
('rate_agreement_rule', 'rateAgreementId', 'agreement.id'),
('rate_agreement_rule', 'serviceTypeIds', 'serviceTypeIDs'),
('rate_agreement_rule', 'shipmentTypeIds', 'shipmentTypeIDs'),
('rate_agreement_rule', 'sourceImportRowId', 'sourceImportRowID'),
('rate_agreement_rule', 'supersedesRuleId', 'supersedesRuleID'),
('rate_agreement_rule', 'tractorTypeIds', 'tractorTypeIDs'),
('rate_agreement_rule', 'trailerTypeIds', 'trailerTypeIDs'),
('rate_agreement_version', 'billToCustomerId', 'billToCustomerID'),
('rate_agreement_version', 'carrierId', 'carrierID'),
('rate_agreement_version', 'createdById', 'createdByID'),
('rate_agreement_version', 'customerId', 'customerID'),
('rate_agreement_version', 'documentId', 'documentID'),
('rate_agreement_version', 'rateAgreementId', 'agreement.id'),
('rate_quote', 'exchangeRateId', 'exchangeRateID'),
('rate_quote', 'formulaTemplateId', 'formulaTemplateID'),
('rate_quote', 'partyId', 'partyID'),
('rate_quote', 'rateAgreementId', 'agreement.id'),
('rate_quote', 'rateAgreementRuleId', 'rateAgreementRuleID'),
('rate_quote', 'ratedById', 'ratedByID'),
('rate_quote', 'shipmentId', 'shipmentID'),
('rate_quote', 'shipmentMoveId', 'shipmentMoveID'),
('recurring_deduction', 'createdById', 'createdByID'),
('recurring_deduction', 'escrowAccountId', 'escrowAccount.id'),
('recurring_deduction', 'payCodeId', 'payCode.id'),
('recurring_deduction', 'workerId', 'worker.id'),
('recurring_earning', 'createdById', 'createdByID'),
('recurring_earning', 'payCodeId', 'payCode.id'),
('recurring_earning', 'workerId', 'worker.id'),
('report_definition', 'ownerId', 'ownerID'),
('report_run', 'definitionId', 'reportDefinition.id'),
('report_run', 'requestedById', 'requestedByID'),
('report_run', 'scheduleId', 'scheduleID'),
('shipment', 'billToCustomerId', 'billToCustomer.id'),
('shipment', 'canceledById', 'canceledByID'),
('shipment', 'consolidationGroupId', 'consolidationGroupID'),
('shipment', 'customerId', 'customer.id'),
('shipment', 'enteredById', 'enteredByID'),
('shipment', 'formulaTemplateId', 'formulaTemplateID'),
('shipment', 'orderId', 'orderID'),
('shipment', 'ownerId', 'ownerID'),
('shipment', 'rateAgreementId', 'rateAgreementID'),
('shipment', 'rateAgreementRuleId', 'rateAgreementRuleID'),
('shipment', 'rateOverrideById', 'rateOverrideByID'),
('shipment', 'rateQuoteId', 'rateQuoteID'),
('shipment', 'serviceTypeId', 'serviceType.id'),
('shipment', 'shipmentTypeId', 'shipmentType.id'),
('shipment', 'tractorTypeId', 'tractorType.id'),
('shipment', 'trailerTypeId', 'trailerType.id'),
('shipment_commodity', 'commodityId', 'commodity.id'),
('shipment_commodity', 'shipmentId', 'shipment.id'),
('shipment_move', 'shipmentId', 'shipment.id'),
('stop', 'locationId', 'location.id'),
('stop', 'shipmentMoveId', 'shipmentMoveID'),
('tractor', 'equipmentManufacturerId', 'equipmentManufacturer.id'),
('tractor', 'equipmentTypeId', 'equipmentType.id'),
('tractor', 'externalId', 'externalID'),
('tractor', 'fleetCodeId', 'fleetCode.id'),
('tractor', 'ownerWorkerId', 'ownerWorker.id'),
('tractor', 'primaryWorkerId', 'primaryWorker.id'),
('tractor', 'secondaryWorkerId', 'secondaryWorker.id'),
('tractor', 'stateId', 'stateID'),
('trailer', 'equipmentManufacturerId', 'equipmentManufacturer.id'),
('trailer', 'equipmentTypeId', 'equipmentType.id'),
('trailer', 'externalId', 'externalID'),
('trailer', 'fleetCodeId', 'fleetCode.id'),
('trailer', 'ownerWorkerId', 'ownerWorker.id'),
('trailer', 'registrationStateId', 'registrationStateID'),
('worker', 'externalId', 'externalID'),
('worker', 'fleetCodeId', 'fleetCode.id'),
('worker', 'managerId', 'managerID'),
('worker', 'positionId', 'position.id'),
('worker', 'profilePicUrl', 'profilePicURL'),
('worker', 'stateId', 'stateID'),
('worker', 'userId', 'userID'),
('worker_credential', 'archivedById', 'archivedByID'),
('worker_credential', 'credentialTypeId', 'credentialType.id'),
('worker_credential', 'documentId', 'documentID'),
('worker_credential', 'verifiedById', 'verifiedByID'),
('worker_credential', 'workerId', 'worker.id'),
('worker_employment_event', 'amendedById', 'amendedByID'),
('worker_employment_event', 'documentId', 'documentID'),
('worker_employment_event', 'recordedById', 'recordedByID'),
('worker_employment_event', 'workerId', 'worker.id'),
('worker_profile', 'lastMvrCheck', 'lastMVRCheck'),
('worker_profile', 'licenseStateId', 'licenseStateID'),
('worker_profile', 'medicalExaminerNpi', 'medicalExaminerNPI'),
('worker_profile', 'workerId', 'workerID'),
('worker_pto', 'approverId', 'approverID'),
('worker_pto', 'cancelledById', 'cancelledByID'),
('worker_pto', 'rejectorId', 'rejectorID'),
('worker_pto', 'workerId', 'worker.id'),
('worker_pto_balance', 'accruedYtdDays', 'accruedYTDDays'),
('worker_pto_balance', 'usedYtdDays', 'usedYTDDays'),
('worker_pto_balance', 'workerId', 'worker.id');

INSERT INTO "rpt_edge"("entity", "old", "new", "target") VALUES
('assignment', 'primaryWorker', 'primaryWorker', 'worker'),
('assignment', 'secondaryWorker', 'secondaryWorker', 'worker'),
('assignment', 'tractor', 'tractor', 'tractor'),
('assignment', 'trailer', 'trailer', 'trailer'),
('bank_receipt', 'matchedCustomerPayment', 'matchedCustomerPayment', 'customer_payment'),
('commodity', 'hazardousMaterial', 'hazardousMaterial', 'hazardous_material'),
('customer_ledger_entry', 'customer', 'customer', 'customer'),
('customer_ledger_entry', 'relatedInvoice', 'relatedInvoice', 'invoice'),
('customer_payment', 'applications', 'applications', 'customer_payment_application'),
('customer_payment', 'customer', 'customer', 'customer'),
('customer_payment_application', 'invoice', 'invoice', 'invoice'),
('customer_payment_application', 'payment', 'payment', 'customer_payment'),
('driver_pay_event', 'settlement', 'settlement', 'driver_settlement'),
('driver_pay_event', 'shipment', 'shipment', 'shipment'),
('driver_pay_event', 'worker', 'worker', 'worker'),
('driver_settlement', 'batch', 'batch', 'driver_settlement_batch'),
('driver_settlement', 'lines', 'lines', 'driver_settlement_line'),
('driver_settlement', 'worker', 'worker', 'worker'),
('driver_settlement_batch', 'settlements', 'settlements', 'driver_settlement'),
('driver_settlement_line', 'payCode', 'payCode', 'pay_code'),
('driver_settlement_line', 'settlement', 'settlement', 'driver_settlement'),
('driver_settlement_line', 'shipment', 'shipment', 'shipment'),
('escrow_account', 'latestTransaction', 'latestTransaction', 'escrow_transaction'),
('escrow_account', 'transactions', 'transactions', 'escrow_transaction'),
('escrow_account', 'worker', 'worker', 'worker'),
('escrow_transaction', 'escrowAccount', 'escrowAccount', 'escrow_account'),
('factoring_agreement', 'factoringCompany', 'factoringCompany', 'factoring_company'),
('factoring_invoice_state', 'customer', 'customer', 'customer'),
('factoring_invoice_state', 'factoringCompany', 'factoringCompany', 'factoring_company'),
('factoring_invoice_state', 'invoice', 'invoice', 'invoice'),
('factoring_noa', 'customer', 'customer', 'customer'),
('factoring_noa', 'factoringCompany', 'factoringCompany', 'factoring_company'),
('factoring_remittance', 'factoringCompany', 'factoringCompany', 'factoring_company'),
('factoring_reserve_entry', 'customer', 'customer', 'customer'),
('factoring_reserve_entry', 'factoringCompany', 'factoringCompany', 'factoring_company'),
('factoring_reserve_entry', 'invoice', 'invoice', 'invoice'),
('factoring_reserve_entry', 'submission', 'submission', 'factoring_submission'),
('factoring_submission', 'agreement', 'agreement', 'factoring_agreement'),
('factoring_submission', 'factoringCompany', 'factoringCompany', 'factoring_company'),
('factoring_submission_item', 'customer', 'customer', 'customer'),
('factoring_submission_item', 'invoice', 'invoice', 'invoice'),
('fiscal_year', 'periods', 'periods', 'fiscal_period'),
('fuel_card', 'assignedTractor', 'assignedTractor', 'tractor'),
('fuel_card', 'assignedWorker', 'assignedWorker', 'worker'),
('fuel_purchase', 'fuelCard', 'fuelCard', 'fuel_card'),
('fuel_purchase', 'tractor', 'tractor', 'tractor'),
('fuel_purchase', 'worker', 'worker', 'worker'),
('gl_account', 'accountType', 'accountType', 'account_type'),
('gl_account', 'children', 'children', 'gl_account'),
('gl_account', 'parent', 'parent', 'gl_account'),
('gl_account_balance', 'fiscalPeriod', 'fiscalPeriod', 'fiscal_period'),
('gl_account_balance', 'fiscalYear', 'fiscalYear', 'fiscal_year'),
('gl_account_balance', 'glAccount', 'glAccount', 'gl_account'),
('gl_account_balance', 'lastJournalEntry', 'lastJournalEntry', 'journal_entry'),
('ifta_jurisdiction_mileage_entry', 'tractor', 'tractor', 'tractor'),
('ifta_return', 'lines', 'lines', 'ifta_return_line'),
('invoice', 'customer', 'customer', 'customer'),
('invoice', 'lines', 'lines', 'invoice_line'),
('invoice', 'order', 'order', 'order'),
('invoice', 'shipment', 'shipment', 'shipment'),
('invoice_adjustment', 'lines', 'lines', 'invoice_adjustment_line'),
('job_position', 'reportsTo', 'reportsTo', 'job_position'),
('journal_batch', 'entries', 'entries', 'journal_entry'),
('journal_batch', 'fiscalPeriod', 'fiscalPeriod', 'fiscal_period'),
('journal_batch', 'fiscalYear', 'fiscalYear', 'fiscal_year'),
('journal_entry', 'batch', 'batch', 'journal_batch'),
('journal_entry', 'fiscalPeriod', 'fiscalPeriod', 'fiscal_period'),
('journal_entry', 'fiscalYear', 'fiscalYear', 'fiscal_year'),
('journal_entry', 'lines', 'lines', 'journal_entry_line'),
('journal_entry', 'reversalOf', 'reversalOf', 'journal_entry'),
('journal_entry', 'reversedBy', 'reversedBy', 'journal_entry'),
('journal_entry_line', 'customer', 'customer', 'customer'),
('journal_entry_line', 'glAccount', 'glAccount', 'gl_account'),
('journal_entry_line', 'journalEntry', 'journalEntry', 'journal_entry'),
('journal_entry_line', 'location', 'location', 'location'),
('journal_source', 'journalBatch', 'journalBatch', 'journal_batch'),
('journal_source', 'journalEntry', 'journalEntry', 'journal_entry'),
('location', 'locationCategory', 'locationCategory', 'location_category'),
('order', 'customer', 'customer', 'customer'),
('order', 'shipments', 'shipments', 'shipment'),
('pay_advance', 'worker', 'worker', 'worker'),
('pay_code', 'glAccount', 'glAccount', 'gl_account'),
('rate_agreement', 'accessorials', 'accessorials', 'rate_agreement_accessorial'),
('rate_agreement', 'carrier', 'carrier', 'carrier'),
('rate_agreement', 'currentVersion', 'currentVersion', 'rate_agreement_version'),
('rate_agreement', 'customer', 'customer', 'customer'),
('rate_agreement', 'rules', 'rules', 'rate_agreement_rule'),
('rate_agreement', 'versions', 'versions', 'rate_agreement_version'),
('rate_agreement_accessorial', 'accessorialCharge', 'accessorialCharge', 'accessorial_charge'),
('rate_quote', 'agreement', 'agreement', 'rate_agreement'),
('recurring_deduction', 'escrowAccount', 'escrowAccount', 'escrow_account'),
('recurring_deduction', 'payCode', 'payCode', 'pay_code'),
('recurring_deduction', 'worker', 'worker', 'worker'),
('recurring_earning', 'payCode', 'payCode', 'pay_code'),
('recurring_earning', 'worker', 'worker', 'worker'),
('report_run', 'reportDefinition', 'reportDefinition', 'report_definition'),
('shipment', 'commodities', 'commodities', 'shipment_commodity'),
('shipment', 'customer', 'customer', 'customer'),
('shipment', 'destinationStop', 'destinationStop', 'stop'),
('shipment', 'moves', 'moves', 'shipment_move'),
('shipment', 'originStop', 'originStop', 'stop'),
('shipment', 'serviceType', 'serviceType', 'service_type'),
('shipment', 'shipmentType', 'shipmentType', 'shipment_type'),
('shipment', 'tractorType', 'tractorType', 'equipment_type'),
('shipment', 'trailerType', 'trailerType', 'equipment_type'),
('shipment_commodity', 'commodity', 'commodity', 'commodity'),
('shipment_commodity', 'shipment', 'shipment', 'shipment'),
('shipment_move', 'assignment', 'assignment', 'assignment'),
('shipment_move', 'shipment', 'shipment', 'shipment'),
('shipment_move', 'stops', 'stops', 'stop'),
('stop', 'location', 'location', 'location'),
('tractor', 'equipmentManufacturer', 'equipmentManufacturer', 'equipment_manufacturer'),
('tractor', 'equipmentType', 'equipmentType', 'equipment_type'),
('tractor', 'fleetCode', 'fleetCode', 'fleet_code'),
('tractor', 'primaryWorker', 'primaryWorker', 'worker'),
('tractor', 'secondaryWorker', 'secondaryWorker', 'worker'),
('trailer', 'equipmentManufacturer', 'equipmentManufacturer', 'equipment_manufacturer'),
('trailer', 'equipmentType', 'equipmentType', 'equipment_type'),
('trailer', 'fleetCode', 'fleetCode', 'fleet_code'),
('worker', 'fleetCode', 'fleetCode', 'fleet_code'),
('worker', 'position', 'position', 'job_position'),
('worker', 'profile', 'profile', 'worker_profile'),
('worker', 'pto', 'pto', 'worker_pto'),
('worker_credential', 'credentialType', 'credentialType', 'worker_credential_type'),
('worker_credential', 'worker', 'worker', 'worker'),
('worker_employment_event', 'worker', 'worker', 'worker'),
('worker_pto', 'worker', 'worker', 'worker'),
('worker_pto_balance', 'worker', 'worker', 'worker');

CREATE OR REPLACE FUNCTION pg_temp.rpt_entity_key(text)
    RETURNS text
    LANGUAGE sql
    STABLE
AS $$
    SELECT coalesce((SELECT e."new" FROM "rpt_entity" e WHERE e."old" = $1), $1)
$$;

CREATE OR REPLACE FUNCTION pg_temp.rpt_ref(ref jsonb)
    RETURNS text
    LANGUAGE sql
    IMMUTABLE
AS $$
    SELECT CASE
        WHEN coalesce(ref ->> 'field', '') = '' THEN ''
        ELSE concat_ws('.', VARIADIC array(SELECT jsonb_array_elements_text(coalesce(ref -> 'path', '[]'::jsonb))) || (ref ->> 'field'))
    END
$$;

CREATE OR REPLACE FUNCTION pg_temp.rpt_path(p_entity text, p_path text)
    RETURNS text
    LANGUAGE plpgsql
    STABLE
AS $$
DECLARE
    parts text[] := string_to_array(p_path, '.');
    cur_entity text := p_entity;
    hop record;
    mapped text[] := '{}';
    leaf_name text;
    i integer;
BEGIN
    IF p_path IS NULL OR p_path = '' THEN
        RETURN p_path;
    END IF;
    FOR i IN 1 .. array_length(parts, 1) - 1 LOOP
        SELECT e."new" AS name, e."target" AS target INTO hop
        FROM "rpt_edge" e
        WHERE e."entity" = cur_entity AND e."old" = parts[i];
        IF NOT FOUND THEN
            RETURN p_path;
        END IF;
        mapped := mapped || hop.name;
        cur_entity := hop.target;
    END LOOP;
    SELECT f."new" INTO leaf_name
    FROM "rpt_field" f
    WHERE f."entity" = cur_entity AND f."old" = parts[array_length(parts, 1)];
    IF NOT FOUND THEN
        leaf_name := parts[array_length(parts, 1)];
    END IF;
    RETURN array_to_string(mapped || leaf_name, '.');
END
$$;

CREATE OR REPLACE FUNCTION pg_temp.rpt_op(op text)
    RETURNS text
    LANGUAGE sql
    IMMUTABLE
AS $$
    SELECT CASE op
        WHEN 'startswith' THEN 'starts_with'
        WHEN 'endswith' THEN 'ends_with'
        WHEN 'notin' THEN 'not_in'
        WHEN 'isnull' THEN 'is_null'
        WHEN 'isnotnull' THEN 'is_not_null'
        WHEN 'daterange' THEN 'between'
        WHEN 'lastndays' THEN 'last_n_days'
        WHEN 'nextndays' THEN 'next_n_days'
        WHEN 'thisweek' THEN 'this_week'
        WHEN 'lastweek' THEN 'last_week'
        WHEN 'thismonth' THEN 'this_month'
        WHEN 'lastmonth' THEN 'last_month'
        WHEN 'thisquarter' THEN 'this_quarter'
        WHEN 'lastquarter' THEN 'last_quarter'
        WHEN 'thisyear' THEN 'this_year'
        WHEN 'lastyear' THEN 'last_year'
        ELSE op
    END
$$;

CREATE OR REPLACE FUNCTION pg_temp.rpt_filter(entity text, grp jsonb)
    RETURNS jsonb
    LANGUAGE plpgsql
    STABLE
AS $$
DECLARE
    children jsonb := '[]'::jsonb;
    leaf jsonb;
    node jsonb;
    nested jsonb;
BEGIN
    IF grp IS NULL OR jsonb_typeof(grp) <> 'object'
        OR (coalesce(jsonb_array_length(grp -> 'filters'), 0) = 0 AND coalesce(jsonb_array_length(grp -> 'groups'), 0) = 0) THEN
        RETURN NULL;
    END IF;
    FOR leaf IN SELECT value FROM jsonb_array_elements(coalesce(grp -> 'filters', '[]'::jsonb)) LOOP
        node := jsonb_build_object(
            'field', pg_temp.rpt_path(entity, pg_temp.rpt_ref(leaf -> 'ref')),
            'op', pg_temp.rpt_op(leaf ->> 'operator'));
        IF leaf ? 'value' AND leaf -> 'value' <> 'null'::jsonb THEN
            node := node || jsonb_build_object('value', leaf -> 'value');
        END IF;
        IF coalesce(leaf ->> 'param', '') <> '' THEN
            node := node || jsonb_build_object('param', leaf -> 'param');
        END IF;
        IF leaf ? 'transform' AND leaf -> 'transform' <> 'null'::jsonb THEN
            node := node || jsonb_build_object('transform', leaf -> 'transform');
        END IF;
        children := children || jsonb_build_array(node);
    END LOOP;
    FOR leaf IN SELECT value FROM jsonb_array_elements(coalesce(grp -> 'groups', '[]'::jsonb)) LOOP
        nested := pg_temp.rpt_filter(entity, leaf);
        IF nested IS NOT NULL THEN
            children := children || jsonb_build_array(nested);
        END IF;
    END LOOP;
    IF grp ->> 'op' = 'or' THEN
        RETURN jsonb_build_object('or', children);
    END IF;
    RETURN jsonb_build_object('and', children);
END
$$;

CREATE OR REPLACE FUNCTION pg_temp.rpt_number(value jsonb)
    RETURNS text
    LANGUAGE sql
    IMMUTABLE
AS $$
    SELECT CASE
        WHEN (value #>> '{}')::numeric < 0 THEN '-' || trim_scale(abs((value #>> '{}')::numeric))::text
        ELSE trim_scale((value #>> '{}')::numeric)::text
    END
$$;

CREATE OR REPLACE FUNCTION pg_temp.rpt_operand(id text, value jsonb)
    RETURNS text
    LANGUAGE sql
    IMMUTABLE
AS $$
    SELECT CASE
        WHEN coalesce(id, '') <> '' THEN '[' || id || ']'
        ELSE pg_temp.rpt_number(value)
    END
$$;

CREATE OR REPLACE FUNCTION pg_temp.rpt_allowed(kind text, value text)
    RETURNS jsonb
    LANGUAGE plpgsql
    IMMUTABLE
AS $$
BEGIN
    IF kind IN ('int', 'decimal', 'float') AND value ~ '^-?[0-9]+(\.[0-9]+)?([eE][-+]?[0-9]+)?$' THEN
        RETURN to_jsonb(value::numeric);
    END IF;
    IF kind = 'bool' AND value IN ('1', 't', 'T', 'TRUE', 'true', 'True', '0', 'f', 'F', 'FALSE', 'false', 'False') THEN
        RETURN to_jsonb(value::boolean);
    END IF;
    RETURN to_jsonb(value);
END
$$;

CREATE OR REPLACE FUNCTION pg_temp.rpt_parameter(p jsonb)
    RETURNS jsonb
    LANGUAGE plpgsql
    STABLE
AS $$
DECLARE
    res jsonb := jsonb_build_object('name', p -> 'name', 'type', p -> 'type');
    allowed jsonb := '[]'::jsonb;
    item text;
BEGIN
    IF coalesce(p ->> 'label', '') <> '' THEN
        res := res || jsonb_build_object('label', p -> 'label');
    END IF;
    IF coalesce((p ->> 'required')::boolean, false) THEN
        res := res || jsonb_build_object('required', true);
    END IF;
    IF coalesce((p ->> 'multi')::boolean, false) THEN
        res := res || jsonb_build_object('multi', true);
    END IF;
    IF p ? 'default' AND p -> 'default' <> 'null'::jsonb THEN
        res := res || jsonb_build_object('default', p -> 'default');
    END IF;
    IF jsonb_typeof(p -> 'allowedValues') = 'array' AND jsonb_array_length(p -> 'allowedValues') > 0 THEN
        FOR item IN SELECT jsonb_array_elements_text(p -> 'allowedValues') LOOP
            allowed := allowed || jsonb_build_array(pg_temp.rpt_allowed(p ->> 'type', item));
        END LOOP;
        res := res || jsonb_build_object('allowed', allowed);
    END IF;
    IF coalesce(p ->> 'refEntity', '') <> '' THEN
        res := res || jsonb_build_object('entity', pg_temp.rpt_entity_key(p ->> 'refEntity'));
    END IF;
    RETURN res;
END
$$;

CREATE OR REPLACE FUNCTION pg_temp.rpt_having(entity text, grp jsonb, columns jsonb, olds jsonb, INOUT extra jsonb, OUT node jsonb)
    LANGUAGE plpgsql
    STABLE
AS $$
DECLARE
    children jsonb := '[]'::jsonb;
    leaf jsonb;
    nested record;
    old_path text;
    agg text;
    column_id text;
    base text;
    n integer;
    all_columns jsonb;
    all_olds jsonb;
    i integer;
BEGIN
    FOR leaf IN SELECT value FROM jsonb_array_elements(coalesce(grp -> 'filters', '[]'::jsonb)) LOOP
        old_path := pg_temp.rpt_ref(leaf -> 'ref');
        agg := leaf ->> 'agg';
        column_id := NULL;
        all_columns := columns || (SELECT coalesce(jsonb_agg(value -> 'column'), '[]'::jsonb) FROM jsonb_array_elements(extra));
        all_olds := olds || (SELECT coalesce(jsonb_agg(value -> 'old'), '[]'::jsonb) FROM jsonb_array_elements(extra));
        FOR i IN 0 .. jsonb_array_length(all_columns) - 1 LOOP
            IF all_columns -> i -> 'measure' IS NOT NULL
                AND all_columns -> i -> 'measure' ->> 'agg' = agg
                AND all_olds ->> i = old_path
                AND (all_columns -> i -> 'measure' -> 'filter') IS NULL
                AND (all_columns -> i -> 'transform') IS NULL THEN
                column_id := all_columns -> i ->> 'id';
                EXIT;
            END IF;
        END LOOP;
        IF column_id IS NULL THEN
            base := 'having_' || agg || '_' || replace(old_path, '.', '_');
            column_id := base;
            n := 2;
            WHILE EXISTS (SELECT 1 FROM jsonb_array_elements(all_columns) c WHERE c ->> 'id' = column_id) LOOP
                column_id := base || '_' || n;
                n := n + 1;
            END LOOP;
            extra := extra || jsonb_build_array(jsonb_build_object(
                'old', old_path,
                'column', jsonb_build_object(
                    'id', column_id,
                    'hidden', true,
                    'measure', jsonb_build_object('field', pg_temp.rpt_path(entity, old_path), 'agg', agg))));
        END IF;
        node := jsonb_build_object('column', column_id, 'op', pg_temp.rpt_op(leaf ->> 'operator'));
        IF leaf ? 'value' AND leaf -> 'value' <> 'null'::jsonb THEN
            node := node || jsonb_build_object('value', leaf -> 'value');
        END IF;
        IF coalesce(leaf ->> 'param', '') <> '' THEN
            node := node || jsonb_build_object('param', leaf -> 'param');
        END IF;
        children := children || jsonb_build_array(node);
    END LOOP;
    FOR leaf IN SELECT value FROM jsonb_array_elements(coalesce(grp -> 'groups', '[]'::jsonb)) LOOP
        IF coalesce(jsonb_array_length(leaf -> 'filters'), 0) = 0 AND coalesce(jsonb_array_length(leaf -> 'groups'), 0) = 0 THEN
            CONTINUE;
        END IF;
        SELECT * INTO nested FROM pg_temp.rpt_having(entity, leaf, columns, olds, extra);
        extra := nested.extra;
        children := children || jsonb_build_array(nested.node);
    END LOOP;
    IF grp ->> 'op' = 'or' THEN
        node := jsonb_build_object('or', children);
    ELSE
        node := jsonb_build_object('and', children);
    END IF;
END
$$;

CREATE OR REPLACE FUNCTION pg_temp.rpt_chart(c jsonb)
    RETURNS jsonb
    LANGUAGE plpgsql
    IMMUTABLE
AS $$
DECLARE
    res jsonb := jsonb_build_object('id', c -> 'id', 'type', c -> 'type');
BEGIN
    IF coalesce(c ->> 'title', '') <> '' THEN res := res || jsonb_build_object('title', c -> 'title'); END IF;
    IF coalesce(c ->> 'xColumnId', '') <> '' THEN res := res || jsonb_build_object('x', c -> 'xColumnId'); END IF;
    IF jsonb_typeof(c -> 'seriesIds') = 'array' AND jsonb_array_length(c -> 'seriesIds') > 0 THEN
        res := res || jsonb_build_object('series', c -> 'seriesIds');
    END IF;
    IF coalesce((c ->> 'stacked')::boolean, false) AND c ->> 'type' IN ('bar', 'hbar', 'area') THEN
        res := res || jsonb_build_object('stacked', true);
    END IF;
    IF coalesce((c ->> 'hideLegend')::boolean, false) THEN res := res || jsonb_build_object('hideLegend', true); END IF;
    IF coalesce((c ->> 'showValues')::boolean, false) THEN res := res || jsonb_build_object('showValues', true); END IF;
    IF coalesce((c ->> 'curved')::boolean, false) THEN res := res || jsonb_build_object('curved', true); END IF;
    IF coalesce((c ->> 'limit')::integer, 0) <> 0 THEN res := res || jsonb_build_object('limit', c -> 'limit'); END IF;
    IF coalesce(c ->> 'compareId', '') <> '' THEN res := res || jsonb_build_object('compare', c -> 'compareId'); END IF;
    IF coalesce(c ->> 'latColumnId', '') <> '' THEN res := res || jsonb_build_object('lat', c -> 'latColumnId'); END IF;
    IF coalesce(c ->> 'lngColumnId', '') <> '' THEN res := res || jsonb_build_object('lng', c -> 'lngColumnId'); END IF;
    IF coalesce(c ->> 'labelColumnId', '') <> '' THEN res := res || jsonb_build_object('pointLabel', c -> 'labelColumnId'); END IF;
    IF jsonb_typeof(c -> 'goal') = 'object' AND (c -> 'goal' ? 'value' OR coalesce(c -> 'goal' ->> 'columnId', '') <> '') THEN
        res := res || jsonb_build_object('goal', jsonb_strip_nulls(jsonb_build_object(
            'value', c -> 'goal' -> 'value',
            'column', nullif(c -> 'goal' ->> 'columnId', ''),
            'label', nullif(c -> 'goal' ->> 'label', ''))));
    END IF;
    RETURN res;
END
$$;

CREATE OR REPLACE FUNCTION pg_temp.rpt_definition(def jsonb)
    RETURNS jsonb
    LANGUAGE plpgsql
    STABLE
AS $$
DECLARE
    entity text := def ->> 'entity';
    columns jsonb := '[]'::jsonb;
    olds jsonb := '[]'::jsonb;
    col jsonb;
    res jsonb;
    calc text;
    op text;
    fmt text;
    path text;
    measure jsonb;
    dim jsonb;
    cond jsonb;
    hv record;
    extra jsonb;
    result jsonb;
BEGIN
    IF def IS NULL OR def ? 'version' OR NOT def ? 'irVersion' THEN
        RETURN def;
    END IF;

    FOR col IN SELECT value FROM jsonb_array_elements(coalesce(def -> 'columns', '[]'::jsonb)) LOOP
        res := jsonb_build_object('id', col -> 'id');
        IF coalesce(col ->> 'label', '') <> '' THEN
            res := res || jsonb_build_object('label', col -> 'label');
        END IF;
        path := pg_temp.rpt_ref(col -> 'ref');
        CASE col ->> 'kind'
            WHEN 'dimension' THEN
                dim := jsonb_build_object('field', pg_temp.rpt_path(entity, path));
                IF coalesce(col ->> 'bucket', '') <> '' THEN
                    dim := dim || jsonb_build_object('bucket', col -> 'bucket');
                END IF;
                IF jsonb_typeof(col -> 'band') = 'object' THEN
                    dim := dim || jsonb_build_object('band', col -> 'band');
                END IF;
                res := res || jsonb_build_object('dimension', dim);
            WHEN 'measure' THEN
                measure := jsonb_build_object('agg', col -> 'agg');
                IF path <> '' THEN
                    measure := measure || jsonb_build_object('field', pg_temp.rpt_path(entity, path));
                END IF;
                cond := pg_temp.rpt_filter(entity, col -> 'filter');
                IF cond IS NOT NULL THEN
                    measure := measure || jsonb_build_object('filter', cond);
                END IF;
                res := res || jsonb_build_object('measure', measure);
            WHEN 'computed' THEN
                op := CASE col -> 'computed' ->> 'op'
                    WHEN 'add' THEN '+' WHEN 'subtract' THEN '-' WHEN 'multiply' THEN '*' WHEN 'divide' THEN '/'
                END;
                calc := pg_temp.rpt_operand(col -> 'computed' ->> 'leftId', col -> 'computed' -> 'leftValue')
                    || ' ' || op || ' '
                    || pg_temp.rpt_operand(col -> 'computed' ->> 'rightId', col -> 'computed' -> 'rightValue');
                res := res || jsonb_build_object('calc', calc);
                fmt := col -> 'computed' ->> 'format';
                IF coalesce(fmt, '') NOT IN ('', 'none') THEN
                    res := res || jsonb_build_object('format', fmt);
                END IF;
            ELSE
                RAISE EXCEPTION 'report column kind % has no keel equivalent', col ->> 'kind';
        END CASE;
        IF jsonb_typeof(col -> 'transform') = 'object' THEN
            res := res || jsonb_build_object('transform', col -> 'transform');
        END IF;
        IF jsonb_typeof(col -> 'display') = 'object' THEN
            res := res || jsonb_build_object('display', col -> 'display');
        END IF;
        columns := columns || jsonb_build_array(res);
        olds := olds || jsonb_build_array(CASE WHEN col ->> 'kind' = 'measure' THEN path END);
    END LOOP;

    result := jsonb_build_object('version', 2, 'entity', pg_temp.rpt_entity_key(entity));

    cond := pg_temp.rpt_filter(entity, def -> 'filters');
    IF cond IS NOT NULL THEN
        result := result || jsonb_build_object('filters', cond);
    END IF;

    IF jsonb_typeof(def -> 'having') = 'object'
        AND (coalesce(jsonb_array_length(def -> 'having' -> 'filters'), 0) > 0
             OR coalesce(jsonb_array_length(def -> 'having' -> 'groups'), 0) > 0) THEN
        SELECT * INTO hv FROM pg_temp.rpt_having(entity, def -> 'having', columns, olds, '[]'::jsonb);
        result := result || jsonb_build_object('having', hv.node);
        extra := hv.extra;
        columns := columns || (SELECT coalesce(jsonb_agg(value -> 'column'), '[]'::jsonb) FROM jsonb_array_elements(extra));
    END IF;
    result := result || jsonb_build_object('columns', columns);

    IF coalesce((def ->> 'totals')::boolean, false) THEN
        result := result || jsonb_build_object('groupings', jsonb_build_object('total', true));
    END IF;
    IF jsonb_typeof(def -> 'sort') = 'array' AND jsonb_array_length(def -> 'sort') > 0 THEN
        result := result || jsonb_build_object('sort', (
            SELECT jsonb_agg(CASE WHEN s ->> 'direction' = 'desc'
                THEN jsonb_build_object('column', s -> 'columnId', 'desc', true)
                ELSE jsonb_build_object('column', s -> 'columnId') END ORDER BY ord)
            FROM jsonb_array_elements(def -> 'sort') WITH ORDINALITY AS t(s, ord)));
    END IF;
    IF coalesce((def ->> 'limit')::integer, 0) <> 0 THEN
        result := result || jsonb_build_object('limit', (def ->> 'limit')::integer);
    END IF;
    IF jsonb_typeof(def -> 'pivot') = 'object' THEN
        result := result || jsonb_build_object('pivot', jsonb_strip_nulls(jsonb_build_object(
            'field', pg_temp.rpt_path(entity, pg_temp.rpt_ref(def -> 'pivot' -> 'ref')),
            'values', coalesce(nullif(def -> 'pivot' -> 'values', 'null'::jsonb), '[]'::jsonb),
            'labels', CASE WHEN jsonb_typeof(def -> 'pivot' -> 'labels') = 'array' AND jsonb_array_length(def -> 'pivot' -> 'labels') > 0
                THEN def -> 'pivot' -> 'labels' END,
            'columns', def -> 'pivot' -> 'measureIds',
            'other', CASE WHEN coalesce((def -> 'pivot' ->> 'includeOther')::boolean, false) THEN true END)));
    END IF;
    IF jsonb_typeof(def -> 'parameters') = 'array' AND jsonb_array_length(def -> 'parameters') > 0 THEN
        result := result || jsonb_build_object('parameters', (
            SELECT jsonb_agg(pg_temp.rpt_parameter(p) ORDER BY ord)
            FROM jsonb_array_elements(def -> 'parameters') WITH ORDINALITY AS t(p, ord)));
    END IF;
    IF jsonb_typeof(def -> 'charts') = 'array' AND jsonb_array_length(def -> 'charts') > 0 THEN
        result := result || jsonb_build_object('charts', (
            SELECT jsonb_agg(pg_temp.rpt_chart(c) ORDER BY ord)
            FROM jsonb_array_elements(def -> 'charts') WITH ORDINALITY AS t(c, ord)));
    END IF;
    RETURN result;
END
$$;

CREATE OR REPLACE FUNCTION pg_temp.rpt_layout(layout jsonb)
    RETURNS jsonb
    LANGUAGE plpgsql
    STABLE
AS $$
DECLARE
    filters jsonb;
    params jsonb;
BEGIN
    IF layout IS NULL OR jsonb_typeof(layout) <> 'object' THEN
        RETURN layout;
    END IF;
    IF jsonb_typeof(layout -> 'filters') = 'array' THEN
        SELECT coalesce(jsonb_agg(CASE WHEN f ? 'ref' THEN jsonb_strip_nulls(jsonb_build_object(
                'id', f -> 'id',
                'label', nullif(f ->> 'label', ''),
                'entity', pg_temp.rpt_entity_key(f ->> 'entity'),
                'field', pg_temp.rpt_path(f ->> 'entity', pg_temp.rpt_ref(f -> 'ref')),
                'op', pg_temp.rpt_op(f ->> 'operator'),
                'default', f -> 'default'))
            ELSE f END ORDER BY ord), '[]'::jsonb)
        INTO filters
        FROM jsonb_array_elements(layout -> 'filters') WITH ORDINALITY AS t(f, ord);
        layout := layout || jsonb_build_object('filters', filters);
    END IF;
    IF jsonb_typeof(layout -> 'parameters') = 'array' THEN
        SELECT coalesce(jsonb_agg(CASE WHEN p ? 'allowedValues' OR p ? 'refEntity' OR p ? 'required'
                THEN pg_temp.rpt_parameter(p) ELSE p END ORDER BY ord), '[]'::jsonb)
        INTO params
        FROM jsonb_array_elements(layout -> 'parameters') WITH ORDINALITY AS t(p, ord);
        layout := layout || jsonb_build_object('parameters', params);
    END IF;
    RETURN layout;
END
$$;

UPDATE "report_definitions"
SET "definition" = pg_temp.rpt_definition("definition"),
    "catalog_version" = ''
WHERE "definition" ? 'irVersion';

UPDATE "report_definition_revisions"
SET "definition" = pg_temp.rpt_definition("definition"),
    "catalog_version" = ''
WHERE "definition" ? 'irVersion';

UPDATE "report_dashboards"
SET "layout" = pg_temp.rpt_layout("layout")
WHERE jsonb_typeof("layout" -> 'filters') = 'array' OR jsonb_typeof("layout" -> 'parameters') = 'array';
