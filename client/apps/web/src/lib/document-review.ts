import type { StatusPhase } from "@trenova/shared/lib/status-phase";
import type { Document } from "@trenova/shared/types/document";
import type {
  ShipmentBillingDocumentStanding,
  ShipmentBillingRequirement,
} from "@trenova/shared/types/shipment";
import { z } from "zod";
import { translate } from "@trenova/shared/i18n/runtime";

export const DOCUMENT_REJECTION_REASON_MAX_LENGTH = 1000;

export const rejectDocumentFormSchema = z.object({
  reason: z
    .string()
    .trim()
    .min(1, { error: () => translate("Say why the document is rejected") })
    .max(DOCUMENT_REJECTION_REASON_MAX_LENGTH, {
      error: () => translate("Rejection reason cannot be longer than 1000 characters"),
    }),
});

export type RejectDocumentFormValues = z.infer<typeof rejectDocumentFormSchema>;

export type DocumentReviewState = "approved" | "rejected" | "awaiting" | "unreviewed" | "inactive";

export function documentReviewState(
  document: Pick<Document, "status" | "approvedAt" | "isCurrentVersion">,
): DocumentReviewState {
  if (!document.isCurrentVersion || document.status === "Archived") {
    return "inactive";
  }

  switch (document.status) {
    case "Rejected":
      return "rejected";
    case "Draft":
    case "Pending":
    case "PendingApproval":
      return "awaiting";
    case "Active":
      return document.approvedAt ? "approved" : "unreviewed";
    default:
      return "inactive";
  }
}

export function canApproveDocument(
  document: Pick<Document, "status" | "approvedAt" | "isCurrentVersion">,
): boolean {
  const state = documentReviewState(document);
  return state === "rejected" || state === "awaiting" || state === "unreviewed";
}

export function canRejectDocument(
  document: Pick<Document, "status" | "approvedAt" | "isCurrentVersion">,
): boolean {
  const state = documentReviewState(document);
  return state === "approved" || state === "awaiting" || state === "unreviewed";
}

export const DOCUMENT_REVIEW_PHASE: Record<
  Exclude<DocumentReviewState, "unreviewed" | "inactive">,
  StatusPhase
> = {
  approved: "complete",
  rejected: "failed",
  awaiting: "awaiting",
};

const INELIGIBLE_STANDING_PRIORITY: ShipmentBillingDocumentStanding[] = [
  "rejected",
  "expired",
  "pending_review",
  "inactive",
];

export function unmetRequirementStanding(
  requirement: Pick<ShipmentBillingRequirement, "satisfied" | "ineligibleDocuments">,
): ShipmentBillingDocumentStanding | null {
  if (requirement.satisfied) {
    return null;
  }

  const standings = new Set(requirement.ineligibleDocuments.map((item) => item.standing));
  return INELIGIBLE_STANDING_PRIORITY.find((standing) => standings.has(standing)) ?? null;
}
