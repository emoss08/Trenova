import type { InboundClassification, InboundMessageStatus } from "@/lib/graphql/inbox";
import type { BadgeVariant } from "@trenova/shared/types/badge";
import {
  CircleHelpIcon,
  ClockAlertIcon,
  FileCheckIcon,
  FileSignatureIcon,
  PackagePlusIcon,
  ReceiptTextIcon,
  RadarIcon,
  type LucideIcon,
} from "lucide-react";

/** Every kind a message can be read as, in the order the server lists them. */
export const INBOUND_CLASSIFICATIONS = [
  "Tender",
  "RateConfirmation",
  "ProofOfDelivery",
  "Invoice",
  "StatusRequest",
  "DetentionDispute",
  "Other",
] as const satisfies readonly InboundClassification[];

/** A glyph per kind, so the rail and a row can be told apart at a glance. */
export const CLASSIFICATION_ICON: Record<InboundClassification, LucideIcon> = {
  Tender: PackagePlusIcon,
  RateConfirmation: FileSignatureIcon,
  ProofOfDelivery: FileCheckIcon,
  Invoice: ReceiptTextIcon,
  StatusRequest: RadarIcon,
  DetentionDispute: ClockAlertIcon,
  Other: CircleHelpIcon,
};

/**
 * A classification is a category, not a severity: a tender is not worse than
 * an invoice. Each one takes a categorical accent so nothing about the colour
 * implies an ordering.
 */
export const CLASSIFICATION_VARIANT: Record<InboundClassification, BadgeVariant> = {
  Tender: "accent-violet",
  RateConfirmation: "accent-teal",
  ProofOfDelivery: "accent-emerald",
  Invoice: "accent-amber",
  StatusRequest: "accent-sky",
  DetentionDispute: "accent-rose",
  Other: "accent-slate",
};

/**
 * A status is a lifecycle phase, and the phase decides the tone — so a status
 * added later cannot pick its own colour.
 */
export const STATUS_VARIANT: Record<InboundMessageStatus, BadgeVariant> = {
  Received: "neutral",
  Processing: "info",
  Classified: "info",
  InReview: "warning",
  Actioned: "success",
  Ignored: "neutral",
  Quarantined: "danger",
};

/** The status in the reader's words. The enum's own spelling is not English. */
export function statusLabel(t: (value: string) => string, status: InboundMessageStatus): string {
  switch (status) {
    case "Received":
      return t("Received");
    case "Processing":
      return t("Being read");
    case "Classified":
      return t("Read");
    case "InReview":
      return t("Waiting on you");
    case "Actioned":
      return t("Handled");
    case "Ignored":
      return t("Ignored");
    case "Quarantined":
      return t("Held back");
  }
}

export function classificationLabel(
  t: (value: string) => string,
  classification: InboundClassification,
): string {
  switch (classification) {
    case "Tender":
      return t("Tender");
    case "RateConfirmation":
      return t("Rate confirmation");
    case "ProofOfDelivery":
      return t("Proof of delivery");
    case "Invoice":
      return t("Invoice");
    case "StatusRequest":
      return t("Status request");
    case "DetentionDispute":
      return t("Detention dispute");
    case "Other":
      return t("Other");
  }
}
