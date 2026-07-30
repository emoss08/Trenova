import { getUserDatePreferences, setUserDatePreferences } from "@trenova/shared/lib/date";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { userSchema, type User } from "@trenova/shared/types/user";
import { ApiRequestError, clearCsrfToken, setCsrfToken } from "@trenova/shared/lib/api";
import { userService } from "@trenova/shared/services/user";
import { authService } from "@trenova/shared/services/auth";
import { realtimeService } from "@trenova/shared/services/realtime";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// Parsed through userSchema rather than cast, so adding a required field to User fails
// here instead of letting a fixture drift out from under the store.
function makeUser(overrides: Partial<User> = {}): User {
  return userSchema.parse({
    id: "usr_01KSJG9AG6XFQ8HDX5309293A9",
    version: 0,
    createdAt: 1779811265,
    updatedAt: 1779811264,
    businessUnitId: "bu_01KSJG9ADGK726S3YXAWMWQ2G3",
    currentOrganizationId: "org_01KSJG9ADP4CFEFS4P1CZ699NB",
    status: "Active",
    name: "System Administrator",
    username: "admin",
    emailAddress: "admin@trenova.app",
    profilePicUrl: "",
    thumbnailUrl: "",
    timezone: "America/Los_Angeles",
    timeFormat: "12-hour",
    isLocked: false,
    mustChangePassword: false,
    lastLoginAt: 1779811303,
    ...overrides,
  });
}

function authError(status: number): ApiRequestError {
  return new ApiRequestError(status, {
    type: `https://api.trenova.app/problems/${
      status === 401 ? "authentication-error" : "internal-error"
    }`,
    title: status === 401 ? "Authentication Failed" : "Internal Error",
    status,
  });
}

describe("auth store date preferences", () => {
  afterEach(() => {
    useAuthStore.getState().setUser(null);
    setUserDatePreferences({});
  });

  it("mirrors the signed-in user's zone and clock into the date module", () => {
    useAuthStore
      .getState()
      .setUser(makeUser({ timezone: "America/Denver", timeFormat: "24-hour" }));

    expect(getUserDatePreferences()).toEqual({
      timezone: "America/Denver",
      timeFormat: "24-hour",
    });
  });

  it("falls back to the browser zone once the user is cleared", () => {
    useAuthStore
      .getState()
      .setUser(makeUser({ timezone: "America/Denver", timeFormat: "24-hour" }));
    useAuthStore.getState().setUser(null);

    expect(getUserDatePreferences()).toEqual({ timezone: "auto", timeFormat: "12-hour" });
  });
});

describe("auth store checkAuth", () => {
  let clearPermissions: ReturnType<typeof vi.spyOn>;
  let checkForUpdates: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    clearPermissions = vi
      .spyOn(usePermissionStore.getState(), "clearPermissions")
      .mockImplementation(() => {});
    checkForUpdates = vi
      .spyOn(usePermissionStore.getState(), "checkForUpdates")
      .mockResolvedValue(undefined as never);
    setCsrfToken("session-token");
    useAuthStore.getState().setUser(makeUser());
  });

  afterEach(() => {
    vi.restoreAllMocks();
    clearCsrfToken();
    useAuthStore.getState().setUser(null);
  });

  it("marks the session authenticated and refreshes permissions on success", async () => {
    const user = makeUser({ name: "Fresh User" });
    vi.spyOn(userService, "currentUser").mockResolvedValue(user);

    await expect(useAuthStore.getState().checkAuth()).resolves.toBe(true);

    expect(useAuthStore.getState().isAuthenticated).toBe(true);
    expect(useAuthStore.getState().user?.name).toBe("Fresh User");
    expect(checkForUpdates).toHaveBeenCalledWith(user.currentOrganizationId);
  });

  // This branch is what the GraphQL transport's session-expiry handler routes into. It had
  // never been executed by a test before.
  it("clears the session, the CSRF token and permissions on a 401", async () => {
    vi.spyOn(userService, "currentUser").mockRejectedValue(authError(401));

    await expect(useAuthStore.getState().checkAuth()).resolves.toBe(false);

    expect(useAuthStore.getState().isAuthenticated).toBe(false);
    expect(useAuthStore.getState().user).toBeNull();
    expect(clearPermissions).toHaveBeenCalledTimes(1);
  });

  // A dropped connection is not an expiry. Tearing the session down on a blip would sign
  // the user out of a working session because one request failed.
  it("leaves the session intact when the request fails without a 401", async () => {
    vi.spyOn(userService, "currentUser").mockRejectedValue(new TypeError("Failed to fetch"));

    await expect(useAuthStore.getState().checkAuth()).resolves.toBe(false);

    expect(useAuthStore.getState().user).not.toBeNull();
    expect(useAuthStore.getState().isAuthenticated).toBe(true);
    expect(clearPermissions).not.toHaveBeenCalled();
  });

  it("leaves the session intact on a 500", async () => {
    vi.spyOn(userService, "currentUser").mockRejectedValue(authError(500));

    await expect(useAuthStore.getState().checkAuth()).resolves.toBe(false);

    expect(useAuthStore.getState().user).not.toBeNull();
    expect(clearPermissions).not.toHaveBeenCalled();
  });
});

describe("auth store teardown", () => {
  let clearPermissions: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    clearPermissions = vi
      .spyOn(usePermissionStore.getState(), "clearPermissions")
      .mockImplementation(() => {});
    setCsrfToken("session-token");
    useAuthStore.getState().setUser(makeUser());
  });

  afterEach(() => {
    vi.restoreAllMocks();
    clearCsrfToken();
    useAuthStore.getState().setUser(null);
  });

  it("clearAuth drops the user, the loading flag and permissions", () => {
    useAuthStore.getState().clearAuth();

    const state = useAuthStore.getState();
    expect(state.user).toBeNull();
    expect(state.isAuthenticated).toBe(false);
    expect(state.isLoading).toBe(false);
    expect(clearPermissions).toHaveBeenCalledTimes(1);
  });

  it("logout closes the realtime connection and clears the session", async () => {
    const logoutCall = vi.spyOn(authService, "logout").mockResolvedValue(undefined);
    const safeClose = vi.spyOn(realtimeService, "safeClose").mockImplementation(() => {});

    await useAuthStore.getState().logout();

    expect(logoutCall).toHaveBeenCalledTimes(1);
    expect(safeClose).toHaveBeenCalledTimes(1);
    expect(useAuthStore.getState().user).toBeNull();
    expect(useAuthStore.getState().isAuthenticated).toBe(false);
    expect(clearPermissions).toHaveBeenCalledTimes(1);
  });

  // The teardown is in a finally block, so a failing request must not leave the client
  // believing it is still signed in.
  it("logout clears the session even when the request fails", async () => {
    vi.spyOn(authService, "logout").mockRejectedValue(authError(500));
    vi.spyOn(realtimeService, "safeClose").mockImplementation(() => {});

    await expect(useAuthStore.getState().logout()).rejects.toBeInstanceOf(ApiRequestError);

    expect(useAuthStore.getState().user).toBeNull();
    expect(useAuthStore.getState().isAuthenticated).toBe(false);
  });
});
