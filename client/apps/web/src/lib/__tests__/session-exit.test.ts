import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  calls: [] as string[],
  isAuthenticated: true,
}));

vi.mock("@/lib/session-cache", () => ({
  clearSessionCache: async () => {
    mocks.calls.push("clear cache");
  },
}));

vi.mock("@/lib/reload-guards", () => ({
  leaveRegardlessOfWork: (destination: string) => {
    mocks.calls.push(`leave past unsaved work for ${destination}`);
  },
}));

vi.mock("@trenova/shared/stores/auth-store", () => ({
  useAuthStore: {
    getState: () => ({
      isAuthenticated: mocks.isAuthenticated,
      clearAuth: () => {
        mocks.calls.push("clear auth");
      },
    }),
  },
}));

vi.mock("@trenova/shared/lib/session-context", () => ({
  announceSignedOut: () => {
    mocks.calls.push("tell the other tabs");
  },
}));

const { leaveEndedSession, leaveExpiredSession } = await import("@/lib/session-exit");

describe("leaveEndedSession", () => {
  beforeEach(() => {
    mocks.calls.length = 0;
    mocks.isAuthenticated = true;
    Object.defineProperty(window, "location", {
      configurable: true,
      value: {
        ...window.location,
        assign: (url: string) => mocks.calls.push(`assign ${url}`),
      },
    });
  });

  it("clears the session, then leaves past unsaved work when the session was ended on purpose", async () => {
    await leaveEndedSession({ destination: "/login?signed_out=elsewhere", pastUnsavedWork: true });

    expect(mocks.calls).toEqual([
      "clear auth",
      "clear cache",
      "leave past unsaved work for /login?signed_out=elsewhere",
    ]);
  });

  it("leaves through the browser's prompt when the server ended the session under someone", async () => {
    await leaveEndedSession({ destination: "/login?signed_out=expired", pastUnsavedWork: false });

    expect(mocks.calls).toEqual(["clear auth", "clear cache", "assign /login?signed_out=expired"]);
  });

  it("stays on the page when there is nowhere to go", async () => {
    await leaveEndedSession({ destination: null, pastUnsavedWork: false });

    expect(mocks.calls).toEqual(["clear auth", "clear cache"]);
  });

  it("tells the other tabs a session the server ended is over, then leaves through the prompt", async () => {
    leaveExpiredSession();
    await vi.waitFor(() => expect(mocks.calls).toHaveLength(4));

    expect(mocks.calls).toEqual([
      "tell the other tabs",
      "clear auth",
      "clear cache",
      "assign /login?signed_out=expired",
    ]);
  });

  it("does nothing when the tab is already signed out", () => {
    mocks.isAuthenticated = false;

    leaveExpiredSession();

    expect(mocks.calls).toEqual([]);
  });
});
