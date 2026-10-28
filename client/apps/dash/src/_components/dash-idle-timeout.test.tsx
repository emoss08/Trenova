import type { SignOutNotice } from "@trenova/shared/lib/sign-out-notice";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
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

function renderTimeout(queryClient = new QueryClient()) {
  render(
    <QueryClientProvider client={queryClient}>
      <DashIdleTimeout />
    </QueryClientProvider>,
  );
  return queryClient;
}

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
    renderTimeout();

    await endSession("idle");

    expect(mocks.logout).toHaveBeenCalledTimes(1);
    expect(mocks.navigate).toHaveBeenCalledWith("/dash/login?signed_out=idle", { replace: true });
  });

  it("still reaches the sign-in page when the server already ended the session", async () => {
    mocks.logout.mockRejectedValue(new Error("Session has expired"));
    renderTimeout();

    await endSession("expired");

    expect(mocks.navigate).toHaveBeenCalledWith("/dash/login?signed_out=expired", {
      replace: true,
    });
  });

  it("shows no notice when the driver chose to sign out", async () => {
    mocks.logout.mockResolvedValue(undefined);
    renderTimeout();

    await endSession(undefined);

    expect(mocks.navigate).toHaveBeenCalledWith("/dash/login", { replace: true });
  });

  it("forgets the driver's pay and loads, so the next driver on the phone never sees them", async () => {
    mocks.logout.mockResolvedValue(undefined);
    const queryClient = new QueryClient();
    queryClient.setQueryData(["dash-total-comp"], { totalMinor: 125000 });
    queryClient.setQueryData(["dash-loads"], [{ id: "shp_1" }]);
    renderTimeout(queryClient);

    await endSession("idle");

    expect(queryClient.getQueryCache().getAll()).toEqual([]);
  });
});
