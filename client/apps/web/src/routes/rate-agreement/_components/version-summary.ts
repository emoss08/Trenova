import type { RateAgreementVersion } from "@trenova/shared/types/rate";

/**
 * Labels for the header terms a version can record, in the words the form uses.
 * A path missing from this map falls back to its raw key, so a newly versioned
 * field degrades to legible-but-plain rather than to nothing.
 */
const HEADER_FIELD_LABELS: Record<string, string> = {
  partyType: "Party type",
  customerId: "Customer",
  carrierId: "Carrier",
  code: "Code",
  name: "Name",
  description: "Description",
  agreementType: "Agreement type",
  contractRef: "Contract reference",
  documentId: "Contract document",
  priority: "Priority",
  agreementEffectiveFrom: "Effective from",
  agreementEffectiveTo: "Effective to",
  autoRenew: "Auto-renew",
  renewalNoticeDays: "Renewal notice days",
  billToCustomerId: "Bill-to customer",
  currency: "Currency",
  defaultMinCharge: "Default minimum charge",
  defaultMaxCharge: "Default maximum charge",
  roundingMode: "Rounding",
  roundingPrecision: "Rounding precision",
  marginFloorPercent: "Margin floor",
  maxPayPercentOfSell: "Max pay percent of sell",
};

const ACCESSORIAL_FIELD_LABELS: Record<string, string> = {
  method: "method",
  rateUnit: "rate unit",
  amount: "amount",
  waived: "waiver",
  autoApply: "auto-apply",
  applyCondition: "condition",
  freeUnits: "free units",
  maxAmount: "cap",
  formulaTemplateId: "rating method",
  serviceTypeIds: "applicability",
  shipmentTypeIds: "applicability",
  appliesFrom: "window",
  appliesTo: "window",
};

const ACCESSORIAL_PREFIX = "accessorialTerms.";

type FieldChange = { from?: unknown; to?: unknown };

function accessorialPhrase(
  path: string,
  change: FieldChange,
  names: Record<string, string>,
): string {
  const rest = path.slice(ACCESSORIAL_PREFIX.length);
  const dot = rest.indexOf(".");
  const chargeId = dot === -1 ? rest : rest.slice(0, dot);
  const field = dot === -1 ? "" : rest.slice(dot + 1);
  const name = names[chargeId];

  if (!field) {
    // A whole term appearing or vanishing is the schedule itself changing.
    const added = change.to != null;
    if (added) return name ? `Added ${name} accessorial` : "Added accessorial";
    return name ? `Removed ${name} accessorial` : "Removed accessorial";
  }

  const fieldLabel = ACCESSORIAL_FIELD_LABELS[field] ?? field;
  return name ? `${name} ${fieldLabel}` : `Accessorial ${fieldLabel}`;
}

/**
 * One line saying what a version changed, written for the person reading the
 * history — field names in the form's words, accessorials named by their
 * charge's code, and never a record id.
 */
export function describeVersion(version: RateAgreementVersion): string {
  if (version.changeMessage) return version.changeMessage;

  const summary = version.changeSummary ?? {};
  const names = version.accessorialNames ?? {};

  const phrases: string[] = [];
  for (const [path, change] of Object.entries(summary)) {
    let phrase: string;
    if (path.startsWith(ACCESSORIAL_PREFIX)) {
      phrase = accessorialPhrase(path, change as FieldChange, names);
    } else if (path === "fuelTerms" || path.startsWith("fuelTerms.")) {
      phrase = "Fuel terms";
    } else {
      phrase = HEADER_FIELD_LABELS[path] ?? path;
    }
    if (!phrases.includes(phrase)) {
      phrases.push(phrase);
    }
  }

  if (phrases.length === 0) return "—";

  return phrases.join(", ");
}
