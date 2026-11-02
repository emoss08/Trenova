import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { HoverGlideGroup } from "../hover-glide";

afterEach(cleanup);

function place(
  element: Element,
  rect: { left: number; top: number; width: number; height: number },
) {
  element.getBoundingClientRect = () =>
    ({
      ...rect,
      x: rect.left,
      y: rect.top,
      right: rect.left + rect.width,
      bottom: rect.top + rect.height,
      toJSON: () => ({}),
    }) as DOMRect;
}

function renderGroup() {
  const view = render(
    <HoverGlideGroup>
      <button type="button" data-glide-target>
        <svg data-testid="shipments-icon" />
        Shipments
      </button>
      <hr data-testid="rule" />
      <button type="button" data-glide-target>
        Billing
      </button>
      <button type="button">Account</button>
    </HoverGlideGroup>,
  );
  const group = view.container.firstElementChild as HTMLElement;
  place(group, { left: 100, top: 200, width: 48, height: 400 });
  place(screen.getByRole("button", { name: "Shipments" }), {
    left: 108,
    top: 204,
    width: 32,
    height: 32,
  });
  place(screen.getByRole("button", { name: "Billing" }), {
    left: 108,
    top: 250,
    width: 32,
    height: 32,
  });
  const glide = group.querySelector('[data-slot="hover-glide"]') as HTMLElement;
  return { group, glide };
}

/**
 * The rail draws one hover background per group and moves it: hovering a
 * tile puts it there, hovering the next slides it across, leaving fades it.
 */
describe("the hover glide", () => {
  it("appears in place over the first tile the pointer reaches", () => {
    const { glide } = renderGroup();
    expect(glide).toHaveAttribute("data-state", "hidden");

    fireEvent.pointerOver(screen.getByRole("button", { name: "Shipments" }));

    expect(glide).toHaveAttribute("data-state", "shown");
    expect(glide.style.transform).toBe("translate(8px, 4px)");
    expect(glide.style.width).toBe("32px");
    expect(glide.style.height).toBe("32px");
    expect(glide).toHaveClass("transition-opacity");
    expect(glide).not.toHaveClass("transition-[transform,opacity]");
  });

  it("slides to the next tile hovered while it is shown", () => {
    const { glide } = renderGroup();

    fireEvent.pointerOver(screen.getByRole("button", { name: "Shipments" }));
    fireEvent.pointerOver(screen.getByRole("button", { name: "Billing" }));

    expect(glide.style.transform).toBe("translate(8px, 50px)");
    expect(glide).toHaveClass("transition-[transform,opacity]");
  });

  it("follows the tile when the pointer is over something inside it", () => {
    const { glide } = renderGroup();

    fireEvent.pointerOver(screen.getByTestId("shipments-icon"));

    expect(glide).toHaveAttribute("data-state", "shown");
    expect(glide.style.transform).toBe("translate(8px, 4px)");
  });

  it("stays on the last tile while the pointer crosses a rule between tiles", () => {
    const { glide } = renderGroup();

    fireEvent.pointerOver(screen.getByRole("button", { name: "Shipments" }));
    fireEvent.pointerOver(screen.getByTestId("rule"));

    expect(glide).toHaveAttribute("data-state", "shown");
    expect(glide.style.transform).toBe("translate(8px, 4px)");
  });

  it("fades when the pointer reaches a control that is not a tile", () => {
    const { glide } = renderGroup();

    fireEvent.pointerOver(screen.getByRole("button", { name: "Shipments" }));
    fireEvent.pointerOver(screen.getByRole("button", { name: "Account" }));

    expect(glide).toHaveAttribute("data-state", "hidden");
    expect(glide.style.opacity).toBe("0");
  });

  it("fades when the pointer leaves the group, then reappears in place without sliding", () => {
    const { group, glide } = renderGroup();

    fireEvent.pointerOver(screen.getByRole("button", { name: "Shipments" }));
    fireEvent.pointerLeave(group);
    expect(glide).toHaveAttribute("data-state", "hidden");
    expect(glide.style.opacity).toBe("0");

    fireEvent.pointerOver(screen.getByRole("button", { name: "Billing" }));
    expect(glide).toHaveAttribute("data-state", "shown");
    expect(glide.style.opacity).toBe("1");
    expect(glide.style.transform).toBe("translate(8px, 50px)");
    expect(glide).toHaveClass("transition-opacity");
    expect(glide).not.toHaveClass("transition-[transform,opacity]");
  });
});
