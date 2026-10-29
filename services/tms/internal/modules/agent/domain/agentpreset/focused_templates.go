package agentpreset

const (
	toolDescribeReport     = "report_get"
	toolGetReportRun       = "report_get_run"
	toolListReports        = "report_list"
	toolPreviewReport      = "report_preview"
	toolRunReport          = "report_run"
	toolGetCustomer        = "customer_get"
	toolGetDocumentSummary = "document_get_summary"
	toolShipmentGet        = "shipment_get"
	toolGetWorker          = "worker_get"
	toolListCommodities    = "masterdata_list_commodities"
	toolListCustomers      = "customer_list"
	toolListTimeOff        = "worker_list_time_off"
	toolListTractors       = "equipment_list_tractors"
	toolListTrailers       = "equipment_list_trailers"
	toolListWorkers        = "worker_list"
	toolShipmentList       = "shipment_list"

	toolShipmentAddComment  = "shipment_add_comment"
	toolShipmentGetTracking = "shipment_get_tracking"
	toolIntakeGetDraft      = "intake_get_draft"
)

func masterDataStewardTools() []string {
	return []string{
		toolListCarriers,
		toolGetCarrier,
		"carrier_create",
		"carrier_update",
		"carrier_set_status",
		toolListCustomers,
		toolGetCustomer,
		"customer_create",
		"customer_update",
		"customer_set_status",
		toolListCommodities,
		ToolCreateCommodity,
		"masterdata_update_commodity",
		"masterdata_set_commodity_status",
		"masterdata_list_hazardous_materials",
		"masterdata_create_hazardous_material",
		"masterdata_update_hazardous_material",
		"masterdata_set_hazardous_material_status",
		toolListLocations,
		"location_get",
		"location_list_categories",
		"location_create",
		"location_update",
		"location_set_status",
		toolListTractors,
		"equipment_get_tractor",
		"equipment_create_tractor",
		"equipment_update_tractor",
		"equipment_set_tractor_status",
		"equipment_set_tractor_location",
		toolListTrailers,
		"equipment_get_trailer",
		"equipment_create_trailer",
		"equipment_update_trailer",
		"equipment_set_trailer_status",
		"equipment_set_trailer_location",
		"equipment_list_types",
		"equipment_list_manufacturers",
		"masterdata_list_fleet_codes",
		toolShipmentList,
		toolSearchDocuments,
		toolGetDocumentSummary,
		"document_list_types",
		"document_delete",
		"document_restore_version",
		"inbox_list_capture_batches",
		"inbox_file_capture_item",
		"inbox_discard_capture_item",
		"inbox_discard_capture_batch",
		"report_list_table_change_alerts",
		"report_create_table_change_alert",
		"report_update_table_change_alert",
		"report_set_table_change_alert_status",
		"report_delete_table_change_alert",
		"exception_list_watchtower_items",
		"exception_dismiss_watchtower_item",
	}
}

func workforceCoordinatorTools() []string {
	return []string{
		toolListWorkers,
		toolGetWorker,
		toolShipmentList,
		toolListTimeOff,
		"worker_request_pto",
		"worker_update_pto",
		"worker_approve_pto",
		"worker_reject_pto",
		"worker_cancel_pto",
		"worker_adjust_pto_balance",
		"worker_list_leave_cases",
		"worker_open_leave_case",
		"worker_update_leave_case",
		"worker_request_leave_certification",
		"worker_record_leave_day",
		"worker_update_leave_day",
		"worker_delete_leave_day",
		"worker_close_leave_case",
		"worker_list_injuries",
		"worker_record_injury",
		"worker_update_injury",
		"worker_delete_injury",
		"worker_list_performance_reviews",
		"worker_start_performance_review",
		"worker_update_performance_review",
		"worker_delete_performance_review",
		"worker_record_recognition",
		"worker_delete_recognition",
		"worker_list_safety_events",
		"worker_update_safety_event",
		"worker_delete_safety_event",
		"worker_update_safety_violation",
		"worker_delete_safety_violation",
		"worker_list_dot_tests",
		"worker_cancel_dot_test",
		"worker_list_dot_random_draws",
		"worker_get_dot_random_draw",
		"worker_run_dot_random_draw",
		"worker_finalize_dot_random_draw",
		"worker_cancel_dot_random_draw",
		"worker_list_checklists",
		"worker_start_checklist",
		"worker_update_checklist_item",
		"worker_cancel_checklist",
		"worker_list_training",
		"worker_attach_training_document",
		"worker_close_training",
		"worker_list_credentials",
		"worker_archive_credential",
		"worker_list_employment_verifications",
		"worker_delete_employment_verification",
		"shipment_list_permits",
		"shipment_record_permit",
		"shipment_update_permit",
	}
}

func fuelTaxClerkTools() []string {
	return []string{
		toolListTractors,
		"equipment_get_tractor",
		toolListWorkers,
		toolShipmentList,
		toolShipmentGet,
		toolSearchDocuments,
		toolGetDocumentSummary,
		"fuel_list_ifta_jurisdictions",
		"fuel_list_purchases",
		"fuel_record_purchase",
		"fuel_correct_purchase",
		"fuel_delete_purchase",
		"fuel_list_cards",
		"fuel_assign_card",
		"fuel_list_purchase_imports",
		"fuel_resolve_purchase_import_rows",
		"fuel_commit_purchase_import",
		"fuel_discard_purchase_import",
		"fuel_list_ifta_mileage_entries",
		"fuel_record_ifta_mileage_entry",
		"fuel_correct_ifta_mileage_entry",
		"fuel_delete_ifta_mileage_entry",
		"fuel_recalculate_jurisdiction_miles",
		"fuel_backfill_jurisdiction_miles",
		"fuel_list_ifta_returns",
		"fuel_generate_ifta_return",
		"fuel_recalculate_ifta_return",
		"fuel_delete_ifta_return",
		"fuel_amend_ifta_return",
		"rate_list_fuel_index_prices",
		"rate_record_fuel_index_price",
		"rate_correct_fuel_index_price",
		"rate_list_fuel_surcharge_programs",
	}
}

func reportAnalystTools() []string {
	return []string{
		toolListReports,
		toolDescribeReport,
		toolPreviewReport,
		toolRunReport,
		toolGetReportRun,
		"report_list_runs",
		"report_compare_runs",
		"report_cancel_run",
		"report_list_datasets",
		"report_get_dataset",
		"report_create",
		"report_update",
		"report_fork",
		"report_reset_fork",
		"report_delete",
		"report_list_schedules",
		"report_create_schedule",
		"report_update_schedule",
		"report_delete_schedule",
		"report_list_dashboards",
		"report_create_dashboard",
		"report_add_dashboard_tile",
		"report_delete_dashboard",
	}
}
