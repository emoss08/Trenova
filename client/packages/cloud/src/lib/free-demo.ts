import {
  PLAN_CAPABILITY_LABELS,
  planCapabilityLabel,
  planDisplayName as hostPlanDisplayName,
} from "@/lib/plan-meters";

export const PLAN_USAGE_PATH = "/admin/plan-usage";

export const FREE_DEMO_PLAN_KEY = "free_demo";

export type PlanCapabilityDefinition = {
  label: string;
  explanation: string;
};

const DEMO_EXPLANATIONS: Readonly<Record<string, string>> = {
  "email.outbound":
    "Invoices, rate confirmations, tenders, detention notices and driver invites are not emailed from a demo organization. Sign-in and password email still arrive.",
  integrations:
    "Connections to Samsara, QuickBooks, Xero, EDI partners, Google Maps and other third-party services are not available on the demo.",
  api_keys: "Programmatic access with API keys is not available on the demo.",
  "agent.automation":
    "Scheduled and event-triggered agents, the morning briefing and background agent runs are not available on the demo. You can still talk to the assistant within the AI limits.",
  "agent.web_search": "Agents cannot search the web from a demo organization.",
  "carrier_intelligence.paid": "Paid carrier intelligence lookups are not available on the demo.",
  document_intelligence:
    "Automatic document extraction and classification are not available on the demo. Documents can still be uploaded and filed by hand.",
  sms: "Text messages are not sent from a demo organization.",
  sso: "Organization SSO and SCIM provisioning are not available on the demo.",
};

const UNKNOWN_EXPLANATION = "This part of Trenova is not included in your current plan.";

/** Every capability the free demo withholds, in the order the plan page lists them. */
export const PLAN_CAPABILITIES: Readonly<Record<string, PlanCapabilityDefinition>> =
  Object.fromEntries(
    Object.keys(PLAN_CAPABILITY_LABELS).map((key) => [
      key,
      {
        label: planCapabilityLabel(key),
        explanation: DEMO_EXPLANATIONS[key] ?? UNKNOWN_EXPLANATION,
      },
    ]),
  );

export function planCapabilityDefinition(capability: string): PlanCapabilityDefinition {
  return (
    PLAN_CAPABILITIES[capability] ?? {
      label: planCapabilityLabel(capability),
      explanation: UNKNOWN_EXPLANATION,
    }
  );
}

export function planDisplayName(planKey: string, planName?: string): string {
  if ((!planName || planName.trim() === "") && planKey === FREE_DEMO_PLAN_KEY) {
    return "Free demo";
  }
  return hostPlanDisplayName(planKey, planName);
}
