package agenttoolpolicy_test

import (
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
		"get_inbound_message":          agent.ExternalReadAlways,
		"list_inbound_messages":        agent.ExternalReadAlways,
		"search_inbound_messages":      agent.ExternalReadAlways,
		"search_documents":             agent.ExternalReadAlways,
		"get_document_summary":         agent.ExternalReadAlways,
		"intake_get_draft":             agent.ExternalReadAlways,
		"get_bank_receipt":             agent.ExternalReadAlways,
		"list_bank_receipt_exceptions": agent.ExternalReadAlways,
		"list_weather_alerts":          agent.ExternalReadAlways,
		"recall_memory":                agent.ExternalReadMarked,
		"get_agent_run":                agent.ExternalReadMarked,
		"shipment_get":                 agent.ExternalReadMarked,
		"list_watchtower_items":        agent.ExternalReadMarked,
		"list_agent_runs":              agent.ExternalReadMarked,
		"get_daily_briefing":           agent.ExternalReadNever,
		"get_customer":                 agent.ExternalReadNever,
		"list_edi_inbound_files":       agent.ExternalReadAlways,
		"get_edi_inbound_file":         agent.ExternalReadAlways,
		"get_edi_message":              agent.ExternalReadAlways,
		"list_edi_transfers":           agent.ExternalReadAlways,
		"get_edi_partner":              agent.ExternalReadNever,
		"settlement_get_dispute":       agent.ExternalReadMarked,
		"list_carrier_invoice_matches": agent.ExternalReadMarked,
		"invoice_get":                  agent.ExternalReadNever,
		"get_ar_aging":                 agent.ExternalReadNever,
		"settlement_get_driver":        agent.ExternalReadNever,
		"get_rate_agreement":           agent.ExternalReadNever,
		"order_get":                    agent.ExternalReadNever,
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
		"list_edi_inbound_files",
		"get_edi_inbound_file",
		"get_edi_message",
		"list_edi_transfers",
		"list_carrier_invoice_matches",
	} {
		assert.Equal(t, agent.TaintSourceEDI, registeredPolicy(t, name).Source, name)
	}
	assert.Equal(t, agent.TaintSourceRunRecord, registeredPolicy(t, "list_agent_runs").Source)
	watchtower := registeredPolicy(t, "list_watchtower_items")
	assert.Equal(t, agent.TaintSourceInboundMessage, watchtower.Source)
	assert.ElementsMatch(t, []agent.TaintSource{
		agent.TaintSourceEDI,
		agent.TaintSourceWeather,
		agent.TaintSourceRunRecord,
	}, watchtower.Sources)
	assert.True(t, registeredPolicy(t, "remember").CarriesTaint)
	assert.NotNil(t, registeredPolicy(t, "remember").TaintHold)
	assert.True(t, registeredPolicy(t, "guide_profile").CarriesTaint)
	assert.NotNil(t, registeredPolicy(t, "guide_profile").TaintHold)
	assert.True(t, registeredPolicy(t, "shipment_add_comment").CarriesTaint)
}

