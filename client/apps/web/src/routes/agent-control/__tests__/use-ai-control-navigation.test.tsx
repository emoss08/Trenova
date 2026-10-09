import { act, renderHook, waitFor } from "@testing-library/react";
import { NuqsTestingAdapter, type UrlUpdateEvent } from "nuqs/adapters/testing";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { AGENT_OPEN_PARAM } from "../ai-control-tabs";
import { useAIControlNavigation } from "../use-ai-control-navigation";

function setup(searchParams = "") {
  const onUrlUpdate = vi.fn<(event: UrlUpdateEvent) => void>();
  const wrapper = ({ children }: { children: ReactNode }) => (
    <NuqsTestingAdapter searchParams={searchParams} onUrlUpdate={onUrlUpdate}>
      {children}
    </NuqsTestingAdapter>
  );
  const { result } = renderHook(() => useAIControlNavigation(), { wrapper });
  const lastParams = async () => {
    await waitFor(() => expect(onUrlUpdate).toHaveBeenCalled());
    return onUrlUpdate.mock.lastCall![0].searchParams;
  };
  return { go: result.current, lastParams };
}

describe("useAIControlNavigation", () => {
  it("opens one agent on the Agents tab from the address", async () => {
    const { go, lastParams } = setup("?tab=overview");
    await act(async () => go({ tab: "agents", agent: "agdef_1" }));

    const params = await lastParams();
    expect(params.get("tab")).toBe("agents");
    expect(params.get(AGENT_OPEN_PARAM)).toBe("agdef_1");
  });

  it("closes an open agent when it moves anywhere else", async () => {
    const { go, lastParams } = setup(`?tab=agents&${AGENT_OPEN_PARAM}=agdef_1`);
    await act(async () => go({ tab: "activity", view: "runs" }));

    const params = await lastParams();
    expect(params.get("tab")).toBe("activity");
    expect(params.has(AGENT_OPEN_PARAM)).toBe(false);
  });

  it("names no agent when the Agents tab is opened without one", async () => {
    const { go, lastParams } = setup(`?tab=agents&${AGENT_OPEN_PARAM}=agdef_1`);
    await act(async () => go({ tab: "agents", agentFilter: "shadow" }));

    const params = await lastParams();
    expect(params.get("agents")).toBe("shadow");
    expect(params.has(AGENT_OPEN_PARAM)).toBe(false);
  });
});
