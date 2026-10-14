import { cleanup, render } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { MemoryTabSkeleton } from "../memory-skeleton";

afterEach(() => {
  cleanup();
});

describe("MemoryTabSkeleton", () => {
  it("draws the lead line over the five columns the memory table opens with", () => {
    const { container } = render(<MemoryTabSkeleton />);

    const root = container.firstElementChild;
    expect(root).toHaveClass("tabp");
    expect(root).toHaveAttribute("aria-busy", "true");
    expect(root).toHaveAttribute("aria-label", "Loading memory");
    expect(root?.querySelectorAll(":scope > .lead")).toHaveLength(1);
    expect(root?.querySelectorAll(".sec th")).toHaveLength(5);
    expect(container.textContent).toBe("");
  });
});
