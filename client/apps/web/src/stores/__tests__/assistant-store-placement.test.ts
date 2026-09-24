import { describe, expect, it } from "vitest";
import { DEFAULT_ASSISTANT_DOCK } from "@/lib/assistant-dock";
import { mergePersisted, useAssistantStore } from "../assistant-store";

const current = () => useAssistantStore.getInitialState();

/**
 * Where the assistant sits is read back from the browser on every load. A
 * value this build does not understand must not put it somewhere it cannot
 * be seen or reached.
 */
describe("restoring the assistant's placement", () => {
  it("keeps a saved corner, size and hidden launcher", () => {
    const restored = mergePersisted(
      { dock: "top-left", launcherHidden: true, panelSize: { width: 420, height: 480 } },
      current(),
    );

    expect(restored.dock).toBe("top-left");
    expect(restored.launcherHidden).toBe(true);
    expect(restored.panelSize).toEqual({ width: 420, height: 480 });
  });

  it("falls back to the default corner for one it does not know", () => {
    expect(mergePersisted({ dock: "middle" }, current()).dock).toBe(DEFAULT_ASSISTANT_DOCK);
  });

  it("drops a size that is not two numbers", () => {
    expect(mergePersisted({ panelSize: { width: "wide" } }, current()).panelSize).toBeNull();
  });

  // Only an explicit true hides the launcher; anything else leaves it visible,
  // because a hidden launcher that should not be is harder to notice.
  it("shows the launcher unless it was explicitly hidden", () => {
    expect(mergePersisted({ launcherHidden: "yes" }, current()).launcherHidden).toBe(false);
    expect(mergePersisted({}, current()).launcherHidden).toBe(false);
  });

  it("starts from the defaults when nothing was saved", () => {
    const restored = mergePersisted(undefined, current());

    expect(restored.dock).toBe(DEFAULT_ASSISTANT_DOCK);
    expect(restored.panelSize).toBeNull();
  });

  it("keeps the rest of what was saved", () => {
    expect(mergePersisted({ lastAgentId: "agent_1" }, current()).lastAgentId).toBe("agent_1");
  });
});
