import { act, cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AssistantEdgeTab } from "../assistant-edge-tab";
import { AssistantLauncher } from "../assistant-launcher";
import { beaconState } from "../beacon-state";
import {
  LAUNCHER_GAP,
  beaconBottom,
  highestReach,
  reachOf,
  useClearOfLauncher,
  useLauncherClearanceStore,
} from "../launcher-clearance";

afterEach(() => {
  cleanup();
  useLauncherClearanceStore.setState({ bars: new Map() });
  vi.restoreAllMocks();
});

const IDLE = beaconState({
  pendingCount: 0,
  liveTurns: [],
  repliedAgentName: null,
  lastAgentName: "Billing exceptions",
});

function beaconBox() {
  return screen.getByRole("button", { name: /^Open the assistant/ }).closest(".as-beacon");
}

/** A bar that floats along the bottom of the screen, as Triage's action bar does. */
function BottomBar({ top, height }: { top: number; height: number }) {
  const clear = useClearOfLauncher();

  return (
    <div
      data-testid="bar"
      ref={(element) => {
        if (element !== null) {
          element.getBoundingClientRect = () =>
            ({ top, height, bottom: top + height, left: 0, right: 0, width: 0 }) as DOMRect;
        }
        return clear(element);
      }}
    />
  );
}

/**
 * The beacon sits in a corner of every page, and Triage floats its next step
 * along the bottom of the reader. Whatever the width, whichever corner the
 * beacon was dragged to and however wide it is while it speaks, the bar must
 * never end up under it.
 */
describe("the beacon and a bar along the bottom of the page", () => {
  it("measures how far up the screen a bar reaches, and nothing for a bar that draws nothing", () => {
    expect(reachOf({ top: 830, height: 52 }, 900)).toBe(70);
    expect(reachOf({ top: 830.4, height: 52 }, 900)).toBe(70);
    expect(reachOf({ top: 900, height: 0 }, 900)).toBe(0);
  });

  it("stands above the highest of several bars, and in its corner when there is none", () => {
    const bars = new Map([
      [Symbol("reader"), 70],
      [Symbol("list"), 64],
    ]);

    expect(highestReach(bars)).toBe(70);
    expect(highestReach(new Map())).toBe(0);
    expect(beaconBottom(20, 70)).toBe(70 + LAUNCHER_GAP);
    expect(beaconBottom(20, 0)).toBe(20);
  });

  it("lifts a beacon in a bottom corner above a mounted bar, and lets it back down when the bar goes", () => {
    vi.spyOn(window, "innerHeight", "get").mockReturnValue(900);
    const { rerender } = render(
      <>
        <AssistantLauncher beacon={IDLE} dock="bottom-left" onClick={vi.fn()} />
        <BottomBar top={830} height={52} />
      </>,
    );

    expect(beaconBox()).toHaveStyle({ bottom: `${70 + LAUNCHER_GAP}px` });

    rerender(<AssistantLauncher beacon={IDLE} dock="bottom-left" onClick={vi.fn()} />);

    expect(beaconBox()).toHaveStyle({ bottom: "20px" });
  });

  it("leaves a beacon in a top corner where it is", () => {
    act(() => useLauncherClearanceStore.getState().report(Symbol("bar"), 70));
    render(<AssistantLauncher beacon={IDLE} dock="top-right" onClick={vi.fn()} />);

    expect(beaconBox()?.getAttribute("style") ?? "").not.toContain("bottom");
  });

  it("lifts the hidden beacon's tab the same way", () => {
    act(() => useLauncherClearanceStore.getState().report(Symbol("bar"), 70));
    render(<AssistantEdgeTab beacon={IDLE} dock="bottom-right" onClick={vi.fn()} />);

    expect(screen.getByRole("button", { name: /^Open the assistant/ })).toHaveStyle({
      bottom: `${70 + LAUNCHER_GAP}px`,
    });
  });
});
