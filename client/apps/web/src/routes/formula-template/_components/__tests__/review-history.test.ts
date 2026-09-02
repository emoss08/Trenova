import type { FormulaTemplateReview } from "@trenova/shared/types/formula-template";
import { describe, expect, it } from "vitest";
import { describeReviewDecision, groupReviewRounds } from "../review-rounds";

function entry(
  round: number,
  decision: FormulaTemplateReview["decision"],
  createdAt: number,
  baseVersionNumber = 0,
): FormulaTemplateReview {
  return {
    id: `ftr_${round}_${createdAt}`,
    templateId: "ft_1",
    round,
    decision,
    actorId: null,
    comment: "",
    baseVersionNumber,
    createdAt,
    actor: null,
  };
}

describe("groupReviewRounds", () => {
  it("groups newest round first with each conversation reading oldest to newest", () => {
    const rounds = groupReviewRounds([
      entry(2, "Submitted", 400, 3),
      entry(1, "Approved", 300, 0),
      entry(1, "Submitted", 200, 0),
      entry(1, "ChangesRequested", 250, 0),
      entry(1, "Submitted", 100, 0),
    ]);

    expect(rounds.map((round) => round.round)).toEqual([2, 1]);
    expect(rounds[1].entries.map((e) => e.createdAt)).toEqual([100, 200, 250, 300]);
    expect(rounds[1].outcome).toBe("Approved");
    expect(rounds[0].outcome).toBeNull();
    expect(rounds[0].baseVersionNumber).toBe(3);
  });

  it("treats a round waiting after a change request as still open", () => {
    const rounds = groupReviewRounds([
      entry(1, "ChangesRequested", 200),
      entry(1, "Submitted", 100),
    ]);
    expect(rounds[0].outcome).toBeNull();
  });
});

describe("describeReviewDecision", () => {
  it("words every decision for a billing clerk", () => {
    expect(describeReviewDecision("ChangesRequested")).toEqual({
      label: "Changes requested",
      tone: "warning",
    });
    expect(describeReviewDecision("Expired").tone).toBe("muted");
  });
});
