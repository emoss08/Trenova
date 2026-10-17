import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";

afterEach(cleanup);

function badge(container: HTMLElement): HTMLElement {
  const element = container.querySelector<HTMLElement>('[data-slot="nova-badge"]');
  if (!element) throw new Error("no badge rendered");
  return element;
}

describe("NovaBadge", () => {
  it("is decoration unless it is named", () => {
    const { container } = render(<NovaBadge />);

    expect(badge(container).getAttribute("aria-hidden")).toBe("true");
    expect(badge(container).getAttribute("role")).toBeNull();
  });

  it("is an image named for assistive technology when given a label", () => {
    render(<NovaBadge label="Nova" />);

    const named = screen.getByRole("img", { name: "Nova" });
    expect(named.getAttribute("aria-hidden")).toBeNull();
  });

  it.each([
    ["xs", "size-3"],
    ["sm", "size-3.5"],
    ["md", "size-4"],
    ["lg", "size-5"],
    ["xl", "size-6"],
  ] as const)("draws the %s size at %s", (size, sizeClass) => {
    const { container } = render(<NovaBadge size={size} />);

    expect(badge(container).classList).toContain(sizeClass);
    expect(badge(container).classList).toContain("nova-badge");
  });

  it("is medium when no size is given", () => {
    const { container } = render(<NovaBadge />);

    expect(badge(container).classList).toContain("size-4");
  });

  it("stays still at rest and turns only while asked to", () => {
    const still = badge(render(<NovaBadge />).container);
    expect(still.className).not.toContain("animate-nova-spin");
    cleanup();

    const working = badge(render(<NovaBadge spin="always" />).container);
    expect(working.classList).toContain("animate-nova-spin");
    cleanup();

    const hovered = badge(render(<NovaBadge spin="hover" />).container);
    expect(hovered.classList).toContain("group-hover:animate-nova-spin");
    expect(hovered.classList).not.toContain("animate-nova-spin");
  });

  it("lets a caller's class win over its own size", () => {
    const { container } = render(<NovaBadge size="lg" className="size-3.5" />);

    expect(badge(container).classList).toContain("size-3.5");
    expect(badge(container).classList).not.toContain("size-5");
  });
});
