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
