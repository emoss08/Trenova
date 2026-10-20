package agenttoolpolicy_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy/registered"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/deskcase"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildRegistered(t *testing.T) registered.Tools {
	t.Helper()

	tools, err := registered.Build()
	require.NoError(t, err)

	return tools
}

func registeredPolicy(t *testing.T, name string) agentports.ToolPolicy {
	t.Helper()

	tools := buildRegistered(t)
	for _, tool := range tools.Actions {
		if tool.Name() == name {
			return tool.Policy()
		}
	}
	for _, tool := range tools.Queries {
		if tool.Name() == name {
			return tool.Policy()
		}
	}
	t.Fatalf("%q is not a registered tool", name)

	return agentports.ToolPolicy{}
}

/*
Every tool the system registers declares a policy the catalog accepts. The
catalog refuses to boot on anything else, so this is the same check made where
a failure costs a test rather than a deploy.
*/
func TestEveryRegisteredToolValidates(t *testing.T) {
	t.Parallel()

	tools := buildRegistered(t)
	catalog, err := agenttoolpolicy.Build(
		tools.Queries,
		tools.Actions,
		agentruntime.RuntimePolicies(),
	)
	require.NoError(t, err)

	total := len(tools.Queries) + len(tools.Actions) + len(agentruntime.RuntimePolicies())
	assert.Len(t, catalog.All(), total)
	for _, policy := range catalog.All() {
		found, ok := catalog.Get(policy.Name)
		require.True(t, ok, policy.Name)
		assert.Equal(t, policy.Name, found.Name)
	}
}

func TestEveryQueryToolChangesNothingAndSendsNothing(t *testing.T) {
	t.Parallel()

	for _, tool := range buildRegistered(t).Queries {
		policy := tool.Policy()
		assert.Equal(t, agent.ToolKindQuery, policy.Kind, tool.Name())
		assert.Equal(t, []agent.EgressClass{agent.EgressNone}, policy.Egress, tool.Name())
	}
}

// The tools whose results carry text someone outside the organization wrote
// say so, which is what lets a run that read them be held back.
func TestToolsThatReadOutsideTextSayWhere(t *testing.T) {
	t.Parallel()

	cases := map[string]agent.ExternalRead{
		"inbox_get_inbound_message":               agent.ExternalReadAlways,
		"inbox_list_inbound_messages":             agent.ExternalReadAlways,
		"document_list":                           agent.ExternalReadAlways,
		"document_get_summary":                    agent.ExternalReadAlways,
		"intake_get_draft":                        agent.ExternalReadAlways,
		"receivable_get_bank_receipt":             agent.ExternalReadAlways,
		"receivable_list_bank_receipt_exceptions": agent.ExternalReadAlways,
		"insight_list_weather_alerts":             agent.ExternalReadAlways,
		"agent_list_memories":                     agent.ExternalReadMarked,
		"agent_get_run":                           agent.ExternalReadMarked,
		"shipment_get":                            agent.ExternalReadMarked,
		"exception_list_watchtower_items":         agent.ExternalReadMarked,
		"agent_list_runs":                         agent.ExternalReadMarked,
		"insight_get_daily_briefing":              agent.ExternalReadNever,
		"customer_get":                            agent.ExternalReadNever,
		"edi_list_inbound_files":                  agent.ExternalReadAlways,
		"edi_get_inbound_file":                    agent.ExternalReadAlways,
		"edi_get_message":                         agent.ExternalReadAlways,
		"edi_list_transfers":                      agent.ExternalReadAlways,
		"edi_get_partner":                         agent.ExternalReadNever,
		"settlement_get_dispute":                  agent.ExternalReadMarked,
		"carrier_list_invoice_matches":            agent.ExternalReadMarked,
		"invoice_get":                             agent.ExternalReadNever,
		"receivable_get_aging":                    agent.ExternalReadNever,
		"settlement_get_driver":                   agent.ExternalReadNever,
		"rate_get_agreement":                      agent.ExternalReadNever,
		"order_get":                               agent.ExternalReadNever,
	}
	for name, want := range cases {
		policy := registeredPolicy(t, name)
		assert.Equal(t, want, policy.ReadsExternal, name)
		if want != agent.ExternalReadNever {
			assert.True(t, policy.Source.IsValid(), name)
		}
	}
	assert.Equal(t, agent.TaintSourceRecordNote, registeredPolicy(t, "shipment_get").Source)
	assert.Equal(t, agent.TaintSourceRecordNote,
		registeredPolicy(t, "settlement_get_dispute").Source)
	for _, name := range []string{
		"edi_list_inbound_files",
		"edi_get_inbound_file",
		"edi_get_message",
		"edi_list_transfers",
		"carrier_list_invoice_matches",
	} {
		assert.Equal(t, agent.TaintSourceEDI, registeredPolicy(t, name).Source, name)
	}
	assert.Equal(t, agent.TaintSourceRunRecord, registeredPolicy(t, "agent_list_runs").Source)
	watchtower := registeredPolicy(t, "exception_list_watchtower_items")
	assert.Equal(t, agent.TaintSourceInboundMessage, watchtower.Source)
	assert.ElementsMatch(t, []agent.TaintSource{
		agent.TaintSourceEDI,
		agent.TaintSourceWeather,
		agent.TaintSourceRunRecord,
	}, watchtower.Sources)
	assert.True(t, registeredPolicy(t, "agent_create_memory").CarriesTaint)
	assert.NotNil(t, registeredPolicy(t, "agent_create_memory").TaintHold)
	assert.True(t, registeredPolicy(t, "agent_guide_profile").CarriesTaint)
	assert.NotNil(t, registeredPolicy(t, "agent_guide_profile").TaintHold)
	assert.True(t, registeredPolicy(t, "shipment_add_comment").CarriesTaint)
}

