import { describe, expect, it } from "vitest";
import { rowsLike } from "../batch-selection";
import type { DecisionRowView } from "../decision-presenters";

function row(id: string, toolName: string, batchable = true): DecisionRowView {
  return {
    id,
    kind: batchable ? "proposal" : "plan",
    title: toolName,
    summary: "",
    toolName,
    agent: null,
    createdAt: 1,
    stepCount: 0,
    batchable,
  };
}

describe("rowsLike", () => {
  const rows = [
    row("a", "post_invoice"),
    row("b", "send_invoice"),
    row("c", "post_invoice"),
    row("d", "plan", false),
  ];

  it("finds every batchable row of the focused row's tool, in queue order", () => {
    expect(rowsLike(rows, "c")).toEqual(["a", "c"]);
  });

  it("finds none for a lone tool, a plan or nothing focused", () => {
    expect(rowsLike(rows, "b")).toEqual([]);
    expect(rowsLike(rows, "d")).toEqual([]);
    expect(rowsLike(rows, null)).toEqual([]);
  });
});
