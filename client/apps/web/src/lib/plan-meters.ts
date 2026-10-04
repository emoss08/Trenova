import { formatCurrency, formatFileSize } from "@trenova/shared/lib/utils";

export const PLAN_USAGE_PATH = "/admin/plan-usage";

export type PlanMeterUnit = "count" | "bytes" | "cents";
export type PlanMeterWindow = "lifetime" | "month" | "item";

export type PlanMeterDefinition = {
  label: string;
  description: string;
  unit: PlanMeterUnit;
  window: PlanMeterWindow;
};

/**
 * The free demo plan's meters, keyed as the server reports them (platformcatalog
 * MeterKey). The order here is the order the plan page lists them in: the records a
 * demo holds first, then storage, then AI.
 */
export const PLAN_METERS: Readonly<Record<string, PlanMeterDefinition>> = {
  "shipments.total": {
    label: "Shipments",
    description: "Shipments in the organization, however they were created.",
    unit: "count",
    window: "lifetime",
  },
  "recurring_shipments.series": {
    label: "Recurring shipment series",
    description: "Recurring shipment schedules.",
    unit: "count",
    window: "lifetime",
  },
  "customers.total": {
    label: "Customers",
    description: "Customer records.",
    unit: "count",
    window: "lifetime",
  },
  "locations.total": {
    label: "Locations",
    description: "Pickup, delivery and other location records.",
    unit: "count",
    window: "lifetime",
  },
  "workers.total": {
    label: "Workers",
    description: "Drivers and other worker records.",
    unit: "count",
    window: "lifetime",
  },
  "tractors.total": {
    label: "Tractors",
    description: "Tractor records.",
    unit: "count",
    window: "lifetime",
  },
  "trailers.total": {
    label: "Trailers",
    description: "Trailer records.",
    unit: "count",
    window: "lifetime",
  },
  "users.seats": {
    label: "User seats",
    description: "People with access to the organization.",
    unit: "count",
    window: "lifetime",
  },
  "documents.uploads": {
    label: "Document uploads",
    description: "Documents stored in the organization.",
    unit: "count",
    window: "lifetime",
  },
  "documents.storage_bytes": {
    label: "Document storage",
    description: "Total size of every stored document.",
    unit: "bytes",
    window: "lifetime",
  },
  "documents.file_bytes": {
    label: "Largest upload",
    description: "The size one uploaded file may be.",
    unit: "bytes",
    window: "item",
  },
  "ai.assistant_messages": {
    label: "Assistant messages",
    description: "Messages sent to the assistant this month.",
    unit: "count",
    window: "month",
  },
  "ai.spend_cents": {
    label: "AI spend",
    description: "Model usage cost this month.",
    unit: "cents",
    window: "month",
  },
};

export const PLAN_METER_ORDER: readonly string[] = Object.keys(PLAN_METERS);

function humanizeKey(key: string): string {
  const words = key.split(/[._]/).filter(Boolean).join(" ").trim();
  if (words === "") {
    return "Usage";
  }
  return words.charAt(0).toUpperCase() + words.slice(1);
}

export function planMeterDefinition(meterKey: string): PlanMeterDefinition {
  return (
    PLAN_METERS[meterKey] ?? {
      label: humanizeKey(meterKey),
      description: "",
      unit: "count",
      window: "lifetime",
    }
  );
}

export function planMeterLabel(meterKey: string): string {
  return planMeterDefinition(meterKey).label;
}

const countFormatter = new Intl.NumberFormat();

export function formatPlanMeterValue(meterKey: string, value: number): string {
  const { unit } = planMeterDefinition(meterKey);
  if (unit === "bytes") {
    return formatFileSize(value);
  }
  if (unit === "cents") {
    return formatCurrency(value / 100);
  }
  return countFormatter.format(value);
}

/** Sorts usage rows into the catalog order, with meters the client does not know last. */
export function comparePlanMeters(a: string, b: string): number {
  const ai = PLAN_METER_ORDER.indexOf(a);
  const bi = PLAN_METER_ORDER.indexOf(b);
  if (ai === -1 && bi === -1) {
    return a.localeCompare(b);
  }
  if (ai === -1) return 1;
  if (bi === -1) return -1;
  return ai - bi;
}

export type PlanCapabilityDefinition = {
  label: string;
  explanation: string;
};

export const PLAN_CAPABILITIES: Readonly<Record<string, PlanCapabilityDefinition>> = {
  "email.outbound": {
    label: "Outbound email",
    explanation:
      "Invoices, rate confirmations, tenders, detention notices and driver invites are not emailed from a demo organization. Sign-in and password email still arrive.",
  },
  integrations: {
    label: "Integrations",
    explanation:
      "Connections to Samsara, QuickBooks, Xero, EDI partners, Google Maps and other third-party services are not available on the demo.",
  },
  api_keys: {
    label: "API keys",
    explanation: "Programmatic access with API keys is not available on the demo.",
  },
  "agent.automation": {
    label: "Agent automation",
    explanation:
      "Scheduled and event-triggered agents, the morning briefing and background agent runs are not available on the demo. You can still talk to the assistant within the AI limits.",
  },
  "agent.web_search": {
    label: "Agent web search",
    explanation: "Agents cannot search the web from a demo organization.",
  },
  "carrier_intelligence.paid": {
    label: "Carrier intelligence lookups",
    explanation: "Paid carrier intelligence lookups are not available on the demo.",
  },
  document_intelligence: {
    label: "Document intelligence",
    explanation:
      "Automatic document extraction and classification are not available on the demo. Documents can still be uploaded and filed by hand.",
  },
  sms: {
    label: "Text messages",
    explanation: "Text messages are not sent from a demo organization.",
  },
  sso: {
    label: "Single sign-on",
    explanation: "Organization SSO and SCIM provisioning are not available on the demo.",
  },
};

export function planCapabilityDefinition(capability: string): PlanCapabilityDefinition {
  return (
    PLAN_CAPABILITIES[capability] ?? {
      label: humanizeKey(capability),
      explanation: "This part of Trenova is not included in your current plan.",
    }
  );
}

export const FREE_DEMO_PLAN_KEY = "free_demo";

export function planDisplayName(planKey: string, planName?: string): string {
  if (planName && planName.trim() !== "") {
    return planName;
  }
  if (planKey === FREE_DEMO_PLAN_KEY) {
    return "Free demo";
  }
  return planKey === "" ? "Current plan" : humanizeKey(planKey);
}
