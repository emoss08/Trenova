import { hasRoute, layoutsAbove } from "@/test/route-tree";
import { DeskShellLayout } from "@/routes/desk/shell-layout";
import { describe, expect, it } from "vitest";

/**
 * Triage is worked in the Desk's room, beside the rest of what needs a person,
 * and that is the only place it lives.
 */
describe("triage route", () => {
  it("renders inside the Desk's shell", () => {
    expect(layoutsAbove("/desk/triage")).toContain(DeskShellLayout);
  });

  it("has one address, with none kept beside it", () => {
    expect(hasRoute("/triage")).toBe(false);
    expect(hasRoute("/inbox")).toBe(false);
  });
});
