import { describe, expect, it, vi } from "vitest";
import { preloadRulesWhenIdle } from "./preload";

function manualEnvironment(load: () => Promise<unknown>) {
  const loaded: Array<() => void> = [];
  const idle: Array<() => void> = [];
  const report = vi.fn();
  preloadRulesWhenIdle({
    load,
    report,
    whenLoaded: (task) => loaded.push(task),
    whenIdle: (task) => idle.push(task),
  });
  return { loaded, idle, report };
}

describe("preloadRulesWhenIdle", () => {
  it("loads only after the page has loaded and the browser is idle", () => {
    const load = vi.fn(() => Promise.resolve());
    const { loaded, idle } = manualEnvironment(load);

    expect(load).not.toHaveBeenCalled();
    expect(idle).toHaveLength(0);

    loaded[0]!();
    expect(load).not.toHaveBeenCalled();
    expect(idle).toHaveLength(1);

    idle[0]!();
    expect(load).toHaveBeenCalledTimes(1);
  });

  it("reports a failed load instead of throwing it", async () => {
    const failure = new Error("offline");
    const { loaded, idle, report } = manualEnvironment(() => Promise.reject(failure));

    loaded[0]!();
    idle[0]!();
    await vi.waitFor(() => expect(report).toHaveBeenCalledWith(failure));
  });
});