func TestEveryRegisteredToolIsInTheClassItWasGiven(t *testing.T) {
	t.Parallel()

	cases := map[string][]agent.EgressClass{
		"home_add_widget":                     {agent.EgressPersonal},
		"home_remove_widget":                  {agent.EgressPersonal},
		"home_arrange_layout":                 {agent.EgressPersonal},
		"report_create":                       {agent.EgressPersonal, agent.EgressInternal},
		"report_save_table_view":              {agent.EgressPersonal, agent.EgressInternal},
		"billing_start_review":                {agent.EgressInternal},
		"billing_transfer":                    {agent.EgressInternal},
		"billing_return_to_operations":        {agent.EgressInternal},
		"billing_raise_exception":             {agent.EgressInternal},
		"billing_hold_queue_item":             {agent.EgressInternal},
		"billing_release_queue_item":          {agent.EgressInternal},
		"billing_resolve_issue":               {agent.EgressMoney, agent.EgressDriverVisible},
		"billing_undo_issue":                  {agent.EgressMoney},
		"billing_assign_biller":               {agent.EgressInternal},
		"billing_approve_queue_item":          {agent.EgressMoney},
		"billing_cancel_queue_item":           {agent.EgressMoney},
		"invoice_post":                        {agent.EgressMoney},
		"invoice_send":                        {agent.EgressExternalRecipient},
		"invoice_open_dispute":                {agent.EgressInternal},
		"invoice_resolve_dispute":             {agent.EgressInternal},
		"invoice_withdraw_dispute":            {agent.EgressInternal},
		"receivable_apply_customer_payment":   {agent.EgressMoney},
		"receivable_reverse_customer_payment": {agent.EgressMoney},
		"receivable_apply_credit_memo":        {agent.EgressMoney},
		"receivable_unapply_credit_memo":      {agent.EgressMoney},
		"invoice_save_adjustment_draft":       {agent.EgressInternal},
		"invoice_submit_adjustment":           {agent.EgressMoney},
		"invoice_approve_adjustment":          {agent.EgressMoney},
		"invoice_reject_adjustment":           {agent.EgressInternal},
		"invoice_build_run":                   {agent.EgressInternal},
		"invoice_adjust_run":                  {agent.EgressInternal},
		"invoice_commit_run":                  {agent.EgressMoney},
		"invoice_cancel_run":                  {agent.EgressInternal},
		"invoice_bill_statement":              {agent.EgressMoney},
		"receivable_assess_late_charges":      {agent.EgressMoney},
		"invoice_share":                       {agent.EgressInternal},
		"invoice_update_draft":                {agent.EgressInternal},
		"invoice_generate_pdf":                {agent.EgressInternal},
		"invoice_create":                      {agent.EgressMoney},
		"invoice_create_memo":                 {agent.EgressMoney},
		"invoice_void":                        {agent.EgressMoney},
		"invoice_send_edi":                    {agent.EgressExternalRecipient},
		"agent_raise_exception":               {agent.EgressInternal},
		"agent_create_wait":                   {agent.EgressInternal},
		"agent_cancel_wait":                   {agent.EgressInternal},
		"report_create_dashboard":             {agent.EgressInternal},
		"report_add_dashboard_tile":           {agent.EgressInternal},
		"report_create_table_change_alert":    {agent.EgressInternal},
		"report_update":                       {agent.EgressInternal},
		"report_fork":                         {agent.EgressInternal},
		"dispatch_assign_move":                {agent.EgressInternal},
		"dispatch_record_stop_actual":         {agent.EgressInternal},
		"dispatch_record_check_call":          {agent.EgressInternal},
		"dispatch_share_tracking_link":        {agent.EgressExternalRecipient},
		"dispatch_revoke_tracking_link":       {agent.EgressExternalRecipient},
		"shipment_attach_document":            {agent.EgressInternal},
		"shipment_place_hold":                 {agent.EgressInternal},
		"shipment_release_hold":               {agent.EgressInternal},
		"shipment_share_tracking_link": {
			agent.EgressCustomerVisible,
			agent.EgressExternalRecipient,
		},
		"shipment_revoke_tracking_link": {agent.EgressCustomerVisible},
		"shipment_create":               {agent.EgressInternal},
		"location_create":               {agent.EgressInternal},
		"detention_escalate":            {agent.EgressInternal},
		"worker_place_dispatch_hold":    {agent.EgressInternal},
		"equipment_set_tractor_status":  {agent.EgressInternal},
		"equipment_set_trailer_status":  {agent.EgressInternal},
		"worker_approve_pto":            {agent.EgressInternal},
		"carrier_acknowledge_intelligence_event": {
			agent.EgressInternal,
		},
		"carrier_resolve_intelligence_event":        {agent.EgressInternal},
		"insight_dismiss":                           {agent.EgressInternal},
		"agent_create_memory":                       {agent.EgressPersonal, agent.EgressInternal},
		"agent_guide_profile":                       {agent.EgressPersonal, agent.EgressInternal},
		"agent_retire_memory":                       {agent.EgressInternal},
		"exception_evaluate_service_failures":       {agent.EgressInternal},
		"receivable_resolve_bank_receipt_work_item": {agent.EgressInternal},
		"inbox_link_inbound_message":                {agent.EgressInternal},
		"inbox_mark_inbound_message":                {agent.EgressInternal},
		"inbox_assign_inbound_message":              {agent.EgressInternal},
		"inbox_reopen_inbound_message":              {agent.EgressInternal},
		"inbox_snooze_inbound_message":              {agent.EgressInternal},
		"inbox_wake_inbound_message":                {agent.EgressInternal},
		"inbox_await_reply":                         {agent.EgressInternal},
		"inbox_unlink_inbound_message":              {agent.EgressInternal},
		"inbox_assign_queue":                        {agent.EgressInternal},
		"inbox_block_sender":                        {agent.EgressInternal},
		"inbox_unblock_sender":                      {agent.EgressInternal},
		"inbox_draft_reply":                         {agent.EgressInternal},
		"inbox_discard_reply_draft":                 {agent.EgressInternal},
		"inbox_draft_load":                          {agent.EgressInternal},
		"inbox_create_load":                         {agent.EgressInternal},
		"inbox_discard_load":                        {agent.EgressInternal},
		"inbox_file_attachment":                     {agent.EgressInternal},
		"inbox_unfile_attachment":                   {agent.EgressInternal},
		"inbox_record_discrepancy":                  {agent.EgressInternal},
		"detention_resolve_dispute":                 {agent.EgressMoney},
		"inbox_resolve_conflict":                    {agent.EgressInternal},
		"inbox_reopen_conflict":                     {agent.EgressInternal},
		"accounting_check_connection":               {agent.EgressInternal},
		"accounting_set_mapping":                    {agent.EgressInternal},
		"accounting_clear_mapping":                  {agent.EgressInternal},
		"accounting_create_reference_record": {
			agent.EgressInternal,
		},
		"accounting_refresh_reference_data": {agent.EgressInternal},
		"accounting_retry_sync_record":      {agent.EgressInternal},
		"accounting_skip_sync_record":       {agent.EgressInternal},
		"accounting_redate_sync_record":     {agent.EgressInternal},
		"accounting_ignore_inbound_change":  {agent.EgressInternal},
		"accounting_dismiss_drift_finding":  {agent.EgressInternal},
		"accounting_check_drift":            {agent.EgressInternal},
		"accounting_pause_sync":             {agent.EgressInternal},
		"accounting_resume_sync":            {agent.EgressInternal},
		"accounting_request_backfill":       {agent.EgressInternal},
		"shipment_add_comment": {
			agent.EgressInternal,
			agent.EgressCustomerVisible,
			agent.EgressDriverVisible,
		},
		"worker_send_message":                       {agent.EgressDriverVisible},
		"worker_request_credential_renewal":         {agent.EgressDriverVisible},
		"worker_reject_pto":                         {agent.EgressDriverVisible},
		"worker_cancel_pto":                         {agent.EgressDriverVisible},
		"customer_send_email":                       {agent.EgressExternalRecipient},
		"detention_send_notice":                     {agent.EgressExternalRecipient},
		"inbox_send_reply":                          {agent.EgressExternalRecipient},
		"shipment_request_documents":                {agent.EgressExternalRecipient},
		"dispatch_create_tender_from_routing_guide": {agent.EgressExternalRecipient},
		"dispatch_create_tender":                    {agent.EgressExternalRecipient},
		"report_create_schedule":                    {agent.EgressExternalRecipient},
		"shipment_cancel":                           {agent.EgressExternalRecipient},
		"shipment_update":                           {agent.EgressExternalRecipient},
		"exception_resolve_service_failure":         {agent.EgressExternalRecipient},
		"receivable_post_customer_payment":          {agent.EgressMoney},
		"accounting_apply_inbound_change":           {agent.EgressMoney},
		"accounting_resolve_drift_finding":          {agent.EgressMoney},
		"receivable_match_bank_receipt":             {agent.EgressMoney},
		"detention_waive":                           {agent.EgressMoney},
		"detention_approve":                         {agent.EgressMoney},
		"billing_correct_charge_codes":              {agent.EgressMoney},
		"dispatch_unassign_move":                    {agent.EgressInternal},
		"dispatch_set_move_status":                  {agent.EgressInternal},
		"rate_generate_confirmation":                {agent.EgressInternal},
		"dispatch_cancel_tender":                    {agent.EgressExternalRecipient},
		"rate_send_confirmation":                    {agent.EgressExternalRecipient},
		"dispatch_record_tender_response": {
			agent.EgressExternalRecipient,
			agent.EgressMoney,
		},
		"dispatch_create_carrier_assignment": {agent.EgressMoney},
		"dispatch_cancel_carrier_assignment": {agent.EgressMoney},
		"rate_void_confirmation":             {agent.EgressMoney},
		"rate_record_confirmation":           {agent.EgressMoney},
		"edi_accept_tender":                  {agent.EgressExternalRecipient},
		"edi_decline_tender":                 {agent.EgressExternalRecipient},
		"edi_cancel_tender":                  {agent.EgressExternalRecipient},
		"edi_expire_tender":                  {agent.EgressExternalRecipient},
		"edi_review_tender_change":           {agent.EgressExternalRecipient},
		"edi_review_transfer_change":         {agent.EgressExternalRecipient},
		"edi_retry_message":                  {agent.EgressExternalRecipient},
		"edi_replay_message":                 {agent.EgressExternalRecipient},
		"edi_reprocess_inbound_file":         {agent.EgressExternalRecipient},
		"edi_send_tender":                    {agent.EgressExternalRecipient},
		"edi_send_status_update":             {agent.EgressExternalRecipient},
		"edi_apply_connection_draft":         {agent.EgressInternal},
		"settlement_submit_driver":           {agent.EgressInternal},
		"settlement_submit_carrier":          {agent.EgressInternal},
		"settlement_reject_driver":           {agent.EgressInternal},
		"settlement_reject_carrier":          {agent.EgressInternal},
		"settlement_recalculate_driver":      {agent.EgressInternal},
		"settlement_recalculate_carrier":     {agent.EgressInternal},
		"settlement_release_pay_event":       {agent.EgressInternal},
		"settlement_attach_pay_events":       {agent.EgressInternal},
		"settlement_detach_pay_event":        {agent.EgressInternal},
		"settlement_generate_driver":         {agent.EgressInternal},
		"settlement_generate_driver_batch":   {agent.EgressInternal},
		"settlement_generate_carrier_batch":  {agent.EgressInternal},
		"carrier_create_invoice_match":       {agent.EgressInternal},
		"carrier_reject_invoice_match":       {agent.EgressInternal},
		"edi_link_carrier_invoice": {
			agent.EgressInternal,
		},
		"settlement_hold_pay_event": {agent.EgressDriverVisible},
		"settlement_approve_driver": {
			agent.EgressDriverVisible,
			agent.EgressMoney,
		},
		"settlement_post_driver": {
			agent.EgressDriverVisible,
			agent.EgressMoney,
		},
		"settlement_record_driver_payment": {
			agent.EgressDriverVisible,
			agent.EgressMoney,
		},
		"settlement_pay_now": {
			agent.EgressDriverVisible,
			agent.EgressMoney,
		},
		"settlement_approve_carrier":           {agent.EgressMoney},
		"settlement_post_carrier":              {agent.EgressMoney},
		"settlement_record_carrier_payment":    {agent.EgressMoney},
		"settlement_void_driver":               {agent.EgressMoney},
		"settlement_void_carrier":              {agent.EgressMoney},
		"settlement_add_driver_adjustment":     {agent.EgressMoney},
		"settlement_add_carrier_adjustment":    {agent.EgressMoney},
		"settlement_remove_driver_adjustment":  {agent.EgressMoney},
		"settlement_remove_carrier_adjustment": {agent.EgressMoney},
		"carrier_accept_invoice_match":         {agent.EgressMoney},
		"carrier_accept_invoice_match_with_variance": {
			agent.EgressMoney,
		},
		"settlement_issue_advance":              {agent.EgressMoney},
		"settlement_forgive_advance":            {agent.EgressMoney},
		"settlement_open_escrow_account":        {agent.EgressMoney},
		"settlement_update_escrow_account":      {agent.EgressMoney},
		"settlement_adjust_escrow_account":      {agent.EgressMoney},
		"settlement_close_escrow_account":       {agent.EgressMoney},
		"settlement_assign_pay_profile":         {agent.EgressMoney},
		"settlement_end_pay_assignment":         {agent.EgressMoney},
		"settlement_create_recurring_deduction": {agent.EgressMoney},
		"settlement_update_recurring_deduction": {agent.EgressMoney},
		"settlement_create_recurring_earning":   {agent.EgressMoney},
		"settlement_update_recurring_earning":   {agent.EgressMoney},
		"shipment_uncancel":                     {agent.EgressInternal},
		"shipment_transfer_ownership":           {agent.EgressInternal},
		"shipment_rerate":                       {agent.EgressMoney},
		"shipment_recalculate_distance":         {agent.EgressInternal},
		"shipment_duplicate":                    {agent.EgressInternal},
		"shipment_update_hold": {
			agent.EgressInternal,
			agent.EgressCustomerVisible,
		},
		"dispatch_split_move":    {agent.EgressInternal},
		"shipment_pin_comment":   {agent.EgressInternal},
		"shipment_unpin_comment": {agent.EgressInternal},
		"shipment_resolve_comment": {
			agent.EgressInternal,
		},
		"shipment_update_comment": {
			agent.EgressInternal,
			agent.EgressCustomerVisible,
			agent.EgressDriverVisible,
		},
		"shipment_delete_comment":          {agent.EgressInternal},
		"exception_update_service_failure": {agent.EgressInternal},
		"exception_review_service_failure": {agent.EgressExternalRecipient},
		"exception_void_service_failure":   {agent.EgressExternalRecipient},
		"detention_open_dispute":           {agent.EgressMoney},
		"order_create":                     {agent.EgressInternal},
		"order_update":                     {agent.EgressInternal, agent.EgressMoney},
		"order_attach_shipments":           {agent.EgressInternal},
		"order_detach_shipment":            {agent.EgressInternal},
		"order_close":                      {agent.EgressInternal},
		"order_cancel":                     {agent.EgressExternalRecipient},
		"order_add_charge":                 {agent.EgressMoney},
		"order_update_charge":              {agent.EgressMoney},
		"order_set_charge_allocations": {
			agent.EgressMoney,
		},
		"order_remove_charge":       {agent.EgressMoney},
		"shipment_create_recurring": {agent.EgressInternal},
		"shipment_update_recurring": {agent.EgressInternal},
		"shipment_set_recurring_status": {
			agent.EgressInternal,
		},
		"shipment_generate_recurring": {agent.EgressInternal},
		"worker_assign_shift":         {agent.EgressInternal},
		"worker_end_shift_assignment": {agent.EgressInternal},
		"worker_set_availability_preference": {
			agent.EgressInternal,
		},
		"worker_request_shift_swap":          {agent.EgressInternal},
		"worker_approve_shift_swap":          {agent.EgressDriverVisible},
		"worker_reject_shift_swap":           {agent.EgressDriverVisible},
		"worker_withdraw_shift_swap":         {agent.EgressDriverVisible},
		"carrier_vet":                        {agent.EgressMoney},
		"customer_vet_broker":                {agent.EgressMoney},
		"carrier_set_monitoring":             {agent.EgressMoney},
		"carrier_record_intelligence_review": {agent.EgressInternal},
		"carrier_apply_intelligence_suggestions": {
			agent.EgressInternal,
		},
		"carrier_import":              {agent.EgressInternal},
		"carrier_verify_equipment":    {agent.EgressInternal},
		"billing_reassign_charge":     {agent.EgressMoney},
		"billing_manage_transfer_run": {agent.EgressInternal},

		"ledger_create_manual_journal":             {agent.EgressInternal},
		"ledger_update_manual_journal":             {agent.EgressInternal},
		"ledger_submit_manual_journal":             {agent.EgressInternal},
		"ledger_cancel_manual_journal":             {agent.EgressInternal},
		"ledger_post_manual_journal":               {agent.EgressMoney},
		"ledger_create_journal_reversal":           {agent.EgressMoney},
		"ledger_cancel_journal_reversal":           {agent.EgressInternal},
		"ledger_post_journal_reversal":             {agent.EgressMoney},
		"ledger_close_fiscal_period":               {agent.EgressMoney},
		"ledger_lock_fiscal_period":                {agent.EgressMoney},
		"ledger_unlock_fiscal_period":              {agent.EgressMoney},
		"ledger_reopen_fiscal_period":              {agent.EgressMoney},
		"ledger_open_fiscal_period":                {agent.EgressMoney},
		"accounting_confirm_mapping":               {agent.EgressInternal},
		"accounting_reject_mapping":                {agent.EgressInternal},
		"accounting_release_sync_record":           {agent.EgressMoney},
		"accounting_update_backfill":               {agent.EgressInternal},
		"receivable_triage_bank_receipt_work_item": {agent.EgressInternal},

		"fuel_record_purchase":                          {agent.EgressInternal},
		"fuel_correct_purchase":                         {agent.EgressInternal},
		"fuel_delete_purchase":                          {agent.EgressInternal},
		"fuel_assign_card":                              {agent.EgressInternal},
		"fuel_commit_purchase_import":                   {agent.EgressInternal},
		"fuel_resolve_purchase_import_rows":             {agent.EgressInternal},
		"fuel_discard_purchase_import":                  {agent.EgressInternal},
		"fuel_generate_ifta_return":                     {agent.EgressInternal},
		"fuel_recalculate_ifta_return":                  {agent.EgressInternal},
		"fuel_amend_ifta_return":                        {agent.EgressInternal},
		"fuel_delete_ifta_return":                       {agent.EgressInternal},
		"fuel_record_ifta_mileage_entry":                {agent.EgressInternal},
		"fuel_correct_ifta_mileage_entry":               {agent.EgressInternal},
		"fuel_delete_ifta_mileage_entry":                {agent.EgressInternal},
		"fuel_recalculate_jurisdiction_miles":           {agent.EgressInternal},
		"fuel_backfill_jurisdiction_miles":              {agent.EgressInternal},
		"rate_create_agreement":                         {agent.EgressInternal},
		"rate_update_agreement_draft":                   {agent.EgressInternal},
		"rate_duplicate_agreement":                      {agent.EgressInternal},
		"rate_submit_agreement":                         {agent.EgressInternal},
		"rate_reject_agreement":                         {agent.EgressInternal},
		"rate_discard_import":                           {agent.EgressInternal},
		"rate_run_simulation":                           {agent.EgressInternal},
		"report_cancel_run":                             {agent.EgressInternal},
		"report_delete":                                 {agent.EgressInternal},
		"report_reset_fork":                             {agent.EgressInternal},
		"report_delete_dashboard":                       {agent.EgressInternal},
		"report_delete_schedule":                        {agent.EgressInternal},
		"rate_record_fuel_index_price":                  {agent.EgressMoney},
		"rate_correct_fuel_index_price":                 {agent.EgressMoney},
		"rate_approve_agreement":                        {agent.EgressMoney},
		"rate_suspend_agreement":                        {agent.EgressMoney},
		"rate_resume_agreement":                         {agent.EgressMoney},
		"rate_archive_agreement":                        {agent.EgressMoney},
		"rate_amend_agreement_rules":                    {agent.EgressMoney},
		"rate_apply_increase":                           {agent.EgressMoney},
		"rate_commit_import":                            {agent.EgressMoney},
		"report_update_schedule":                        {agent.EgressExternalRecipient},
		"worker_open_safety_event":                      {agent.EgressInternal},
		"worker_update_safety_event":                    {agent.EgressInternal},
		"worker_set_safety_event_status":                {agent.EgressInternal},
		"worker_delete_safety_event":                    {agent.EgressInternal},
		"worker_record_recognition":                     {agent.EgressDriverVisible},
		"worker_delete_recognition":                     {agent.EgressInternal},
		"worker_record_safety_violation":                {agent.EgressInternal},
		"worker_update_safety_violation":                {agent.EgressInternal},
		"worker_delete_safety_violation":                {agent.EgressInternal},
		"worker_schedule_dot_test":                      {agent.EgressInternal},
		"worker_cancel_dot_test":                        {agent.EgressInternal},
		"worker_run_dot_random_draw":                    {agent.EgressInternal},
		"worker_finalize_dot_random_draw":               {agent.EgressInternal},
		"worker_cancel_dot_random_draw":                 {agent.EgressInternal},
		"worker_update_dot_random_selection":            {agent.EgressInternal},
		"worker_open_leave_case":                        {agent.EgressInternal},
		"worker_update_leave_case":                      {agent.EgressInternal},
		"worker_close_leave_case":                       {agent.EgressInternal},
		"worker_request_leave_certification":            {agent.EgressInternal},
		"worker_record_leave_day":                       {agent.EgressInternal},
		"worker_update_leave_day":                       {agent.EgressInternal},
		"worker_delete_leave_day":                       {agent.EgressInternal},
		"worker_assign_training":                        {agent.EgressDriverVisible},
		"worker_assign_required_training":               {agent.EgressDriverVisible},
		"worker_record_training_completion":             {agent.EgressInternal},
		"worker_attach_training_document":               {agent.EgressInternal},
		"worker_close_training":                         {agent.EgressInternal},
		"worker_start_checklist":                        {agent.EgressInternal},
		"worker_update_checklist_item":                  {agent.EgressInternal},
		"worker_cancel_checklist":                       {agent.EgressInternal},
		"worker_start_performance_review":               {agent.EgressInternal},
		"worker_update_performance_review":              {agent.EgressInternal},
		"worker_delete_performance_review":              {agent.EgressInternal},
		"worker_record_employment_verification":         {agent.EgressInternal},
		"worker_update_employment_verification":         {agent.EgressInternal},
		"worker_record_employment_verification_request": {agent.EgressInternal},
		"worker_delete_employment_verification":         {agent.EgressInternal},
		"worker_record_credential":                      {agent.EgressInternal},
		"worker_update_credential":                      {agent.EgressInternal},
		"worker_attach_credential_document":             {agent.EgressInternal},
		"worker_archive_credential":                     {agent.EgressInternal},
		"worker_record_injury":                          {agent.EgressInternal},
		"worker_update_injury":                          {agent.EgressInternal},
		"worker_delete_injury":                          {agent.EgressInternal},
		"worker_request_pto":                            {agent.EgressDriverVisible},
		"worker_update_pto":                             {agent.EgressDriverVisible},
		"worker_adjust_pto_balance":                     {agent.EgressMoney},
		"shipment_record_permit":                        {agent.EgressInternal},
		"shipment_update_permit":                        {agent.EgressInternal},
		"settlement_start_dispute_review":               {agent.EgressDriverVisible},
		"settlement_resolve_dispute":                    {agent.EgressDriverVisible, agent.EgressMoney},
		"settlement_review_driver_expense":              {agent.EgressDriverVisible, agent.EgressMoney},
		"settlement_generate_payroll_export":            {agent.EgressMoney},
		"settlement_void_payroll_export":                {agent.EgressMoney},

		"carrier_create":                           {agent.EgressInternal, agent.EgressMoney},
		"carrier_update":                           {agent.EgressInternal, agent.EgressMoney},
		"carrier_create_capacity_posting":          {agent.EgressInternal},
		"carrier_update_capacity_posting":          {agent.EgressInternal},
		"carrier_delete_capacity_posting":          {agent.EgressInternal},
		"carrier_set_status":                       {agent.EgressInternal},
		"customer_create":                          {agent.EgressInternal, agent.EgressExternalRecipient},
		"customer_update":                          {agent.EgressInternal, agent.EgressExternalRecipient},
		"customer_set_status":                      {agent.EgressInternal},
		"masterdata_create_commodity":              {agent.EgressInternal},
		"masterdata_update_commodity":              {agent.EgressInternal},
		"masterdata_set_commodity_status":          {agent.EgressInternal},
		"masterdata_create_hazardous_material":     {agent.EgressInternal},
		"masterdata_update_hazardous_material":     {agent.EgressInternal},
		"masterdata_set_hazardous_material_status": {agent.EgressInternal},
		"location_update":                          {agent.EgressInternal},
		"location_set_status":                      {agent.EgressInternal},
		"equipment_create_tractor":                 {agent.EgressInternal},
		"equipment_update_tractor":                 {agent.EgressInternal},
		"equipment_set_tractor_location":           {agent.EgressInternal},
		"equipment_create_trailer":                 {agent.EgressInternal},
		"equipment_update_trailer":                 {agent.EgressInternal},
		"equipment_set_trailer_location":           {agent.EgressMoney},
		"document_delete":                          {agent.EgressInternal},
		"document_restore_version":                 {agent.EgressInternal},
		"insight_restore":                          {agent.EgressInternal},
		"report_update_table_change_alert":         {agent.EgressInternal},
		"report_set_table_change_alert_status":     {agent.EgressInternal},
		"report_delete_table_change_alert":         {agent.EgressInternal},
		"exception_dismiss_watchtower_item":        {agent.EgressInternal},
		"exception_assign":                         {agent.EgressInternal},
		"exception_snooze":                         {agent.EgressInternal},
		"exception_wake":                           {agent.EgressInternal},
		"receivable_record_collection_activity":    {agent.EgressInternal},
		"receivable_assign_collector":              {agent.EgressInternal},
		"receivable_schedule_follow_up":            {agent.EgressInternal},
		"receivable_send_payment_reminder":         {agent.EgressExternalRecipient},
		"receivable_send_statement":                {agent.EgressExternalRecipient},
		"exception_set_due":                        {agent.EgressInternal},
		"exception_resolve":                        {agent.EgressInternal},
		"exception_reopen":                         {agent.EgressInternal},
		"exception_add_comment":                    {agent.EgressInternal},
		"inbox_file_capture_item":                  {agent.EgressInternal},
		"inbox_discard_capture_item":               {agent.EgressInternal},
		"inbox_discard_capture_batch":              {agent.EgressInternal},
		"shipment_reschedule_stop":                 {agent.EgressExternalRecipient},
		"shipment_add_charge":                      {agent.EgressMoney},
		"factoring_create_noa":                     {agent.EgressInternal},
		"factoring_send_noa":                       {agent.EgressExternalRecipient},
		"factoring_create_submission":              {agent.EgressExternalRecipient},
		"factoring_retry_submission":               {agent.EgressExternalRecipient},
		"factoring_cancel_submission":              {agent.EgressInternal},
		"factoring_record_noa_acknowledgement":     {agent.EgressInternal},
		"factoring_release_noa":                    {agent.EgressInternal},
		"factoring_set_override":                   {agent.EgressInternal},
		"factoring_request_document":               {agent.EgressExternalRecipient},
		"factoring_record_funding":                 {agent.EgressMoney},
		"factoring_release_reserve":                {agent.EgressMoney},
		"factoring_record_chargeback":              {agent.EgressMoney},
		"factoring_reverse_chargeback":             {agent.EgressMoney},
		"factoring_resolve_remittance_line":        {agent.EgressInternal},
		"factoring_rematch_remittance":             {agent.EgressInternal},
		"factoring_apply_remittance":               {agent.EgressMoney},
		"factoring_discard_remittance":             {agent.EgressInternal},
		"web_use_portal":                           {agent.EgressExternalRecipient},
		"web_watch":                                {agent.EgressInternal},
		"web_stop_watch":                           {agent.EgressInternal},
	}

	tools := buildRegistered(t)
	require.Len(t, cases, len(tools.Actions), "every action tool is classified here")
	for name, want := range cases {
		assert.Equal(t, want, registeredPolicy(t, name).Egress, name)
	}
	for _, name := range []string{"inbox_link_inbound_message", "inbox_mark_inbound_message",
		"inbox_send_reply", "billing_start_review", "inbox_unlink_inbound_message",
		"inbox_draft_reply", "inbox_draft_load", "inbox_discard_load",
		"inbox_file_attachment", "inbox_record_discrepancy", "inbox_reopen_conflict"} {
		assert.NotNil(t, registeredPolicy(t, name).Condition, name)
	}
}

