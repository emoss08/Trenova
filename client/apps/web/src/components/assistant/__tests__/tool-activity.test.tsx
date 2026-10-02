import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it } from "vitest";
import { toolEffect, type ToolStep } from "../activity";
import { ToolActivity } from "../tool-activity";

afterEach(cleanup);

function step(overrides: Partial<ToolStep> & { id: string; name: string }): ToolStep {
  return {
    arguments: {},
    status: "done",
    content: "",
    effect: toolEffect(overrides.name, overrides.effect),
    summary: "",
    durationSeconds: null,
    ...overrides,
  };
}

/**
 * The transcript says why a call did not run in the words of the reason, in
 * the warning tone, and keeps the danger tone for a call that ran and broke.
 */
describe("ToolActivity refusals", () => {
  it("draws a denied write as not permitted, apart from a failed one", () => {
    render(
      <ToolActivity
        steps={[
          step({
            id: "c1",
            name: "update_worker",
            status: "failed",
            verdict: "denied",
            content: 'Tool "update_worker" is not permitted.',
          }),
          step({
            id: "c2",
            name: "create_shipment",
            status: "failed",
            verdict: "failed",
            content: 'Tool "create_shipment" failed: connection reset.',
          }),
        ]}
      />,
    );

    const denied = screen.getByText("Not permitted");
    expect(denied.className).toContain("text-warning");
    expect(denied.className).not.toContain("text-danger");

    const failed = screen.getByText("The change didn't go through");
    expect(failed.className).toContain("text-danger");
  });

  it.each([
    ["invalid", "Not accepted", "text-warning"],
    ["over_budget", "Out of budget", "text-warning"],
    ["duplicate", "Skipped (repeat)", "text-foreground-muted"],
  ] as const)("draws a %s call as %s", (verdict, phrase, tone) => {
    render(
      <ToolActivity
        steps={[step({ id: "c1", name: "assign_move", status: "failed", verdict, content: "no" })]}
      />,
    );

    expect(screen.getByText(phrase).className).toContain(tone);
  });

  it("says why inside a folded group of lookups", async () => {
    const user = userEvent.setup();
    render(
      <ToolActivity
        steps={[
          step({
            id: "c1",
            name: "get_worker",
            status: "failed",
            verdict: "denied",
            content: "no",
          }),
          step({ id: "c2", name: "get_shipment", status: "done", summary: "S-1001" }),
        ]}
      />,
    );

    await user.click(screen.getByRole("button"));

    expect(screen.getByText("Not permitted").className).toContain("text-warning");
  });
});

/**
 * In the transcript a reply's work is one quiet line — how many steps and how
 * long — that opens onto the rows; a surface that wants the rows at once
 * leaves it unfolded.
 */
describe("ToolActivity folded", () => {
  it("folds a saved reply's steps behind one summary line", async () => {
    const user = userEvent.setup();
    render(
      <ToolActivity
        folded
        steps={[
          step({
            id: "c1",
            name: "get_customer",
            summary: "Peak Distributing",
            durationSeconds: 2,
          }),
          step({ id: "c2", name: "open_page", summary: "Report library", durationSeconds: 1 }),
        ]}
      />,
    );

    const summary = screen.getByRole("button", { name: /Worked through 2 steps · 3s/ });
    expect(summary).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByText("Opened Report library")).toBeNull();

    await user.click(summary);

    expect(summary).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByText("Opened Report library")).toBeInTheDocument();
    expect(screen.getByText("Looked up Peak Distributing")).toBeInTheDocument();
  });

  it("keeps the rows in view while the reply is still being written", () => {
    render(
      <ToolActivity
        folded
        live
        steps={[step({ id: "c1", name: "get_customer", summary: "Peak Distributing" })]}
      />,
    );

    expect(screen.queryByRole("button", { name: /Worked through/ })).toBeNull();
    expect(screen.getByText("Looked up Peak Distributing")).toBeInTheDocument();
  });
});
