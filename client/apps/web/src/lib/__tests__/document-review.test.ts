import { describe, expect, it } from "vitest";
import {
  canApproveDocument,
  canRejectDocument,
  documentReviewState,
  rejectDocumentFormSchema,
  unmetRequirementStanding,
} from "../document-review";

const base = { isCurrentVersion: true, approvedAt: null } as const;

describe("documentReviewState", () => {
  it("reads the review state from status and approval", () => {
    expect(documentReviewState({ ...base, status: "Active" })).toBe("unreviewed");
    expect(documentReviewState({ ...base, status: "Active", approvedAt: 100 })).toBe("approved");
    expect(documentReviewState({ ...base, status: "Rejected" })).toBe("rejected");
    expect(documentReviewState({ ...base, status: "PendingApproval" })).toBe("awaiting");
    expect(documentReviewState({ ...base, status: "Archived" })).toBe("inactive");
    expect(documentReviewState({ ...base, isCurrentVersion: false, status: "Active" })).toBe(
      "inactive",
    );
  });

  it("offers only the decisions that change something", () => {
    expect(canApproveDocument({ ...base, status: "Rejected" })).toBe(true);
    expect(canRejectDocument({ ...base, status: "Rejected" })).toBe(false);
    expect(canApproveDocument({ ...base, status: "Active", approvedAt: 100 })).toBe(false);
    expect(canRejectDocument({ ...base, status: "Active", approvedAt: 100 })).toBe(true);
    expect(canApproveDocument({ ...base, isCurrentVersion: false, status: "Rejected" })).toBe(
      false,
    );
  });
});

describe("rejectDocumentFormSchema", () => {
  it("requires a reason within the limit", () => {
    expect(rejectDocumentFormSchema.safeParse({ reason: "   " }).success).toBe(false);
    expect(rejectDocumentFormSchema.safeParse({ reason: "x".repeat(1001) }).success).toBe(false);
    expect(rejectDocumentFormSchema.safeParse({ reason: "Wrong load" }).success).toBe(true);
  });
});

describe("unmetRequirementStanding", () => {
  it("names the most serious reason an attached document does not count", () => {
    expect(
      unmetRequirementStanding({
        satisfied: false,
        ineligibleDocuments: [
          { documentId: "doc_1", standing: "expired" },
          { documentId: "doc_2", standing: "rejected" },
        ],
      }),
    ).toBe("rejected");
    expect(unmetRequirementStanding({ satisfied: false, ineligibleDocuments: [] })).toBeNull();
    expect(
      unmetRequirementStanding({
        satisfied: true,
        ineligibleDocuments: [{ documentId: "doc_1", standing: "rejected" }],
      }),
    ).toBeNull();
  });
});
