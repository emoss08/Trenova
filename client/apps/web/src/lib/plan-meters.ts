import { formatCurrency, formatFileSize } from "@trenova/shared/lib/utils";
import { translateLabel } from "@trenova/shared/i18n/labels";
import { translate } from "@trenova/shared/i18n/runtime";

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
    return translate("Usage");
  }
  return translateLabel(words.charAt(0).toUpperCase() + words.slice(1));
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

/**
 * What each capability a plan can withhold is called, keyed as the server names it
 * (platformplan.Capability). Why a plan withholds one is the edition's to explain.
 */
export const PLAN_CAPABILITY_LABELS: Readonly<Record<string, { label: string }>> = {
  "email.outbound": { label: "Outbound email" },
  integrations: { label: "Integrations" },
  api_keys: { label: "API keys" },
  "agent.automation": { label: "Agent automation" },
  "agent.web_search": { label: "Agent web search" },
  "carrier_intelligence.paid": { label: "Carrier intelligence lookups" },
  document_intelligence: { label: "Document intelligence" },
  sms: { label: "Text messages" },
  sso: { label: "Single sign-on" },
};

export function planCapabilityLabel(capability: string): string {
  return PLAN_CAPABILITY_LABELS[capability]?.label ?? humanizeKey(capability);
}

export function planDisplayName(planKey: string, planName?: string): string {
  if (planName && planName.trim() !== "") {
    return planName;
  }
  return planKey === "" ? translate("Current plan") : humanizeKey(planKey);
}
