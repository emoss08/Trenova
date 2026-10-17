import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { OnItStrip, ringSlots } from "../on-it-strip";
import { watchtowerItem } from "./item-fixture";

const sizes = vi.hoisted(() => ({ width: 0 }));

vi.mock("@trenova/shared/hooks/use-element-size", () => ({
  useElementSize: () => [() => undefined, { width: sizes.width, height: 26 }],
}));

function working(count: number) {
  return Array.from({ length: count }, (_, index) =>
    watchtowerItem({
      id: `wt_${index}`,
      title: `Item ${index}`,
      agentWork: {
        agentId: `adef_${index}`,
        runId: `ar_${index}`,
        status: "Working",
        step: 0,
        steps: ["Read", "Decide", "Approve"],
        note: "",
        failed: false,
        handedAt: 1,
        agent: {
          id: `adef_${index}`,
          name: `Desk ${index}`,
          description: "",
          icon: "radar",
          accent: "sky",
          template: null,
        },
      },
    }),
  );
}

describe("ringSlots", () => {
  it("draws every ring that fits and no chip", () => {
    expect(ringSlots(136, 4)).toEqual({ shown: 4, more: 0 });
    expect(ringSlots(136, 2)).toEqual({ shown: 2, more: 0 });
  });

  it("gives one ring's place to the chip when they do not all fit", () => {
    expect(ringSlots(136, 8)).toEqual({ shown: 3, more: 5 });
    expect(ringSlots(60, 8)).toEqual({ shown: 1, more: 7 });
  });

  it("shows only the chip when there is room for one slot or none", () => {
    expect(ringSlots(26, 3)).toEqual({ shown: 0, more: 3 });
    expect(ringSlots(0, 3)).toEqual({ shown: 0, more: 3 });
  });
});

describe("OnItStrip", () => {
  it("draws the rings that fit, a chip for the rest and how many are working", () => {
    sizes.width = 136;
    render(<OnItStrip working={working(8)} selectedId={null} onSelect={() => undefined} />);

    expect(screen.getAllByRole("button", { name: /is on Item/ })).toHaveLength(3);
    expect(
      screen.getByRole("button", { name: "5 more agents working; open the next item" }),
    ).toBeDefined();
    expect(screen.getByText("+5")).toBeDefined();
    expect(screen.getByText("8 working")).toBeDefined();
  });

  it("opens the first item the rings leave out, then the next one", () => {
    sizes.width = 136;
    const onSelect = vi.fn();
    const items = working(8);
    const { rerender } = render(
      <OnItStrip working={items} selectedId={null} onSelect={onSelect} />,
    );

    fireEvent.click(screen.getByText("+5"));
    expect(onSelect).toHaveBeenLastCalledWith("wt_3");

    rerender(<OnItStrip working={items} selectedId="wt_3" onSelect={onSelect} />);
    fireEvent.click(screen.getByText("+5"));
    expect(onSelect).toHaveBeenLastCalledWith("wt_4");
  });

  it("draws every ring and no chip when they all fit", () => {
    sizes.width = 280;
    render(<OnItStrip working={working(2)} selectedId={null} onSelect={() => undefined} />);

    expect(screen.getAllByRole("button", { name: /is on Item/ })).toHaveLength(2);
    expect(screen.queryByText(/^\+\d/)).toBeNull();
  });
});
