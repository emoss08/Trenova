package agenttoolpolicy_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/deskcase"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy/registered"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildRegistered(t *testing.T) registered.Tools {
	t.Helper()

	tools, err := registered.Build()
	require.NoError(t, err)

	return tools
}

func registeredPolicy(t *testing.T, name string) serviceports.ToolPolicy {
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

	return serviceports.ToolPolicy{}
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
		"get_shipment_draft":           agent.ExternalReadAlways,
		"get_bank_receipt":             agent.ExternalReadAlways,
		"list_bank_receipt_exceptions": agent.ExternalReadAlways,
		"list_weather_alerts":          agent.ExternalReadAlways,
		"recall_memory":                agent.ExternalReadMarked,
		"get_agent_run":                agent.ExternalReadMarked,
		"get_shipment":                 agent.ExternalReadMarked,
		"list_watchtower_items":        agent.ExternalReadMarked,
		"list_agent_runs":              agent.ExternalReadMarked,
		"get_daily_briefing":           agent.ExternalReadNever,
		"get_customer":                 agent.ExternalReadNever,
		"list_edi_inbound_files":       agent.ExternalReadAlways,
		"get_edi_inbound_file":         agent.ExternalReadAlways,
		"list_edi_transfers":           agent.ExternalReadAlways,
		"get_edi_partner":              agent.ExternalReadNever,
		"get_settlement_dispute":       agent.ExternalReadMarked,
		"list_carrier_invoice_matches": agent.ExternalReadMarked,
		"get_invoice":                  agent.ExternalReadNever,
		"get_ar_aging":                 agent.ExternalReadNever,
		"get_driver_settlement":        agent.ExternalReadNever,
		"get_rate_agreement":           agent.ExternalReadNever,
		"get_order":                    agent.ExternalReadNever,
	}
	for name, want := range cases {
		policy := registeredPolicy(t, name)
		assert.Equal(t, want, policy.ReadsExternal, name)
		if want != agent.ExternalReadNever {
			assert.True(t, policy.Source.IsValid(), name)
		}
	}
	assert.Equal(t, agent.TaintSourceRecordNote, registeredPolicy(t, "get_shipment").Source)
	assert.Equal(t, agent.TaintSourceRecordNote,
		registeredPolicy(t, "get_settlement_dispute").Source)
	for _, name := range []string{
		"list_edi_inbound_files",
		"get_edi_inbound_file",
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
	assert.True(t, registeredPolicy(t, "add_shipment_comment").CarriesTaint)
}

