import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { BadgeAttrProps } from "@trenova/shared/lib/status-phase";
import type { AiCorrectionOutcome } from "@trenova/graphql/generated/graphql";
import { defineLabels } from "@trenova/shared/i18n/labels";

/** How far back production accuracy is read. */
export const EXTRACTION_WINDOW_DAYS = 30;

export const EXTRACTION_STALE_MS = 30_000;

export const OUTCOME: Record<AiCorrectionOutcome, BadgeAttrProps> = {
  Correct: { phase: "complete", text: "Correct", description: "Read right" },
  Corrected: { phase: "failed", text: "Corrected", description: "A person changed it" },
  Missed: { phase: "attention", text: "Missed", description: "Not read, but a person entered it" },
  Unconfirmed: {
    phase: "closed",
    text: "Unconfirmed",
    description: "Read, but left empty on the record, so neither right nor wrong",
  },
  Unscored: {
    phase: "draft",
    text: "Unscored",
    description: "Present on both sides but could not be compared",
  },
};

const FIELD_LABEL: Record<string, string> = defineLabels({
  referenceNumber: "Reference number",
  rate: "Rate",
  weight: "Weight",
  pieceCount: "Pieces",
  commodity: "Commodity",
  shipper: "Shipper",
  consignee: "Consignee",
  pickupWindow: "Pickup date",
  deliveryWindow: "Delivery date",
  name: "Name",
  addressLine1: "Address",
  city: "City",
  state: "State",
  postalCode: "ZIP",
  date: "Date",
  appointmentRequired: "Appointment",
});

const STOP_KEY = /^stops\.(pickup|delivery)(?:\[(\d+)\])?\.(\w+)$/;

/**
 * A field key as a person reads it: "rate" is "Rate", a grouped stop field
 * "stops.pickup.city" is "Pickup city", and one stop's field
 * "stops.delivery[1].city" is "Delivery 2 city".
 */
export function fieldLabel(key: string, t: TranslateFn): string {
  const stop = STOP_KEY.exec(key);
  if (!stop) {
    return FIELD_LABEL[key] ? FIELD_LABEL[key] : key;
  }

  const [, role, index, field] = stop;
  const fieldText = FIELD_LABEL[field] ? FIELD_LABEL[field] : field;
  const roleText = role === "pickup" ? t("Pickup") : t("Delivery");
  const inSentence = fieldText === fieldText.toUpperCase() ? fieldText : fieldText.toLowerCase();
  if (index === undefined) {
    return t("{0} {1}", roleText, inSentence);
  }

  return t("{0} {1} {2}", roleText, Number(index) + 1, inSentence);
}

/** A model name, or what produced a draft that no model read. */
export function modelLabel(model: string, t: TranslateFn): string {
  return model.trim() === "" ? t("Rules only") : model;
}

/** A document kind as a person reads it. */
export function documentKindLabel(kind: string, t: TranslateFn): string {
  switch (kind) {
    case "RateConfirmation":
      return t("Rate confirmation");
    case "BillOfLading":
      return t("Bill of lading");
    case "ProofOfDelivery":
      return t("Proof of delivery");
    case "Invoice":
      return t("Invoice");
    case "":
      return t("Unclassified");
    default:
      return kind;
  }
}

/** Accuracy tone: good at 90%, worth a look under 75%. */
export function accuracyTone(
  accuracy: number,
  scored: number,
): "success" | "warning" | "danger" | undefined {
  if (scored === 0) return undefined;
  if (accuracy >= 0.9) return "success";
  if (accuracy >= 0.75) return "warning";
  return "danger";
}
