import type { FormulaLibraryRecord } from "@/lib/graphql/formula-library";
import { describe, expect, it } from "vitest";
import { reviewEntriesOf } from "../library-history-pane";

function review(id: string, decision: string, round = 1): FormulaLibraryRecord["reviews"][number] {
  return {
    id,
    libraryId: "fl_1",
    round,
    decision: decision as FormulaLibraryRecord["reviews"][number]["decision"],
    actorId: "usr_1",
    comment: "",
    baseVersionNumber: 0,
    createdAt: 1_700_000_000,
    requestedReviewerId: null,
    reviewerNote: "",
    actor: { id: "usr_1", name: "Ada", username: "ada", profilePicUrl: "" },
    requestedReviewer: null,
  };
}

describe("reviewEntriesOf", () => {
  it("keeps every review whose decision the client knows, in the record's order", () => {
    const library = {
      reviews: [review("r2", "Approved"), review("r1", "Submitted")],
    } as FormulaLibraryRecord;
    expect(reviewEntriesOf(library).map((entry) => [entry.id, entry.decision])).toEqual([
      ["r2", "Approved"],
      ["r1", "Submitted"],
    ]);
  });

  it("leaves out a decision the client does not know rather than drawing it wrong", () => {
    const library = { reviews: [review("r1", "Escalated")] } as FormulaLibraryRecord;
    expect(reviewEntriesOf(library)).toEqual([]);
  });
});