func TestRuntimeToolsStayInside(t *testing.T) {
	t.Parallel()

	effects := map[string]agent.EgressClass{}
	for _, policy := range agentruntime.RuntimePolicies() {
		effects[policy.Name] = policy.Egress[0]
	}

	assert.Equal(t, map[string]agent.EgressClass{
		"find_tools":        agent.EgressNone,
		"ask_user":          agent.EgressNone,
		"publish_artifact":  agent.EgressPersonal,
		"delegate_task":     agent.EgressNone,
		"request_decision":  agent.EgressPersonal,
		"withdraw_proposal": agent.EgressNone,
		"name_plan":         agent.EgressNone,
	}, effects)
}

/*
Every tool a Desk case step names is a registered tool. A name that drifts
from a renamed tool would make every agent look unable to take the step, and
the Desk would hand off or send the person to a page for work their agent
can do.
*/
func TestEveryToolACaseStepNamesIsRegistered(t *testing.T) {
	t.Parallel()

	tools := buildRegistered(t)
	names := make(map[string]struct{}, len(tools.Actions)+len(tools.Queries))
	for _, tool := range tools.Actions {
		names[tool.Name()] = struct{}{}
	}
	for _, tool := range tools.Queries {
		names[tool.Name()] = struct{}{}
	}

	for _, tool := range deskcase.AllStepTools() {
		_, ok := names[tool]
		assert.Truef(t, ok, "case steps name %q, which is not a registered tool", tool)
	}
}

