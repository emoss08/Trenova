import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { InjuryDialog } from "../injury-dialog";

const mocks = vi.hoisted(() => ({ recordWorkerInjury: vi.fn(), updateWorkerInjury: vi.fn() }));

vi.mock("@/lib/graphql/worker-injury", () => ({
  ...mocks,
  WORKER_INJURIES_KEY: "worker-injuries",
  OSHA_LOG_KEY: "osha-log",
}));

function renderDialog(onOpenChange = vi.fn()) {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <InjuryDialog open onOpenChange={onOpenChange} workerId="wrk_1" injury={null} />
    </QueryClientProvider>,
  );
  return onOpenChange;
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("InjuryDialog", () => {
  it("records a new case for the worker from the shared form and closes", async () => {
    mocks.recordWorkerInjury.mockResolvedValue({
      id: "inj_1",
      caseNumber: 1,
      caseYear: 2026,
      recordable: false,
    });
    const onOpenChange = renderDialog();
    const user = userEvent.setup();

    expect(screen.getByText(/This case is kept on file but is not on the 300 log/)).toBeTruthy();
    await user.type(
      screen.getByPlaceholderText("How the injury occurred, as it would read on the 301 form"),
      "Strained a shoulder pulling a tarp",
    );
    await user.click(screen.getByRole("button", { name: "Record case" }));

    await waitFor(() =>
      expect(mocks.recordWorkerInjury).toHaveBeenCalledWith(
        expect.objectContaining({
          workerId: "wrk_1",
          description: "Strained a shoulder pulling a tarp",
          classification: "NotRecordable",
          illnessType: "Injury",
          claimStatus: "NotFiled",
        }),
      ),
    );
    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
  });
});
