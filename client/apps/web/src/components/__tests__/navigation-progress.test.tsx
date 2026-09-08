import { act, render } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { NavigationProgress } from "../navigation-progress";

const navigationState = vi.hoisted(() => ({ state: "idle" as "idle" | "loading" }));

vi.mock("react-router", () => ({
  useNavigation: () => navigationState,
}));

describe("NavigationProgress", () => {
  afterEach(() => {
    vi.useRealTimers();
    navigationState.state = "idle";
  });

  it("renders nothing while idle", () => {
    const { container } = render(<NavigationProgress />);
    expect(container.firstChild).toBeNull();
  });

  it("shows the flowing bar while a navigation is pending and fades it out when done", () => {
    vi.useFakeTimers();
    navigationState.state = "loading";
    const { container, rerender } = render(<NavigationProgress />);

    const bar = container.querySelector<HTMLDivElement>("div[aria-hidden] > div");
    expect(bar).not.toBeNull();
    expect(bar?.style.transform).toBe("translateX(-85%)");
    expect(bar?.style.opacity).toBe("1");
    expect(container.querySelector(".nav-progress-flow")).not.toBeNull();
    expect(container.querySelector(".nav-progress-sheen")).not.toBeNull();

    act(() => {
      vi.advanceTimersByTime(600);
    });
    const [, mid] = /translateX\(([-\d.]+)%\)/.exec(bar?.style.transform ?? "") ?? [];
    expect(Number(mid)).toBeGreaterThan(-85);
    expect(Number(mid)).toBeLessThanOrEqual(-6);

    navigationState.state = "idle";
    rerender(<NavigationProgress />);
    expect(bar?.style.transform).toBe("translateX(0%)");
    expect(bar?.style.opacity).toBe("0");

    act(() => {
      vi.advanceTimersByTime(400);
    });
    expect(container.firstChild).toBeNull();
  });
});
