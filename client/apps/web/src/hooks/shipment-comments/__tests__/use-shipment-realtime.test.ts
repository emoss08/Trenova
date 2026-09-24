import { act, renderHook } from "@testing-library/react";
import {
  realtimeClient,
  type RealtimePresenceMember,
  type RealtimeTypingEvent,
} from "@trenova/shared/services/realtime";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { User } from "@trenova/shared/types/user";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useShipmentTyping } from "../use-shipment-typing";
import { useShipmentViewers } from "../use-shipment-viewers";

const SHIPMENT = "shp_01J00000000000000000000000";
const SCOPE = `shipment-comments:${SHIPMENT}`;
const PRESENCE_PATH = `/shipments/${SHIPMENT}/comments/presence/`;
const TYPING_PATH = `/shipments/${SHIPMENT}/comments/typing/`;

let typingListener: ((event: RealtimeTypingEvent) => void) | null;
let presenceListener: ((members: RealtimePresenceMember[]) => void) | null;
let leave: ReturnType<typeof vi.fn<() => void>>;
let joinScope: ReturnType<typeof vi.spyOn>;
let sendTyping: ReturnType<typeof vi.spyOn>;
let subscribePresence: ReturnType<typeof vi.spyOn>;

beforeEach(() => {
  useAuthStore.setState({
    user: { id: "usr_me", name: "Me" } as unknown as User,
    isAuthenticated: true,
  });
  typingListener = null;
  presenceListener = null;
  leave = vi.fn<() => void>();
  joinScope = vi.spyOn(realtimeClient, "joinScope").mockReturnValue(leave);
  sendTyping = vi.spyOn(realtimeClient, "sendTyping").mockImplementation(() => undefined);
  vi.spyOn(realtimeClient, "on").mockImplementation((event, listener) => {
    if (event === "typing") {
      typingListener = listener as (event: RealtimeTypingEvent) => void;
    }
    return () => {
      typingListener = null;
    };
  });
  subscribePresence = vi
    .spyOn(realtimeClient, "subscribePresence")
    .mockImplementation((_scope, listener) => {
      presenceListener = listener;
      return () => {
        presenceListener = null;
      };
    });
});

afterEach(() => {
  vi.restoreAllMocks();
  useAuthStore.setState({ user: null, isAuthenticated: false });
});

describe("useShipmentViewers", () => {
  it("joins the thread's scope through its presence endpoint and leaves on unmount", () => {
    const { unmount } = renderHook(() => useShipmentViewers(SHIPMENT));

    expect(joinScope).toHaveBeenCalledWith(SCOPE, PRESENCE_PATH);
    expect(subscribePresence).toHaveBeenCalledWith(SCOPE, expect.any(Function));

    unmount();
    expect(leave).toHaveBeenCalledTimes(1);
  });

  it("lists each other person once, however many tabs they have open, and never the reader", () => {
    const { result } = renderHook(() => useShipmentViewers(SHIPMENT));

    act(() => {
      presenceListener?.([
        { userId: "usr_me", connectionId: "rtc_1", name: "Me" },
        { userId: "usr_pat", connectionId: "rtc_2", name: "" },
        { userId: "usr_pat", connectionId: "rtc_3", name: "Pat" },
        { userId: "usr_sam", connectionId: "rtc_4", name: "" },
      ]);
    });

    expect(result.current.viewers).toEqual([
      { userId: "usr_pat", name: "Pat" },
      { userId: "usr_sam", name: "Someone" },
    ]);
  });
});

describe("useShipmentTyping", () => {
  it("shows other people typing in this thread only", () => {
    const { result } = renderHook(() => useShipmentTyping(SHIPMENT));

    act(() => {
      typingListener?.({ scope: SCOPE, userId: "usr_pat", name: "Pat" });
      typingListener?.({ scope: SCOPE, userId: "usr_me", name: "Me" });
      typingListener?.({ scope: "shipment-comments:shp_other", userId: "usr_sam", name: "Sam" });
    });
    expect(result.current.typingUsers).toEqual([{ userId: "usr_pat", name: "Pat" }]);

    act(() => {
      typingListener?.({ scope: SCOPE, userId: "usr_pat", stop: true });
    });
    expect(result.current.typingUsers).toEqual([]);
  });

  it("throttles typing signals and always sends a stop", () => {
    const { result } = renderHook(() => useShipmentTyping(SHIPMENT));

    act(() => {
      result.current.publishTyping();
      result.current.publishTyping();
    });
    expect(sendTyping).toHaveBeenCalledTimes(1);
    expect(sendTyping).toHaveBeenLastCalledWith(TYPING_PATH, false);

    act(() => {
      result.current.publishStopTyping();
    });
    expect(sendTyping).toHaveBeenLastCalledWith(TYPING_PATH, true);

    act(() => {
      result.current.publishTyping();
    });
    expect(sendTyping).toHaveBeenCalledTimes(3);
  });

  it("says it stopped when the thread closes mid-sentence", () => {
    const { result, unmount } = renderHook(() => useShipmentTyping(SHIPMENT));

    act(() => {
      result.current.publishTyping();
    });
    unmount();

    expect(sendTyping).toHaveBeenLastCalledWith(TYPING_PATH, true);
    expect(leave).toHaveBeenCalledTimes(1);
  });
});
