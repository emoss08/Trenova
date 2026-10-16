import { layoutToastStack } from "@trenova/shared/components/ui/toast-stack";
import { describe, expect, it } from "vitest";

const base = { gap: 8, peek: 10, visible: 3, expandedVisible: 6 } as const;

describe("layoutToastStack", () => {
  it("collapses behind the front toast, each one peeking a little further and a little smaller", () => {
    const stack = layoutToastStack({
      ...base,
      heights: [64, 92, 48, 70],
      expanded: false,
      direction: 1,
    });

    expect(stack.slots).toEqual([
      { y: 0, scale: 1, height: 64, behind: false, hidden: false },
      { y: 10, scale: 0.95, height: 64, behind: true, hidden: false },
      { y: 20, scale: 0.9, height: 64, behind: true, hidden: false },
      { y: 20, scale: 0.9, height: 64, behind: true, hidden: true },
    ]);
    expect(stack.height).toBe(84);
  });

  it("opens into a column at each toast's own height with the gap between", () => {
    const stack = layoutToastStack({
      ...base,
      heights: [64, 92, 48],
      expanded: true,
      direction: 1,
    });

    expect(stack.slots.map((slot) => [slot.y, slot.height, slot.behind])).toEqual([
      [0, 64, false],
      [72, 92, false],
      [172, 48, false],
    ]);
    expect(stack.height).toBe(220);
  });

  it("rises from the bottom edge when the stack sits at the bottom of the screen", () => {
    const stack = layoutToastStack({ ...base, heights: [64, 92], expanded: true, direction: -1 });

    expect(stack.slots.map((slot) => slot.y)).toEqual([0, -72]);
    expect(stack.height).toBe(164);
  });

  it("hides what an open stack has no room for and leaves it out of the height", () => {
    const stack = layoutToastStack({
      ...base,
      expandedVisible: 2,
      heights: [40, 40, 40],
      expanded: true,
      direction: 1,
    });

    expect(stack.slots.map((slot) => slot.hidden)).toEqual([false, false, true]);
    expect(stack.height).toBe(88);
  });

  it("covers nothing when there are no toasts", () => {
    expect(layoutToastStack({ ...base, heights: [], expanded: false, direction: 1 })).toEqual({
      slots: [],
      height: 0,
    });
  });
});
