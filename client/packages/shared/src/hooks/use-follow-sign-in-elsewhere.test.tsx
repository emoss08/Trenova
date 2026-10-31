import { FakeBroadcastChannel } from "@trenova/shared/testing/fake-broadcast-channel";
import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const CHANNEL_NAME = "trenova-session-context";

describe("useFollowSignInElsewhere", () => {
  beforeEach(() => {
    vi.resetModules();
    FakeBroadcastChannel.open = [];
    vi.stubGlobal("BroadcastChannel", FakeBroadcastChannel);
  });

  it("follows a sign-in finished in another tab with the latest callback, until unmounted", async () => {
    const { useFollowSignInElsewhere } =
      await import("@trenova/shared/hooks/use-follow-sign-in-elsewhere");
    const first = vi.fn();
    const latest = vi.fn();
    const { rerender, unmount } = renderHook(({ follow }) => useFollowSignInElsewhere(follow), {
      initialProps: { follow: first },
    });
    rerender({ follow: latest });
    const otherTab = new FakeBroadcastChannel(CHANNEL_NAME);

    otherTab.postMessage({ kind: "signed-in" });
    unmount();
    otherTab.postMessage({ kind: "signed-in" });

    expect(first).not.toHaveBeenCalled();
    expect(latest).toHaveBeenCalledTimes(1);
  });
});
