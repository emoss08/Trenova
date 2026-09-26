package agenttoolpolicy_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
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
		"add_home_widget":              {agent.EgressPersonal},
		"remove_home_widget":           {agent.EgressPersonal},
		"arrange_home_layout":          {agent.EgressPersonal},
		"create_report":                {agent.EgressPersonal, agent.EgressInternal},
		"save_table_view":              {agent.EgressPersonal, agent.EgressInternal},
		"transition_item_to_in_review": {agent.EgressInternal},
		"flag_for_manual_review":       {agent.EgressInternal},
		"raise_exception":              {agent.EgressInternal},
		"create_dashboard":             {agent.EgressInternal},
		"add_dashboard_tile":           {agent.EgressInternal},
		"create_table_change_alert":    {agent.EgressInternal},
		"update_report":                {agent.EgressInternal},
		"fork_report":                  {agent.EgressInternal},
		"assign_move":                  {agent.EgressInternal},
		"record_stop_actual":           {agent.EgressInternal},
		"attach_document_to_shipment":  {agent.EgressInternal},
		"place_shipment_hold":          {agent.EgressInternal},
		"release_shipment_hold":        {agent.EgressInternal},
		"create_shipment":              {agent.EgressInternal},
		"create_location":              {agent.EgressInternal},
		"escalate_detention":           {agent.EgressInternal},
		"place_worker_dispatch_hold":   {agent.EgressInternal},
		"update_tractor_status":        {agent.EgressInternal},
		"update_trailer_status":        {agent.EgressInternal},
		"approve_worker_pto":           {agent.EgressInternal},
		"acknowledge_carrier_intel_event": {
			agent.EgressInternal,
		},
		"resolve_carrier_intel_event":    {agent.EgressInternal},
		"dismiss_insight":                {agent.EgressInternal},
		"remember":                       {agent.EgressInternal},
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
	}, effects)
}
