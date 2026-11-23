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

  it("follows the days away with the classification until somebody picks one", async () => {
    mocks.recordWorkerInjury.mockResolvedValue({
      id: "inj_2",
      caseNumber: 2,
      caseYear: 2026,
      recordable: true,
    });
    renderDialog();
    const user = userEvent.setup();

    await user.type(
      screen.getByPlaceholderText("How the injury occurred, as it would read on the 301 form"),
      "Fell from the trailer step",
    );
    const daysAway = screen.getByRole("textbox", { name: "Days away from work" });
    await user.clear(daysAway);
    await user.type(daysAway, "3");

    expect(await screen.findByText(/This case goes on the OSHA 300 log/)).toBeTruthy();
    await user.click(screen.getByRole("button", { name: "Record case" }));
    await waitFor(() =>
      expect(mocks.recordWorkerInjury).toHaveBeenCalledWith(
        expect.objectContaining({ daysAway: 3, classification: "DaysAway" }),
      ),
    );
  });

  it("keeps what is typed when the case it was opened on is read again", async () => {
    const client = new QueryClient();
    const opened = {
      id: "inj_9",
      workerId: "wrk_1",
      caseNumber: 9,
      caseYear: 2026,
      classification: "OtherRecordable",
      illnessType: "Injury",
      treatment: "MedicalTreatment",
      status: "Open",
      recordable: true,
      occurredAt: 1_790_000_000,
      reportedAt: 1_790_000_000,
      returnedToWorkAt: null,
      location: null,
      description: "Twisted an ankle at the dock",
      bodyPart: "Ankle",
      harmfulAgent: null,
      daysAway: 0,
      daysRestricted: 0,
      privacyCase: false,
      logName: "Maria Alvarez",
      claimStatus: "NotFiled",
      claimNumber: null,
      claimCarrier: null,
      claimFiledAt: null,
      claimClosedAt: null,
      safetyEventId: null,
      documentId: null,
      notes: null,
      version: 1,
    } as const;
    const { rerender } = render(
      <QueryClientProvider client={client}>
        <InjuryDialog open onOpenChange={vi.fn()} workerId="wrk_1" injury={opened} />
      </QueryClientProvider>,
    );
    const user = userEvent.setup();
    const description = screen.getByPlaceholderText(
      "How the injury occurred, as it would read on the 301 form",
    );
    await user.type(description, ", then kept working");

    rerender(
      <QueryClientProvider client={client}>
        <InjuryDialog
          open
          onOpenChange={vi.fn()}
          workerId="wrk_1"
          injury={{ ...opened, version: 2, description: "Changed elsewhere" }}
        />
      </QueryClientProvider>,
    );

    expect((description as HTMLTextAreaElement).value).toBe(
      "Twisted an ankle at the dock, then kept working",
    );
  });
});
