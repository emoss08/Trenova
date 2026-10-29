import type { BillingQueueItem, BillingQueueSummary } from "@trenova/shared/types/billing-queue";
import { describe, expect, it } from "vitest";
import {
  barReason,
  bulkSplit,
  initials,
  itemStage,
  selectionState,
  stageLabel,
  toggleAll,
  toggleOne,
  undoSecondsLeft,
} from "../billing-item-checks";

const t = ((text: string, ...args: unknown[]) =>
  text
    .replace(
      /\{0, plural, one \{([^}]*)\} other \{([^}]*)\}\}/u,
      (_m, one: string, other: string) =>
        (args[0] === 1 ? one : other).replace("#", String(args[0])),
    )
    .replace(/\{(\d)\}/gu, (_m, at) => String(args[Number(at)]))) as never;

function item(extra: Partial<BillingQueueItem> = {}): BillingQueueItem {
  return {
    id: "bqi_1",
    shipmentId: "shp_1",
    status: "InReview",
    billType: "Invoice",
    number: "INV-24101",
    createdAt: 0,
    detentionHolds: [],
    review: {
      checks: [],
      needsCount: 0,
      ready: true,
      blocker: "",
      issues: [],
      documents: [],
      billerName: "Avery Lane",
    },
    ...extra,
  } as unknown as BillingQueueItem;
}

function summary(id: string, extra: Partial<BillingQueueSummary> = {}): BillingQueueSummary {
  return {
    id,
    number: id,
    status: "InReview",
    holdReasonCode: null,
    assignedBillerId: "usr_1",
    allocatedTotalAmount: 100,
    needsCount: 0,
    ready: true,
    ...extra,
  } as BillingQueueSummary;
}

describe("item stage and status words", () => {
  it("reads both review statuses as ready for review", () => {
    expect(itemStage("ReadyForReview")).toBe("review");
    expect(itemStage("InReview")).toBe("review");
    expect(stageLabel("InReview", null, t)).toBe("Ready for review");
  });

  it("names the hold's reason on the pill", () => {
    expect(stageLabel("OnHold", "CustomerDispute", t)).toBe("On hold · Customer dispute");
    expect(stageLabel("OnHold", null, t)).toBe("On hold");
  });

  it("reads approved and posted as their own stages", () => {
    expect(itemStage("Approved")).toBe("approved");
    expect(stageLabel("Posted", null, t)).toBe("Posted");
    expect(itemStage("Canceled")).toBe("other");
  });
});

describe("the action bar's reason", () => {
  it("asks for a biller, then for the flagged check, then nothing", () => {
    const blocked = (blocker: string) =>
      barReason(item({ review: { ...item().review!, blocker, ready: blocker === "" } }), t);
    expect(blocked("biller")).toBe("Assign a biller first");
    expect(blocked("issue")).toBe("Settle the flagged check first");
    expect(blocked("")).toBe("");
  });

  it("says what posting does once approved, and what a hold needs", () => {
    expect(barReason(item({ status: "Approved" }), t)).toBe(
      "Posting sends it to the customer and can't be undone",
    );
    expect(barReason(item({ status: "OnHold" }), t)).toBe("Release the hold to continue");
  });
});

describe("bulk selection", () => {
  const rows = new Map([
    ["a", summary("a")],
    ["b", summary("b", { ready: false, needsCount: 2 })],
    ["c", summary("c", { status: "Approved", ready: false })],
    ["d", summary("d", { status: "OnHold", ready: false })],
  ]);

  it("splits the picked rows into ready, needing a person, and done or held", () => {
    const split = bulkSplit(["a", "b", "c", "d", "gone"], rows);
    expect(split.ready).toEqual(["a"]);
    expect(split.needs).toEqual(["b"]);
    expect(split.other).toBe(3);
  });

  it("sets apart rows whose only want is a biller, which approving can assign", () => {
    const unassigned = new Map([
      ["e", summary("e", { ready: false, needsCount: 1, assignedBillerId: null })],
      ["f", summary("f", { ready: false, needsCount: 2, assignedBillerId: null })],
    ]);
    const split = bulkSplit(["e", "f"], unassigned);
    expect(split.unassigned).toEqual(["e"]);
    expect(split.needs).toEqual(["f"]);
  });

  it("reads the header box as all, some or none", () => {
    expect(selectionState(["a", "b"], ["a", "b"])).toBe("all");
    expect(selectionState(["a", "b"], ["a"])).toBe("some");
    expect(selectionState(["a", "b"], [])).toBe("none");
    expect(selectionState([], [])).toBe("none");
  });

  it("toggles one row, and all rows from the header", () => {
    expect(toggleOne(["a"], "b")).toEqual(["a", "b"]);
    expect(toggleOne(["a", "b"], "a")).toEqual(["b"]);
    expect(toggleAll(["a", "b"], ["a"])).toEqual(["a", "b"]);
    expect(toggleAll(["a", "b"], ["a", "b"])).toEqual([]);
  });
});

describe("the undo countdown", () => {
  it("counts down to when the server commits, never below zero", () => {
    const commitAt = 1_000_006;
    expect(undoSecondsLeft(commitAt, 1_000_000_000)).toBe(6);
    expect(undoSecondsLeft(commitAt, 1_000_000_500)).toBe(6);
    expect(undoSecondsLeft(commitAt, 1_000_005_200)).toBe(1);
    expect(undoSecondsLeft(commitAt, 1_000_006_000)).toBe(0);
    expect(undoSecondsLeft(commitAt, 1_000_009_000)).toBe(0);
  });
});

describe("initials", () => {
  it("takes the first letter of the first two words", () => {
    expect(initials("Avery Lane")).toBe("AL");
    expect(initials("sam okafor jr")).toBe("SO");
  });
});
