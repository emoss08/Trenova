import type { SignOutNotice } from "@trenova/shared/lib/sign-out-notice";
import { act, cleanup, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DashIdleTimeout } from "./dash-idle-timeout";

const mocks = vi.hoisted(() => ({
  logout: vi.fn(),
  navigate: vi.fn(),
  endSession: null as ((notice: SignOutNotice | undefined) => void) | null,
}));

vi.mock("react-router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-router")>()),
  useNavigate: () => mocks.navigate,
}));

vi.mock("@trenova/shared/stores/auth-store", () => ({
  useAuthStore: (selector: (state: { logout: typeof mocks.logout }) => unknown) =>
    selector({ logout: mocks.logout }),
}));

// The guard's own timing is tested with the guard; here only what Dash does once a
// session ends is in question.
vi.mock("@trenova/shared/components/session/idle-session-guard", () => ({
  IdleSessionGuard: ({
    onSessionEnd,
  }: {
    onSessionEnd: (notice: SignOutNotice | undefined) => void;
  }) => {
    mocks.endSession = onSessionEnd;
    return null;
  },
}));

async function endSession(notice: SignOutNotice | undefined) {
  await act(async () => {
    mocks.endSession!(notice);
    await Promise.resolve();
    await Promise.resolve();
  });
}

describe("DashIdleTimeout", () => {
  beforeEach(() => {
    mocks.logout.mockReset();
    mocks.navigate.mockReset();
    mocks.endSession = null;
  });

  afterEach(() => {
    cleanup();
  });

  it("signs an idle driver out and tells the Dash sign-in why", async () => {
    mocks.logout.mockResolvedValue(undefined);
    render(<DashIdleTimeout />);

    await endSession("idle");

    expect(mocks.logout).toHaveBeenCalledTimes(1);
    expect(mocks.navigate).toHaveBeenCalledWith("/dash/login?signed_out=idle", { replace: true });
  });

  it("still reaches the sign-in page when the server already ended the session", async () => {
    mocks.logout.mockRejectedValue(new Error("Session has expired"));
    render(<DashIdleTimeout />);

    await endSession("expired");

    expect(mocks.navigate).toHaveBeenCalledWith("/dash/login?signed_out=expired", {
      replace: true,
    });
  });

  it("shows no notice when the driver chose to sign out", async () => {
    mocks.logout.mockResolvedValue(undefined);
    render(<DashIdleTimeout />);

    await endSession(undefined);

    expect(mocks.navigate).toHaveBeenCalledWith("/dash/login", { replace: true });
  });
});
