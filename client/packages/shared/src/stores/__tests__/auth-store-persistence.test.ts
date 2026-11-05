import { userSchema, type User } from "@trenova/shared/types/user";
import { FakeBroadcastChannel } from "@trenova/shared/testing/fake-broadcast-channel";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

const STORAGE_KEY = "auth-storage";

function makeUser(overrides: Partial<User> = {}): User {
  return userSchema.parse({
    id: "usr_01KSJG9AG6XFQ8HDX5309293A9",
    version: 0,
    createdAt: 1779811265,
    updatedAt: 1779811264,
    businessUnitId: "bu_01KSJG9ADGK726S3YXAWMWQ2G3",
    currentOrganizationId: "org_01KSJG9ADP4CFEFS4P1CZ699NB",
    status: "Active",
    name: "Dana Dispatcher",
    username: "dana",
    emailAddress: "dana@example.com",
    phoneNumber: "+1 555 0100",
    profilePicUrl: "",
    thumbnailUrl: "",
    timezone: "America/Denver",
    timeFormat: "24-hour",
    locale: "es",
    isLocked: false,
    mustChangePassword: false,
    lastLoginAt: 1779811303,
    ...overrides,
  });
}

function stored(): string {
  return localStorage.getItem(STORAGE_KEY) ?? "";
}

async function loadFreshStore() {
  vi.resetModules();
  return import("@trenova/shared/stores/auth-store");
}

const COLD_LOAD_TIMEOUT_MS = 60_000;

describe("auth store persistence", () => {
  // Each test loads the store afresh. The first load is cold, seconds on a busy machine, so it
  // is paid here against its own limit rather than the first test's.
  beforeAll(async () => {
    await import("@trenova/shared/stores/auth-store");
  }, COLD_LOAD_TIMEOUT_MS);

  beforeEach(() => {
    localStorage.clear();
    FakeBroadcastChannel.open = [];
    vi.stubGlobal("BroadcastChannel", FakeBroadcastChannel);
  });

  afterEach(() => {
    localStorage.clear();
    vi.unstubAllGlobals();
  });

  it("writes who and where the person is, and nothing that says who they are", async () => {
    const { useAuthStore } = await loadFreshStore();

    useAuthStore.getState().setUser(makeUser());

    const saved = JSON.parse(stored()) as { state: Record<string, unknown>; version: number };
    expect(saved).toEqual({
      state: {
        restored: {
          id: "usr_01KSJG9AG6XFQ8HDX5309293A9",
          currentOrganizationId: "org_01KSJG9ADP4CFEFS4P1CZ699NB",
          businessUnitId: "bu_01KSJG9ADGK726S3YXAWMWQ2G3",
          locale: "es",
          timezone: "America/Denver",
          timeFormat: "24-hour",
        },
      },
      version: 1,
    });
    expect(stored()).not.toContain("Dana");
    expect(stored()).not.toContain("dana@example.com");
    expect(stored()).not.toContain("555");
  });

  it("drops a whole user that an earlier version stored", async () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ state: { user: makeUser() }, version: 0 }));

    const { useAuthStore } = await loadFreshStore();

    expect(useAuthStore.getState().user).toBeNull();
    expect(useAuthStore.getState().restored).toBeNull();
    expect(stored()).not.toContain("dana@example.com");
  });

  it("paints from the restored profile until the server answers, and forgets it after a sign-out", async () => {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({
        state: {
          restored: {
            id: "usr_restored",
            currentOrganizationId: "org_restored",
            businessUnitId: "bu_restored",
            locale: "es",
            timezone: "America/Denver",
            timeFormat: "24-hour",
          },
        },
        version: 1,
      }),
    );
    const { currentProfile, useAuthStore } = await loadFreshStore();

    expect(currentProfile(useAuthStore.getState())?.currentOrganizationId).toBe("org_restored");

    useAuthStore.getState().setUser(null);

    expect(currentProfile(useAuthStore.getState())).toBeNull();
    expect(JSON.parse(stored())).toEqual({ state: { restored: null }, version: 1 });
  });

  it("takes the server's user over the restored profile once it answers", async () => {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({
        state: { restored: { id: "usr_restored", currentOrganizationId: "org_restored" } },
        version: 1,
      }),
    );
    const { currentProfile, useAuthStore } = await loadFreshStore();

    useAuthStore.getState().setUser(makeUser());

    expect(currentProfile(useAuthStore.getState())?.id).toBe("usr_01KSJG9AG6XFQ8HDX5309293A9");
    expect(useAuthStore.getState().restored).toBeNull();
  });
});