func TestEveryRegisteredToolIsInTheClassItWasGiven(t *testing.T) {
	t.Parallel()

	cases := map[string][]agent.EgressClass{
		"add_home_widget":               {agent.EgressPersonal},
		"remove_home_widget":            {agent.EgressPersonal},
		"arrange_home_layout":           {agent.EgressPersonal},
		"create_report":                 {agent.EgressPersonal, agent.EgressInternal},
		"save_table_view":               {agent.EgressPersonal, agent.EgressInternal},
		"billing_start_review":          {agent.EgressInternal},
		"billing_transfer":              {agent.EgressInternal},
		"billing_return_to_operations":  {agent.EgressInternal},
		"billing_raise_exception":       {agent.EgressInternal},
		"billing_hold_queue_item":       {agent.EgressInternal},
		"billing_release_queue_item":    {agent.EgressInternal},
		"billing_resolve_issue":         {agent.EgressMoney, agent.EgressDriverVisible},
		"billing_undo_issue":            {agent.EgressMoney},
		"billing_assign_biller":         {agent.EgressInternal},
		"billing_approve_queue_item":    {agent.EgressMoney},
		"billing_cancel_queue_item":     {agent.EgressMoney},
		"invoice_post":                  {agent.EgressMoney},
		"invoice_send":                  {agent.EgressExternalRecipient},
		"invoice_open_dispute":          {agent.EgressInternal},
		"invoice_resolve_dispute":       {agent.EgressInternal},
		"invoice_withdraw_dispute":      {agent.EgressInternal},
		"apply_customer_payment":        {agent.EgressMoney},
		"reverse_customer_payment":      {agent.EgressMoney},
		"apply_credit_memo":             {agent.EgressMoney},
		"unapply_credit_memo":           {agent.EgressMoney},
		"invoice_save_adjustment_draft": {agent.EgressInternal},
		"invoice_submit_adjustment":     {agent.EgressMoney},
		"invoice_approve_adjustment":    {agent.EgressMoney},
		"invoice_reject_adjustment":     {agent.EgressInternal},
		"invoice_build_run":             {agent.EgressInternal},
		"invoice_adjust_run":            {agent.EgressInternal},
		"invoice_commit_run":            {agent.EgressMoney},
		"invoice_cancel_run":            {agent.EgressInternal},
		"invoice_bill_statement":        {agent.EgressMoney},
		"assess_late_charges":           {agent.EgressMoney},
		"invoice_share":                 {agent.EgressInternal},
		"invoice_update_draft":          {agent.EgressInternal},
		"invoice_generate_pdf":          {agent.EgressInternal},
		"invoice_create":                {agent.EgressMoney},
		"invoice_create_memo":           {agent.EgressMoney},
		"invoice_void":                  {agent.EgressMoney},
		"invoice_send_edi":              {agent.EgressExternalRecipient},
		"flag_for_manual_review":        {agent.EgressInternal},
		"raise_exception":               {agent.EgressInternal},
		"wait_until":                    {agent.EgressInternal},
		"cancel_wait":                   {agent.EgressInternal},
		"create_dashboard":              {agent.EgressInternal},
		"add_dashboard_tile":            {agent.EgressInternal},
		"create_table_change_alert":     {agent.EgressInternal},
		"update_report":                 {agent.EgressInternal},
		"fork_report":                   {agent.EgressInternal},
		"assign_move":                   {agent.EgressInternal},
		"record_stop_actual":            {agent.EgressInternal},
		"log_check_call":                {agent.EgressInternal},
		"issue_carrier_tracking_link":   {agent.EgressExternalRecipient},
		"revoke_carrier_tracking_link":  {agent.EgressExternalRecipient},
		"shipment_attach_document":      {agent.EgressInternal},
		"shipment_place_hold":           {agent.EgressInternal},
		"shipment_release_hold":         {agent.EgressInternal},
		"shipment_share_tracking_link": {
			agent.EgressCustomerVisible,
			agent.EgressExternalRecipient,
		},
		"shipment_revoke_tracking_link": {agent.EgressCustomerVisible},
		"shipment_create":               {agent.EgressInternal},
		"create_location":               {agent.EgressInternal},
		"escalate_detention":            {agent.EgressInternal},
		"place_worker_dispatch_hold":    {agent.EgressInternal},
		"update_tractor_status":         {agent.EgressInternal},
		"update_trailer_status":         {agent.EgressInternal},
		"approve_worker_pto":            {agent.EgressInternal},
		"acknowledge_carrier_intel_event": {
			agent.EgressInternal,
		},
		"resolve_carrier_intel_event":    {agent.EgressInternal},
		"dismiss_insight":                {agent.EgressInternal},
		"remember":                       {agent.EgressPersonal, agent.EgressInternal},
		"guide_profile":                  {agent.EgressPersonal, agent.EgressInternal},
		"forget_memory":                  {agent.EgressInternal},
		"evaluate_service_failures":      {agent.EgressInternal},
		"resolve_bank_receipt_work_item": {agent.EgressInternal},
		"link_inbound_message":           {agent.EgressInternal},
		"mark_inbound_message":           {agent.EgressInternal},
		"assign_inbound_messages":        {agent.EgressInternal},
		"reopen_inbound_messages":        {agent.EgressInternal},
		"snooze_inbound_messages":        {agent.EgressInternal},
		"wake_inbound_message":           {agent.EgressInternal},
		"wait_for_inbound_reply":         {agent.EgressInternal},
		"unlink_inbound_message":         {agent.EgressInternal},
		"hand_off_triage":                {agent.EgressInternal},
		"block_inbound_sender":           {agent.EgressInternal},
		"unblock_inbound_sender":         {agent.EgressInternal},
		"draft_inbound_reply":            {agent.EgressInternal},
		"discard_inbound_reply_draft":    {agent.EgressInternal},
		"draft_inbound_load":             {agent.EgressInternal},
		"create_inbound_load":            {agent.EgressInternal},
		"discard_inbound_load":           {agent.EgressInternal},
		"file_inbound_attachment":        {agent.EgressInternal},
		"unfile_inbound_attachment":      {agent.EgressInternal},
		"log_inbound_exception":          {agent.EgressInternal},
		"settle_detention_dispute":       {agent.EgressMoney},
		"resolve_inbound_conflict":       {agent.EgressInternal},
		"reopen_inbound_conflict":        {agent.EgressInternal},
		"check_accounting_connection":    {agent.EgressInternal},
		"set_accounting_mapping":         {agent.EgressInternal},
		"clear_accounting_mapping":       {agent.EgressInternal},
		"create_accounting_reference_record": {
			agent.EgressInternal,
		},
		"refresh_accounting_reference_data": {agent.EgressInternal},
		"retry_accounting_sync":             {agent.EgressInternal},
		"skip_accounting_sync":              {agent.EgressInternal},
		"redate_accounting_sync":            {agent.EgressInternal},
		"ignore_accounting_inbound_change":  {agent.EgressInternal},
		"dismiss_accounting_drift":          {agent.EgressInternal},
		"check_accounting_drift":            {agent.EgressInternal},
		"pause_accounting_sync":             {agent.EgressInternal},
		"resume_accounting_sync":            {agent.EgressInternal},
		"request_accounting_backfill":       {agent.EgressInternal},
		"shipment_add_comment": {
			agent.EgressInternal,
			agent.EgressCustomerVisible,
			agent.EgressDriverVisible,
		},
		"notify_driver":                   {agent.EgressDriverVisible},
		"request_credential_renewal":      {agent.EgressDriverVisible},
		"reject_worker_pto":               {agent.EgressDriverVisible},
		"cancel_worker_pto":               {agent.EgressDriverVisible},
		"email_customer":                  {agent.EgressExternalRecipient},
		"send_detention_notice":           {agent.EgressExternalRecipient},
		"reply_to_inbound_message":        {agent.EgressExternalRecipient},
		"shipment_request_documents":      {agent.EgressExternalRecipient},
		"tender_move_to_routing_guide":    {agent.EgressExternalRecipient},
		"tender_move_to_carriers":         {agent.EgressExternalRecipient},
		"schedule_report":                 {agent.EgressExternalRecipient},
		"shipment_cancel":                 {agent.EgressExternalRecipient},
		"shipment_update":                 {agent.EgressExternalRecipient},
		"resolve_service_failure":         {agent.EgressExternalRecipient},
		"post_customer_payment":           {agent.EgressMoney},
		"apply_accounting_inbound_change": {agent.EgressMoney},
		"resolve_accounting_drift":        {agent.EgressMoney},
		"match_bank_receipt":              {agent.EgressMoney},
		"waive_detention":                 {agent.EgressMoney},
		"approve_detention":               {agent.EgressMoney},
		"billing_correct_charge_codes":    {agent.EgressMoney},
		"unassign_moves":                  {agent.EgressInternal},
		"update_move_status":              {agent.EgressInternal},
		"generate_rate_confirmation":      {agent.EgressInternal},
		"cancel_tender":                   {agent.EgressExternalRecipient},
		"send_rate_confirmation":          {agent.EgressExternalRecipient},
		"record_tender_response": {
			agent.EgressExternalRecipient,
			agent.EgressMoney,
		},
		"assign_move_to_carrier":             {agent.EgressMoney},
		"cancel_carrier_assignment":          {agent.EgressMoney},
		"void_rate_confirmation":             {agent.EgressMoney},
		"record_rate_confirmation_confirmed": {agent.EgressMoney},
		"accept_edi_tender":                  {agent.EgressExternalRecipient},
		"decline_edi_tender":                 {agent.EgressExternalRecipient},
		"cancel_edi_tender":                  {agent.EgressExternalRecipient},
		"expire_edi_tender":                  {agent.EgressExternalRecipient},
		"review_edi_tender_change":           {agent.EgressExternalRecipient},
		"review_edi_transfer_change":         {agent.EgressExternalRecipient},
		"retry_edi_message_delivery":         {agent.EgressExternalRecipient},
		"replay_edi_message":                 {agent.EgressExternalRecipient},
		"reprocess_edi_inbound_files":        {agent.EgressExternalRecipient},
		"send_edi_tender":                    {agent.EgressExternalRecipient},
		"send_edi_status_update":             {agent.EgressExternalRecipient},
		"apply_edi_connection_draft":         {agent.EgressInternal},
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
		"create_carrier_invoice_match":       {agent.EgressInternal},
		"reject_carrier_invoice_match":       {agent.EgressInternal},
		"link_edi_carrier_invoice_to_carrier": {
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
		"accept_carrier_invoice_match":         {agent.EgressMoney},
		"accept_carrier_invoice_match_with_variance": {
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
		"split_move_at_relay":    {agent.EgressInternal},
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
		"shipment_delete_comment": {agent.EgressInternal},
		"update_service_failure":  {agent.EgressInternal},
		"review_service_failure":  {agent.EgressExternalRecipient},
		"void_service_failure":    {agent.EgressExternalRecipient},
		"dispute_detention":       {agent.EgressMoney},
		"order_create":            {agent.EgressInternal},
		"order_update":            {agent.EgressInternal, agent.EgressMoney},
		"order_attach_shipments":  {agent.EgressInternal},
		"order_detach_shipment":   {agent.EgressInternal},
		"order_close":             {agent.EgressInternal},
		"order_cancel":            {agent.EgressExternalRecipient},
		"order_add_charge":        {agent.EgressMoney},
		"order_update_charge":     {agent.EgressMoney},
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
		"assign_worker_shift":         {agent.EgressInternal},
		"end_worker_shift_assignment": {agent.EgressInternal},
		"set_worker_availability_preference": {
			agent.EgressInternal,
		},
		"propose_shift_swap":          {agent.EgressInternal},
		"approve_shift_swap":          {agent.EgressDriverVisible},
		"reject_shift_swap":           {agent.EgressDriverVisible},
		"withdraw_shift_swap":         {agent.EgressDriverVisible},
		"vet_carrier":                 {agent.EgressMoney},
		"vet_customer_broker":         {agent.EgressMoney},
		"set_carrier_monitoring":      {agent.EgressMoney},
		"mark_carrier_intel_reviewed": {agent.EgressInternal},
		"apply_carrier_intel_suggestions": {
			agent.EgressInternal,
		},
		"import_sourced_carrier":      {agent.EgressInternal},
		"verify_carrier_equipment":    {agent.EgressInternal},
		"billing_reassign_charge":     {agent.EgressMoney},
		"billing_manage_transfer_run": {agent.EgressInternal},

		"draft_manual_journal":                 {agent.EgressInternal},
		"revise_manual_journal_draft":          {agent.EgressInternal},
		"submit_manual_journal":                {agent.EgressInternal},
		"cancel_manual_journal":                {agent.EgressInternal},
		"post_manual_journal":                  {agent.EgressMoney},
		"request_journal_reversal":             {agent.EgressMoney},
		"cancel_journal_reversal":              {agent.EgressInternal},
		"post_journal_reversal":                {agent.EgressMoney},
		"close_fiscal_period":                  {agent.EgressMoney},
		"lock_fiscal_period":                   {agent.EgressMoney},
		"unlock_fiscal_period":                 {agent.EgressMoney},
		"reopen_fiscal_period":                 {agent.EgressMoney},
		"open_fiscal_period":                   {agent.EgressMoney},
		"confirm_accounting_mapping_proposals": {agent.EgressInternal},
		"reject_accounting_mapping_proposal":   {agent.EgressInternal},
		"release_accounting_sync":              {agent.EgressMoney},
		"change_accounting_backfill":           {agent.EgressInternal},
		"triage_bank_receipt_work_item":        {agent.EgressInternal},

		"record_fuel_purchase":                {agent.EgressInternal},
		"correct_fuel_purchase":               {agent.EgressInternal},
		"delete_fuel_purchase":                {agent.EgressInternal},
		"assign_fuel_card":                    {agent.EgressInternal},
		"commit_fuel_purchase_import":         {agent.EgressInternal},
		"resolve_fuel_purchase_import_rows":   {agent.EgressInternal},
		"discard_fuel_purchase_import":        {agent.EgressInternal},
		"generate_ifta_return":                {agent.EgressInternal},
		"recompute_ifta_return":               {agent.EgressInternal},
		"amend_ifta_return":                   {agent.EgressInternal},
		"delete_ifta_return":                  {agent.EgressInternal},
		"record_ifta_mileage_entry":           {agent.EgressInternal},
		"correct_ifta_mileage_entry":          {agent.EgressInternal},
		"delete_ifta_mileage_entry":           {agent.EgressInternal},
		"recalculate_move_jurisdiction_miles": {agent.EgressInternal},
		"backfill_jurisdiction_miles":         {agent.EgressInternal},
		"draft_rate_agreement":                {agent.EgressInternal},
		"revise_rate_agreement_draft":         {agent.EgressInternal},
		"duplicate_rate_agreement":            {agent.EgressInternal},
		"submit_rate_agreement":               {agent.EgressInternal},
		"reject_rate_agreement":               {agent.EgressInternal},
		"discard_rate_import":                 {agent.EgressInternal},
		"run_rate_simulation":                 {agent.EgressInternal},
		"cancel_report_run":                   {agent.EgressInternal},
		"delete_report":                       {agent.EgressInternal},
		"reset_report_fork":                   {agent.EgressInternal},
		"delete_dashboard":                    {agent.EgressInternal},
		"delete_report_schedule":              {agent.EgressInternal},
		"record_fuel_index_price":             {agent.EgressMoney},
		"correct_fuel_index_price":            {agent.EgressMoney},
		"approve_rate_agreement":              {agent.EgressMoney},
		"suspend_rate_agreement":              {agent.EgressMoney},
		"resume_rate_agreement":               {agent.EgressMoney},
		"archive_rate_agreement":              {agent.EgressMoney},
		"amend_rate_agreement_rules":          {agent.EgressMoney},
		"apply_rate_increase":                 {agent.EgressMoney},
		"commit_rate_import":                  {agent.EgressMoney},
		"update_report_schedule":              {agent.EgressExternalRecipient},
		"open_worker_safety_event":            {agent.EgressInternal},
		"update_worker_safety_event":          {agent.EgressInternal},
		"change_worker_safety_event_status":   {agent.EgressInternal},
		"delete_worker_safety_event":          {agent.EgressInternal},
		"give_worker_recognition":             {agent.EgressDriverVisible},
		"delete_worker_recognition":           {agent.EgressInternal},
		"record_safety_violation":             {agent.EgressInternal},
		"update_safety_violation":             {agent.EgressInternal},
		"delete_safety_violation":             {agent.EgressInternal},
		"schedule_dot_test":                   {agent.EgressInternal},
		"cancel_dot_test":                     {agent.EgressInternal},
		"run_dot_random_draw":                 {agent.EgressInternal},
		"finalize_dot_random_draw":            {agent.EgressInternal},
		"cancel_dot_random_draw":              {agent.EgressInternal},
		"update_dot_random_selection":         {agent.EgressInternal},
		"open_leave_case":                     {agent.EgressInternal},
		"update_leave_case":                   {agent.EgressInternal},
		"close_leave_case":                    {agent.EgressInternal},
		"request_leave_certification":         {agent.EgressInternal},
		"record_leave_day":                    {agent.EgressInternal},
		"update_leave_day":                    {agent.EgressInternal},
		"delete_leave_day":                    {agent.EgressInternal},
		"assign_worker_training":              {agent.EgressDriverVisible},
		"assign_required_worker_training":     {agent.EgressDriverVisible},
		"record_training_completion":          {agent.EgressInternal},
		"attach_worker_training_document":     {agent.EgressInternal},
		"close_worker_training":               {agent.EgressInternal},
		"start_worker_checklist":              {agent.EgressInternal},
		"update_worker_checklist_item":        {agent.EgressInternal},
		"cancel_worker_checklist":             {agent.EgressInternal},
		"start_performance_review":            {agent.EgressInternal},
		"draft_performance_review":            {agent.EgressInternal},
		"delete_performance_review":           {agent.EgressInternal},
		"record_employment_verification":      {agent.EgressInternal},
		"update_employment_verification":      {agent.EgressInternal},
		"log_employment_verification_request": {agent.EgressInternal},
		"delete_employment_verification":      {agent.EgressInternal},
		"record_worker_credential":            {agent.EgressInternal},
		"update_worker_credential":            {agent.EgressInternal},
		"attach_worker_credential_document":   {agent.EgressInternal},
		"archive_worker_credential":           {agent.EgressInternal},
		"record_worker_injury":                {agent.EgressInternal},
		"update_worker_injury":                {agent.EgressInternal},
		"delete_worker_injury":                {agent.EgressInternal},
		"request_worker_pto":                  {agent.EgressDriverVisible},
		"update_worker_pto":                   {agent.EgressDriverVisible},
		"adjust_worker_pto_balance":           {agent.EgressMoney},
		"shipment_record_permit":              {agent.EgressInternal},
		"shipment_update_permit":              {agent.EgressInternal},
		"settlement_start_dispute_review":     {agent.EgressDriverVisible},
		"settlement_resolve_dispute":          {agent.EgressDriverVisible, agent.EgressMoney},
		"review_driver_expense":               {agent.EgressDriverVisible, agent.EgressMoney},
		"settlement_generate_payroll_export":  {agent.EgressMoney},
		"settlement_void_payroll_export":      {agent.EgressMoney},

		"create_carrier":                       {agent.EgressInternal, agent.EgressMoney},
		"update_carrier":                       {agent.EgressInternal, agent.EgressMoney},
		"create_carrier_capacity_posting":      {agent.EgressInternal},
		"update_carrier_capacity_posting":      {agent.EgressInternal},
		"delete_carrier_capacity_posting":      {agent.EgressInternal},
		"update_carrier_status":                {agent.EgressInternal},
		"create_customer":                      {agent.EgressInternal, agent.EgressExternalRecipient},
		"update_customer":                      {agent.EgressInternal, agent.EgressExternalRecipient},
		"update_customer_status":               {agent.EgressInternal},
		"create_commodity":                     {agent.EgressInternal},
		"update_commodity":                     {agent.EgressInternal},
		"update_commodity_status":              {agent.EgressInternal},
		"create_hazardous_material":            {agent.EgressInternal},
		"update_hazardous_material":            {agent.EgressInternal},
		"update_hazardous_material_status":     {agent.EgressInternal},
		"update_location":                      {agent.EgressInternal},
		"update_location_status":               {agent.EgressInternal},
		"create_tractor":                       {agent.EgressInternal},
		"update_tractor":                       {agent.EgressInternal},
		"locate_tractor":                       {agent.EgressInternal},
		"create_trailer":                       {agent.EgressInternal},
		"update_trailer":                       {agent.EgressInternal},
		"locate_trailer":                       {agent.EgressMoney},
		"delete_documents":                     {agent.EgressInternal},
		"restore_document_version":             {agent.EgressInternal},
		"restore_insight":                      {agent.EgressInternal},
		"update_table_change_alert":            {agent.EgressInternal},
		"set_table_change_alert_status":        {agent.EgressInternal},
		"delete_table_change_alert":            {agent.EgressInternal},
		"dismiss_watchtower_item":              {agent.EgressInternal},
		"assign_exception":                     {agent.EgressInternal},
		"snooze_exception":                     {agent.EgressInternal},
		"wake_exception":                       {agent.EgressInternal},
		"log_collection_activity":              {agent.EgressInternal},
		"assign_collector":                     {agent.EgressInternal},
		"schedule_collection_follow_up":        {agent.EgressInternal},
		"send_payment_reminder":                {agent.EgressExternalRecipient},
		"send_customer_statement":              {agent.EgressExternalRecipient},
		"set_exception_due":                    {agent.EgressInternal},
		"resolve_exception":                    {agent.EgressInternal},
		"reopen_exception":                     {agent.EgressInternal},
		"comment_on_exception":                 {agent.EgressInternal},
		"file_capture_items":                   {agent.EgressInternal},
		"discard_capture_item":                 {agent.EgressInternal},
		"discard_capture_batch":                {agent.EgressInternal},
		"shipment_reschedule_stop":             {agent.EgressExternalRecipient},
		"shipment_add_charge":                  {agent.EgressMoney},
		"draft_factoring_noa":                  {agent.EgressInternal},
		"send_factoring_noa":                   {agent.EgressExternalRecipient},
		"submit_invoices_to_factor":            {agent.EgressExternalRecipient},
		"retry_factoring_submission":           {agent.EgressExternalRecipient},
		"cancel_factoring_submission":          {agent.EgressInternal},
		"record_factoring_noa_acknowledgement": {agent.EgressInternal},
		"release_factoring_noa":                {agent.EgressInternal},
		"set_invoice_factoring_override":       {agent.EgressInternal},
		"request_missing_factoring_document":   {agent.EgressExternalRecipient},
		"record_factoring_funding":             {agent.EgressMoney},
		"release_factoring_reserve":            {agent.EgressMoney},
		"record_factoring_chargeback":          {agent.EgressMoney},
		"reverse_factoring_chargeback":         {agent.EgressMoney},
		"resolve_factoring_remittance_line":    {agent.EgressInternal},
		"rematch_factoring_remittance":         {agent.EgressInternal},
		"apply_factoring_remittance":           {agent.EgressMoney},
		"discard_factoring_remittance":         {agent.EgressInternal},
		"act_on_portal":                        {agent.EgressExternalRecipient},
		"watch_web":                            {agent.EgressInternal},
		"stop_web_watch":                       {agent.EgressInternal},
	}

	tools := buildRegistered(t)
	require.Len(t, cases, len(tools.Actions), "every action tool is classified here")
	for name, want := range cases {
		assert.Equal(t, want, registeredPolicy(t, name).Egress, name)
	}
	for _, name := range []string{"link_inbound_message", "mark_inbound_message",
		"reply_to_inbound_message", "billing_start_review", "unlink_inbound_message",
		"draft_inbound_reply", "draft_inbound_load", "discard_inbound_load",
		"file_inbound_attachment", "log_inbound_exception", "reopen_inbound_conflict"} {
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
