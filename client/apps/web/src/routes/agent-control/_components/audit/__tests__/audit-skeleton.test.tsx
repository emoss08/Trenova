import { cleanup, render } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { AuditTabSkeleton } from "../audit-skeleton";

afterEach(() => {
  cleanup();
});

describe("AuditTabSkeleton", () => {
  it("draws the chain's head over the trail's scope bar and its ten columns", () => {
    const { container } = render(<AuditTabSkeleton view="trail" />);

    const root = container.firstElementChild;
    expect(root).toHaveAttribute("aria-busy", "true");
    expect(root).toHaveAttribute("aria-label", "Loading the audit trail");
    expect(root?.querySelectorAll(".hero .hc")).toHaveLength(1);
    expect(root?.querySelectorAll(".figs .fg")).toHaveLength(3);
    expect(root?.querySelectorAll("th")).toHaveLength(10);
  });

  it("draws the exports as their own eleven-column table", () => {
    const { container } = render(<AuditTabSkeleton view="exports" />);

    expect(container.querySelectorAll(".figs .fg")).toHaveLength(3);
    expect(container.querySelectorAll("th")).toHaveLength(11);
  });

  it("puts no visible text on the page", () => {
    const { container } = render(<AuditTabSkeleton view="trail" />);

    expect(container.textContent).toBe("");
  });
});
