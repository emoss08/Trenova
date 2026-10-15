import { useAssistantStore } from "@/stores/assistant-store";
import { act, render, renderHook } from "@testing-library/react";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { userSchema, type User } from "@trenova/shared/types/user";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AssistantWidget, assistantView, useAssistantSideInset } from "../assistant-widget";

const permission = vi.hoisted(() => ({ allowed: true }));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: permission.allowed, isLoading: false }),
}));

function userWith(assistantHidden: boolean): User {
  return userSchema.parse({
    id: "usr_1",
    status: "Active",
    name: "Dana Reyes",
    username: "dreyes",
    emailAddress: "dana@example.com",
    timezone: "America/Chicago",
    assistantHidden,
  });
}

function signIn(user: User | null) {
  act(() => {
    useAuthStore.setState({ user });
  });
}

afterEach(() => {
  permission.allowed = true;
  signIn(null);
  act(() => {
    useAssistantStore.setState({ open: false, layout: "compact", dock: "bottom-right" });
  });
});

/*
In the corner and docked to the side the panel shows one thing at a time:
its home, the conversations, or the open conversation. Full screen, the
conversations are always down the side, so there is no list view to show.
*/
describe("assistantView", () => {
  it("shows the open conversation, or the home when there is none", () => {
    expect(assistantView("compact", false, true)).toBe("thread");
    expect(assistantView("side", false, false)).toBe("home");
  });

  it("shows the conversations over either while the list is asked for", () => {
    expect(assistantView("compact", true, true)).toBe("history");
    expect(assistantView("side", true, false)).toBe("history");
  });

  it("never shows the list full screen, where the sidebar holds it", () => {
    expect(assistantView("full", true, true)).toBe("thread");
    expect(assistantView("full", true, false)).toBe("home");
  });
});

/*
Docked to the side, the panel takes a column of the screen and the page moves
over for it, on the side the panel is docked to. Floating or full screen, or
closed, the page keeps its width.
*/
describe("useAssistantSideInset", () => {
  it("makes room on the docked side while the panel is open there", () => {
    act(() => {
      useAssistantStore.setState({ open: true, layout: "side", dock: "top-right" });
    });
    expect(renderHook(() => useAssistantSideInset()).result.current).toBe("right");

    act(() => {
      useAssistantStore.setState({ dock: "bottom-left" });
    });
    expect(renderHook(() => useAssistantSideInset()).result.current).toBe("left");
  });

  it("makes no room while the panel floats, fills the screen or is closed", () => {
    act(() => {
      useAssistantStore.setState({ open: true, layout: "compact" });
    });
    expect(renderHook(() => useAssistantSideInset()).result.current).toBeNull();

    act(() => {
      useAssistantStore.setState({ layout: "full" });
    });
    expect(renderHook(() => useAssistantSideInset()).result.current).toBeNull();

    act(() => {
      useAssistantStore.setState({ open: false, layout: "side" });
    });
    expect(renderHook(() => useAssistantSideInset()).result.current).toBeNull();
  });

  it("makes no room for someone who cannot use the assistant", () => {
    permission.allowed = false;
    act(() => {
      useAssistantStore.setState({ open: true, layout: "side" });
    });

    expect(renderHook(() => useAssistantSideInset()).result.current).toBeNull();
  });
});

/*
Someone who turns the assistant off in Settings gets none of it on any page:
the shell mounts nothing, and the page keeps its full width even though the
store still remembers the panel docked open down the side.
*/
describe("an assistant turned off in Settings", () => {
  it("mounts nothing, even with the panel left open", () => {
    signIn(userWith(true));
    act(() => {
      useAssistantStore.setState({ open: true, layout: "side", dock: "top-right" });
    });

    const { container } = render(<AssistantWidget />);

    expect(container).toBeEmptyDOMElement();
  });

  it("makes no room for a panel that is not there", () => {
    signIn(userWith(true));
    act(() => {
      useAssistantStore.setState({ open: true, layout: "side", dock: "top-right" });
    });

    expect(renderHook(() => useAssistantSideInset()).result.current).toBeNull();
  });

  it("makes room again once it is turned back on", () => {
    signIn(userWith(false));
    act(() => {
      useAssistantStore.setState({ open: true, layout: "side", dock: "top-right" });
    });

    expect(renderHook(() => useAssistantSideInset()).result.current).toBe("right");
  });
});