func TestEveryRegisteredToolIsInTheClassItWasGiven(t *testing.T) {
	t.Parallel()

	cases := map[string][]agent.EgressClass{
		"add_home_widget":                {agent.EgressPersonal},
		"remove_home_widget":             {agent.EgressPersonal},
		"arrange_home_layout":            {agent.EgressPersonal},
		"create_report":                  {agent.EgressPersonal, agent.EgressInternal},
		"save_table_view":                {agent.EgressPersonal, agent.EgressInternal},
		"transition_item_to_in_review":   {agent.EgressInternal},
		"transfer_to_billing":            {agent.EgressInternal},
		"send_billing_item_back_to_ops":  {agent.EgressInternal},
		"move_billing_item_to_exception": {agent.EgressInternal},
		"hold_billing_queue_item":        {agent.EgressInternal},
		"assign_billing_queue_biller":    {agent.EgressInternal},
		"assign_billing_queue_billers":   {agent.EgressInternal},
		"transition_items_to_in_review":  {agent.EgressInternal},
		"approve_billing_queue_item":     {agent.EgressMoney},
		"cancel_billing_queue_item":      {agent.EgressMoney},
		"post_invoice":                   {agent.EgressMoney},
		"send_invoice":                   {agent.EgressExternalRecipient},
		"post_invoices":                  {agent.EgressMoney},
		"send_invoices":                  {agent.EgressExternalRecipient},
		"approve_billing_queue_items":    {agent.EgressMoney},
		"open_invoice_dispute":           {agent.EgressInternal},
		"resolve_invoice_dispute":        {agent.EgressInternal},
		"withdraw_invoice_dispute":       {agent.EgressInternal},
		"apply_customer_payment":         {agent.EgressMoney},
		"reverse_customer_payment":       {agent.EgressMoney},
		"apply_credit_memo":              {agent.EgressMoney},
		"unapply_credit_memo":            {agent.EgressMoney},
		"save_invoice_adjustment_draft":  {agent.EgressInternal},
		"submit_invoice_adjustment":      {agent.EgressMoney},
		"approve_invoice_adjustment":     {agent.EgressMoney},
		"reject_invoice_adjustment":      {agent.EgressInternal},
		"build_invoice_run":              {agent.EgressInternal},
		"adjust_invoice_run_membership":  {agent.EgressInternal},
		"commit_invoice_run":             {agent.EgressMoney},
		"cancel_invoice_run":             {agent.EgressInternal},
		"bill_statement_now":             {agent.EgressMoney},
		"assess_late_charges":            {agent.EgressMoney},
		"share_invoice":                  {agent.EgressInternal},
		"update_invoice_draft":           {agent.EgressInternal},
		"generate_invoice_pdf":           {agent.EgressInternal},
		"create_invoice":                 {agent.EgressMoney},
		"create_invoice_memo":            {agent.EgressMoney},
		"void_invoice":                   {agent.EgressMoney},
		"send_invoice_edi":               {agent.EgressExternalRecipient},
		"flag_for_manual_review":         {agent.EgressInternal},
		"raise_exception":                {agent.EgressInternal},
		"wait_until":                     {agent.EgressInternal},
		"cancel_wait":                    {agent.EgressInternal},
		"create_dashboard":               {agent.EgressInternal},
		"add_dashboard_tile":             {agent.EgressInternal},
		"create_table_change_alert":      {agent.EgressInternal},
		"update_report":                  {agent.EgressInternal},
		"fork_report":                    {agent.EgressInternal},
		"assign_move":                    {agent.EgressInternal},
		"record_stop_actual":             {agent.EgressInternal},
		"attach_document_to_shipment":    {agent.EgressInternal},
		"place_shipment_hold":            {agent.EgressInternal},
		"release_shipment_hold":          {agent.EgressInternal},
		"create_shipment":                {agent.EgressInternal},
		"create_location":                {agent.EgressInternal},
		"escalate_detention":             {agent.EgressInternal},
		"place_worker_dispatch_hold":     {agent.EgressInternal},
		"update_tractor_status":          {agent.EgressInternal},
		"update_trailer_status":          {agent.EgressInternal},
		"approve_worker_pto":             {agent.EgressInternal},
		"acknowledge_carrier_intel_event": {
			agent.EgressInternal,
		},
		"resolve_carrier_intel_event":    {agent.EgressInternal},
		"dismiss_insight":                {agent.EgressInternal},
		"remember":                       {agent.EgressPersonal, agent.EgressInternal},
		"forget_memory":                  {agent.EgressInternal},
		"evaluate_service_failures":      {agent.EgressInternal},
		"resolve_bank_receipt_work_item": {agent.EgressInternal},
		"link_inbound_message":           {agent.EgressInternal},
		"mark_inbound_message":           {agent.EgressInternal},
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
		"add_shipment_comment": {
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
		"request_missing_docs":            {agent.EgressExternalRecipient},
		"tender_move_to_routing_guide":    {agent.EgressExternalRecipient},
		"tender_move_to_carriers":         {agent.EgressExternalRecipient},
		"schedule_report":                 {agent.EgressExternalRecipient},
		"cancel_shipment":                 {agent.EgressExternalRecipient},
		"update_shipment":                 {agent.EgressExternalRecipient},
		"resolve_service_failure":         {agent.EgressExternalRecipient},
		"post_customer_payment":           {agent.EgressMoney},
		"apply_accounting_inbound_change": {agent.EgressMoney},
		"resolve_accounting_drift":        {agent.EgressMoney},
		"match_bank_receipt":              {agent.EgressMoney},
		"waive_detention":                 {agent.EgressMoney},
		"approve_detention":               {agent.EgressMoney},
		"correct_charge_code":             {agent.EgressMoney},
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
		"submit_driver_settlement":           {agent.EgressInternal},
		"submit_carrier_settlement":          {agent.EgressInternal},
		"reject_driver_settlement":           {agent.EgressInternal},
		"reject_carrier_settlement":          {agent.EgressInternal},
		"recalculate_driver_settlement":      {agent.EgressInternal},
		"recalculate_carrier_settlement":     {agent.EgressInternal},
		"release_driver_pay_event":           {agent.EgressInternal},
		"attach_pay_events_to_settlement":    {agent.EgressInternal},
		"detach_pay_event_from_settlement":   {agent.EgressInternal},
		"generate_driver_settlement":         {agent.EgressInternal},
		"generate_driver_settlement_batch":   {agent.EgressInternal},
		"generate_carrier_settlement_batch":  {agent.EgressInternal},
		"create_carrier_invoice_match":       {agent.EgressInternal},
		"reject_carrier_invoice_match":       {agent.EgressInternal},
		"link_edi_carrier_invoice_to_carrier": {
			agent.EgressInternal,
		},
		"hold_driver_pay_event": {agent.EgressDriverVisible},
		"approve_driver_settlement": {
			agent.EgressDriverVisible,
			agent.EgressMoney,
		},
		"post_driver_settlement": {
			agent.EgressDriverVisible,
			agent.EgressMoney,
		},
		"approve_driver_settlements": {
			agent.EgressDriverVisible,
			agent.EgressMoney,
		},
		"post_driver_settlements": {
			agent.EgressDriverVisible,
			agent.EgressMoney,
		},
		"record_driver_settlement_payment": {
			agent.EgressDriverVisible,
			agent.EgressMoney,
		},
		"pay_driver_now": {
			agent.EgressDriverVisible,
			agent.EgressMoney,
		},
		"approve_carrier_settlement":           {agent.EgressMoney},
		"post_carrier_settlement":              {agent.EgressMoney},
		"approve_carrier_settlements":          {agent.EgressMoney},
		"post_carrier_settlements":             {agent.EgressMoney},
		"record_carrier_settlement_payment":    {agent.EgressMoney},
		"void_driver_settlement":               {agent.EgressMoney},
		"void_carrier_settlement":              {agent.EgressMoney},
		"add_driver_settlement_adjustment":     {agent.EgressMoney},
		"add_carrier_settlement_adjustment":    {agent.EgressMoney},
		"remove_driver_settlement_adjustment":  {agent.EgressMoney},
		"remove_carrier_settlement_adjustment": {agent.EgressMoney},
		"accept_carrier_invoice_match":         {agent.EgressMoney},
		"accept_carrier_invoice_match_with_variance": {
			agent.EgressMoney,
		},
		"issue_pay_advance":             {agent.EgressMoney},
		"write_off_pay_advance":         {agent.EgressMoney},
		"open_escrow_account":           {agent.EgressMoney},
		"update_escrow_account":         {agent.EgressMoney},
		"adjust_escrow_account":         {agent.EgressMoney},
		"close_escrow_account":          {agent.EgressMoney},
		"assign_pay_profile":            {agent.EgressMoney},
		"end_pay_assignment":            {agent.EgressMoney},
		"create_recurring_deduction":    {agent.EgressMoney},
		"update_recurring_deduction":    {agent.EgressMoney},
		"create_recurring_earning":      {agent.EgressMoney},
		"update_recurring_earning":      {agent.EgressMoney},
		"uncancel_shipment":             {agent.EgressInternal},
		"transfer_shipment_ownership":   {agent.EgressInternal},
		"rerate_shipment":               {agent.EgressMoney},
		"recalculate_shipment_distance": {agent.EgressInternal},
		"duplicate_shipment":            {agent.EgressInternal},
		"update_shipment_hold": {
			agent.EgressInternal,
			agent.EgressCustomerVisible,
		},
		"split_move_at_relay":    {agent.EgressInternal},
		"pin_shipment_comment":   {agent.EgressInternal},
		"unpin_shipment_comment": {agent.EgressInternal},
		"resolve_shipment_comment": {
			agent.EgressInternal,
		},
		"edit_shipment_comment": {
			agent.EgressInternal,
			agent.EgressCustomerVisible,
			agent.EgressDriverVisible,
		},
		"delete_shipment_comment": {agent.EgressInternal},
		"update_service_failure":  {agent.EgressInternal},
		"review_service_failure":  {agent.EgressExternalRecipient},
		"void_service_failure":    {agent.EgressExternalRecipient},
		"dispute_detention":       {agent.EgressMoney},
		"create_order":            {agent.EgressInternal},
		"update_order":            {agent.EgressInternal, agent.EgressMoney},
		"attach_order_shipments":  {agent.EgressInternal},
		"detach_order_shipment":   {agent.EgressInternal},
		"close_order":             {agent.EgressInternal},
		"cancel_order":            {agent.EgressExternalRecipient},
		"add_order_charge":        {agent.EgressMoney},
		"update_order_charge":     {agent.EgressMoney},
		"set_order_charge_allocations": {
			agent.EgressMoney,
		},
		"remove_order_charge":       {agent.EgressMoney},
		"create_recurring_shipment": {agent.EgressInternal},
		"update_recurring_shipment": {agent.EgressInternal},
		"set_recurring_shipment_status": {
			agent.EgressInternal,
		},
		"generate_recurring_shipment": {agent.EgressInternal},
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
		"reassign_billing_charge":     {agent.EgressMoney},
		"manage_billing_transfer_run": {agent.EgressInternal},

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
		"record_shipment_permit":              {agent.EgressInternal},
		"update_shipment_permit":              {agent.EgressInternal},
		"start_settlement_dispute_review":     {agent.EgressDriverVisible},
		"resolve_settlement_dispute":          {agent.EgressDriverVisible, agent.EgressMoney},
		"review_driver_expense":               {agent.EgressDriverVisible, agent.EgressMoney},
		"generate_payroll_export":             {agent.EgressMoney},
		"void_payroll_export":                 {agent.EgressMoney},

		"create_carrier":                   {agent.EgressInternal, agent.EgressMoney},
		"update_carrier":                   {agent.EgressInternal, agent.EgressMoney},
		"create_carrier_capacity_posting":  {agent.EgressInternal},
		"update_carrier_capacity_posting":  {agent.EgressInternal},
		"delete_carrier_capacity_posting":  {agent.EgressInternal},
		"update_carrier_status":            {agent.EgressInternal},
		"create_customer":                  {agent.EgressInternal, agent.EgressExternalRecipient},
		"update_customer":                  {agent.EgressInternal, agent.EgressExternalRecipient},
		"update_customer_status":           {agent.EgressInternal},
		"create_commodity":                 {agent.EgressInternal},
		"update_commodity":                 {agent.EgressInternal},
		"update_commodity_status":          {agent.EgressInternal},
		"create_hazardous_material":        {agent.EgressInternal},
		"update_hazardous_material":        {agent.EgressInternal},
		"update_hazardous_material_status": {agent.EgressInternal},
		"update_location":                  {agent.EgressInternal},
		"update_location_status":           {agent.EgressInternal},
		"create_tractor":                   {agent.EgressInternal},
		"update_tractor":                   {agent.EgressInternal},
		"locate_tractor":                   {agent.EgressInternal},
		"create_trailer":                   {agent.EgressInternal},
		"update_trailer":                   {agent.EgressInternal},
		"locate_trailer":                   {agent.EgressMoney},
		"delete_documents":                 {agent.EgressInternal},
		"restore_document_version":         {agent.EgressInternal},
		"restore_insight":                  {agent.EgressInternal},
		"update_table_change_alert":        {agent.EgressInternal},
		"set_table_change_alert_status":    {agent.EgressInternal},
		"delete_table_change_alert":        {agent.EgressInternal},
		"dismiss_watchtower_item":          {agent.EgressInternal},
		"file_capture_items":               {agent.EgressInternal},
		"discard_capture_item":             {agent.EgressInternal},
		"discard_capture_batch":            {agent.EgressInternal},
	}

	tools := buildRegistered(t)
	require.Len(t, cases, len(tools.Actions), "every action tool is classified here")
	for name, want := range cases {
		assert.Equal(t, want, registeredPolicy(t, name).Egress, name)
	}
	for _, name := range []string{"link_inbound_message", "mark_inbound_message",
		"reply_to_inbound_message", "transition_item_to_in_review"} {
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
		"find_tools":       agent.EgressNone,
		"ask_user":         agent.EgressNone,
		"publish_artifact": agent.EgressPersonal,
		"delegate_task":    agent.EgressNone,
		"request_decision": agent.EgressPersonal,
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
