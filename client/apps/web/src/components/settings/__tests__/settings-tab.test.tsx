import { act, renderHook, waitFor } from "@testing-library/react";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { tabHoldingError, useSettingsTab, type SettingsTabSpec } from "../settings-tab";

const TABS: SettingsTabSpec[] = [
  { value: "assignment", label: "Assignment", fields: ["enabled", "horizon"] },
  { value: "costs", label: "Costs", fields: ["economics"] },
  { value: "rules", label: "Rules", fields: ["level"] },
];

function address(searchParams: string, onUrlUpdate = vi.fn()) {
  return {
    onUrlUpdate,
    wrapper: ({ children }: { children: ReactNode }) => (
      <NuqsTestingAdapter hasMemory searchParams={searchParams} onUrlUpdate={onUrlUpdate}>
        {children}
      </NuqsTestingAdapter>
    ),
  };
}

describe("the open tab of a settings page", () => {
  it("is the first tab when the address names none", () => {
    const { result } = renderHook(() => useSettingsTab(TABS), address(""));

    expect(result.current[0]).toBe("assignment");
  });

  it("is the tab the address names", () => {
    const { result } = renderHook(() => useSettingsTab(TABS), address("?tab=costs"));

    expect(result.current[0]).toBe("costs");
  });

  it("is the first tab when the address names one the page does not offer", () => {
    const { result } = renderHook(() => useSettingsTab(TABS), address("?tab=tenders"));

    expect(result.current[0]).toBe("assignment");
  });

  it("is written to the address when a tab is picked, so the tab can be linked to", async () => {
    const { onUrlUpdate, wrapper } = address("");
    const { result } = renderHook(() => useSettingsTab(TABS), { wrapper });

    act(() => result.current[1]("rules"));

    expect(result.current[0]).toBe("rules");
    await waitFor(() => expect(onUrlUpdate).toHaveBeenCalled());
    expect(onUrlUpdate.mock.lastCall?.[0].searchParams.get("tab")).toBe("rules");
  });

  it("leaves the address clean when the first tab is picked", async () => {
    const { onUrlUpdate, wrapper } = address("?tab=costs");
    const { result } = renderHook(() => useSettingsTab(TABS), { wrapper });

    act(() => result.current[1]("assignment"));

    expect(result.current[0]).toBe("assignment");
    await waitFor(() => expect(onUrlUpdate).toHaveBeenCalled());
    expect(onUrlUpdate.mock.lastCall?.[0].searchParams.has("tab")).toBe(false);
  });
});

describe("the tab shown after a save is refused", () => {
  const failing =
    (...fields: string[]) =>
    (field: string) =>
      fields.includes(field);

  it("is the first tab that holds a problem when the open one holds none", () => {
    expect(tabHoldingError(TABS, "assignment", failing("level", "economics"))).toBe("costs");
  });

  it("stays the open tab while it holds a problem, even when an earlier tab does too", () => {
    expect(tabHoldingError(TABS, "rules", failing("horizon", "level"))).toBe("rules");
  });

  it("is none when the problem belongs to no tab", () => {
    expect(tabHoldingError(TABS, "assignment", failing("version"))).toBeNull();
  });
});