// A script's broker runs only reads: a write, a self-scoped tool, an extension
// tool and the script tool itself are never callable from a script, whatever
// their policy otherwise says.
func TestScriptsCallOnlyReads(t *testing.T) {
	t.Parallel()

	tools := buildRegistered(t)
	runners := 0
	for _, tool := range tools.Queries {
		policy := tool.Policy()
		if policy.Script == agentports.ScriptRunner {
			runners++
			assert.False(t, policy.CallableFromScript(), "%s never runs inside a script", tool.Name())
		}
		if !policy.CallableFromScript() {
			continue
		}
		assert.Equal(t, agent.ToolEffectLookup, policy.EffectiveEffect(), tool.Name())
		assert.NotEqual(t, agent.ToolScopeSelf, policy.Scope, tool.Name())
	}
	assert.Equal(t, 1, runners, "agent_run_script is the one tool that runs scripts")
	for _, tool := range tools.Actions {
		policy := tool.Policy()
		assert.False(t, policy.CallableFromScript(), "%s is a write", tool.Name())
	}
}

// A script prepares only writes a person can approve record by record: each
// names the record it changes and previews it, changes the tenant's records
// rather than the agent's own work, never escalates on its own and holds no
// taint of its own. The tools that are the agent's own work stay out.
func TestScriptsPrepareOnlyWritesThatNameAndPreviewTheirRecord(t *testing.T) {
	t.Parallel()

	tools := buildRegistered(t)
	preparable := 0
	for _, tool := range tools.Actions {
		if !agentports.PreparableFromScript(tool) {
			continue
		}
		preparable++
		policy := tool.Policy()
		assert.Equal(t, agent.ToolScopeTenant, policy.Scope, tool.Name())
		assert.False(t, policy.Escalates, tool.Name())
		assert.Nil(t, policy.TaintHold, tool.Name())
		assert.NotEqual(t, agent.ToolEffectLookup, policy.EffectiveEffect(), tool.Name())
		assert.False(t, strings.HasPrefix(tool.Name(), "agent_"), tool.Name())
		_, previews := tool.(agentports.ToolPreviewer)
		_, targeted := tool.(agentports.TargetedTool)
		assert.True(t, previews && targeted, "%s names and previews its record", tool.Name())
	}
	assert.Positive(t, preparable, "scripts have writes they may prepare")

	for _, name := range []string{
		"agent_create_memory", "agent_create_wait", "agent_raise_exception",
	} {
		for _, tool := range tools.Actions {
			if tool.Name() == name {
				assert.False(t, agentports.PreparableFromScript(tool), "%s is the agent's own work", name)
			}
		}
	}
}
