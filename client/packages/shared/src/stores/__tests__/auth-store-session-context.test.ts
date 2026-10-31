import { FakeBroadcastChannel } from "@trenova/shared/testing/fake-broadcast-channel";
import type { User } from "@trenova/shared/types/user";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const RESTORED = {
  id: "usr_restored",
  currentOrganizationId: "org_restored",
  businessUnitId: "bu_restored",
  locale: "en",
  timezone: "America/Chicago",
  timeFormat: "12-hour",
};

async function loadFreshStore() {
  vi.resetModules();
  const { useAuthStore } = await import("@trenova/shared/stores/auth-store");
  const { withSessionContextHeaders } = await import("@trenova/shared/lib/session-context");
  return {
    useAuthStore,
    assertedOrganization: () => withSessionContextHeaders().get("X-Trenova-Organization-Id"),
  };
}

describe("auth store session context", () => {
  beforeEach(() => {
    FakeBroadcastChannel.open = [];
    vi.stubGlobal("BroadcastChannel", FakeBroadcastChannel);
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response(JSON.stringify({}), { status: 200 })),
    );
    localStorage.setItem(
      "auth-storage",
      JSON.stringify({ state: { restored: RESTORED }, version: 1 }),
    );
  });

  afterEach(() => {
    localStorage.clear();
    vi.unstubAllGlobals();
  });

  it("never asserts the profile restored from storage, whatever else the store writes", async () => {
    const { useAuthStore, assertedOrganization } = await loadFreshStore();

    useAuthStore.setState({ isLoading: true });
    useAuthStore.setState({ isLoading: false, isAuthenticated: true });

    expect(useAuthStore.getState().restored?.currentOrganizationId).toBe("org_restored");
    expect(assertedOrganization()).toBeNull();
  });

  it("asserts the session of each user the server answers with, and none once signed out", async () => {
    const { useAuthStore, assertedOrganization } = await loadFreshStore();

    useAuthStore.getState().setUser({
      ...RESTORED,
      currentOrganizationId: "org_session",
      businessUnitId: "bu_session",
    } as User);
    expect(assertedOrganization()).toBe("org_session");

    useAuthStore.getState().setUser(null);
    expect(assertedOrganization()).toBeNull();
  });

  it("tells the other tabs when the person signs out, even when the server cannot be told", async () => {
    const { useAuthStore } = await loadFreshStore();
    const otherTab = new FakeBroadcastChannel("trenova-session-context");
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => {
        throw new TypeError("Failed to fetch");
      }),
    );

    await useAuthStore
      .getState()
      .logout()
      .catch(() => undefined);

    expect(otherTab.received).toEqual([{ kind: "signed-out" }]);
  });
});
