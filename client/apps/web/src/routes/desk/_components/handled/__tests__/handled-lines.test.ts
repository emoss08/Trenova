import type {
  DeskActivityChange,
  DeskActivityDetail,
  DeskActivityOutcome,
} from "@/lib/graphql/desk-attention";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import { runTrail } from "../handled-lines";

const t = ((text?: string | null, ...args: unknown[]) =>
  (text ?? "")
    .replace(
      /\{(\d+), plural, one \{([^}]*)\} other \{([^}]*)\}\}/g,
      (_, index: string, one: string, other: string) => {
        const count = Number(args[Number(index)]);
        return (count === 1 ? one : other).replace("#", String(count));
      },
    )
    .replace(/\{(\d+)\}/g, (_, index: string) => String(args[Number(index)]))) as TranslateFn;

function change(
  proposalId: string,
  proposedAt: number,
  at: number,
  outcome: DeskActivityOutcome,
): DeskActivityChange {
  return {
    proposalId,
    record: null,
    what: "Filed document",
    why: "",
    proposedAt,
    at,
    outcome,
    error: "",
  };
}

function detail(
  outcome: DeskActivityOutcome,
  changes: DeskActivityChange[],
  decidedBy: { id: string; name: string } | null = null,
): DeskActivityDetail {
  return {
    entry: {
      id: "ap_1",
      outcome,
      at: 0,
      agent: { id: "agd_1", name: "Billing" },
      runId: "run_1",
      planId: null,
      title: "Filed 2 documents",
      detail: "",
      count: changes.length,
      records: [],
      recordsMore: 0,
      decidedBy,
      proposalIds: changes.map((entry) => entry.proposalId),
    },
    tool: "file document",
    changes,
    ranBefore: 0,
    ranSince: 0,
  } as unknown as DeskActivityDetail;
}

describe("runTrail", () => {
  it("runs from the earliest proposal to the latest settlement for work done on its own", () => {
    const steps = runTrail(
      t,
      detail("OnItsOwn", [change("a", 120, 300, "OnItsOwn"), change("b", 100, 200, "OnItsOwn")]),
    );

    expect(steps).toEqual([
      { key: "proposed", at: 100, text: "Billing proposed 2 changes" },
      { key: "rule", at: 100, text: "AI control let it run without asking" },
      { key: "settled", at: 300, text: "The server confirmed all 2 changes" },
    ]);
  });

  it("names the person who approved it", () => {
    const steps = runTrail(
      t,
      detail("Approved", [change("a", 100, 200, "Approved")], { id: "usr_1", name: "Maya Chen" }),
    );

    expect(steps.map((step) => step.text)).toEqual([
      "Billing proposed 1 change",
      "Maya Chen approved it",
      "The server confirmed the change",
    ]);
  });

  it("says a preview changed nothing and claims no approval", () => {
    const steps = runTrail(t, detail("Previewed", [change("a", 100, 200, "Previewed")]));

    expect(steps.map((step) => step.key)).toEqual(["proposed", "settled"]);
    expect(steps[1]?.text).toBe("Previewed in shadow; nothing was changed");
  });

  it("counts what went through when some writes were refused", () => {
    const steps = runTrail(
      t,
      detail("Approved", [
        change("a", 100, 200, "Approved"),
        change("b", 100, 210, "Failed"),
        change("c", 100, 220, "Approved"),
      ]),
    );

    expect(steps.at(-1)).toEqual({
      key: "settled",
      at: 220,
      text: "The server confirmed 2 of 3; the rest did not go through",
    });
  });

  it("says nothing changed when every write was refused", () => {
    const steps = runTrail(t, detail("Failed", [change("a", 100, 200, "Failed")]));

    expect(steps.at(-1)?.text).toBe("The server refused it; nothing changed");
  });

  it("has nothing to tell for a line with no writes", () => {
    expect(runTrail(t, detail("OnItsOwn", []))).toEqual([]);
  });
});
