import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { LeaveStandingStrip } from "../leave-standing-strip";

const mocks = vi.hoisted(() => ({ fetchWorkerLeaveFile: vi.fn() }));

vi.mock("@/lib/graphql/worker-leave", () => ({
  fetchWorkerLeaveFile: mocks.fetchWorkerLeaveFile,
  WORKER_LEAVE_KEY: "worker-leave",
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

function renderStrip() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <LeaveStandingStrip workerId="wrk_1" />
    </QueryClientProvider>,
  );
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("LeaveStandingStrip", () => {
  it("says the standing could not be loaded rather than disappearing", async () => {
    mocks.fetchWorkerLeaveFile.mockRejectedValue(new Error("boom"));
    renderStrip();

    expect(await screen.findByText("The leave standing could not be loaded")).toBeInTheDocument();
  });
});
