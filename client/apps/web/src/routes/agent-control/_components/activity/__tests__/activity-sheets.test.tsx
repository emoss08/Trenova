import type { AgentExceptionRow, AgentProposalRow } from "@/lib/graphql/agent-activity-tables";
import { stubLayout } from "@/test/layout";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ExceptionActionsContext, ExceptionSheet } from "../exception-sheet";
import { ProposalActionsContext, ProposalSheet, type ProposalActions } from "../proposal-sheet";

const fetchAgentRunDetail = vi.fn();

vi.mock("@/lib/graphql/agent-activity-tables", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-activity-tables")>()),
  fetchAgentRunDetail: (...args: unknown[]) => fetchAgentRunDetail(...args),
}));

let restoreLayout = () => {};

beforeEach(() => {
  restoreLayout = stubLayout();
  fetchAgentRunDetail.mockReset();
});

afterEach(() => {
  restoreLayout();
  cleanup();
});

function wrap(children: ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <QueryClientProvider client={client}>
      <NuqsTestingAdapter>
        <MemoryRouter>{children}</MemoryRouter>
      </NuqsTestingAdapter>
    </QueryClientProvider>
  );
}

function proposal(overrides: Partial<AgentProposalRow> = {}): AgentProposalRow {
  return {
    id: "aprop_1",
    runId: "arun_1",
    toolName: "release_billing_hold",
    toolParams: {},
    confidence: "0.82",
    rationale: "Release the hold on invoice 12",
    autonomyTier: "ActWithApproval",
    status: "Pending",
    parameterFields: [],
    tainted: false,
    createdAt: 1_800_000_000,
    updatedAt: 1_800_000_000,
    ...overrides,
  } as unknown as AgentProposalRow;
}

function actions(): ProposalActions {
  return { canDecide: true, approve: vi.fn(), modify: vi.fn(), reject: vi.fn() };
}

describe("ProposalSheet", () => {
  it("decides a waiting proposal through the table's own actions", async () => {
    const decide = actions();
    const waiting = proposal();
    render(
      wrap(
        <ProposalActionsContext.Provider value={decide}>
          <ProposalSheet proposal={waiting} onClose={() => {}} />
        </ProposalActionsContext.Provider>,
      ),
    );

    const sheet = await screen.findByRole("complementary", {
      name: "Release the hold on invoice 12",
    });
    expect(within(sheet).getByText("82%")).toBeTruthy();
    expect(within(sheet).queryByRole("button", { name: "Approve with changes" })).toBeNull();
    await userEvent.click(within(sheet).getByRole("button", { name: "Approve" }));
    expect(decide.approve).toHaveBeenCalledWith(waiting);
    await userEvent.click(within(sheet).getByRole("button", { name: "Reject" }));
    expect(decide.reject).toHaveBeenCalledWith(waiting);
  });

  it("warns when the run read outside text, and offers no decision once decided", async () => {
    render(
      wrap(
        <ProposalActionsContext.Provider value={actions()}>
          <ProposalSheet
            proposal={proposal({ tainted: true, status: "Executed" })}
            onClose={() => {}}
          />
        </ProposalActionsContext.Provider>,
      ),
    );

    expect(
      await screen.findByText(/read an email or document written outside the organization/),
    ).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Approve" })).toBeNull();
    expect(screen.getByText(/^Decided /)).toBeTruthy();
  });

  it("opens the run that proposed it", async () => {
    fetchAgentRunDetail.mockResolvedValue({
      run: {
        id: "arun_1",
        agentType: "AssistantChat",
        status: "AwaitingDecision",
        trigger: "Schedule",
        summary: "Looked at the held invoices",
        modelIdentifier: "m",
        errorMessage: "",
        handedBy: null,
        createdAt: 1_800_000_000,
      },
      transcript: null,
    });
    render(
      wrap(
        <ProposalActionsContext.Provider value={actions()}>
          <ProposalSheet proposal={proposal()} onClose={() => {}} />
        </ProposalActionsContext.Provider>,
      ),
    );

    await userEvent.click(await screen.findByRole("button", { name: "Open the run" }));
    expect(
      await screen.findByRole("complementary", { name: "Looked at the held invoices" }),
    ).toBeTruthy();
    expect(fetchAgentRunDetail).toHaveBeenCalledWith("arun_1", expect.anything());
  });
});

function exception(overrides: Partial<AgentExceptionRow> = {}): AgentExceptionRow {
  return {
    id: "aexc_1",
    runId: "arun_1",
    category: "MissingData",
    severity: "Medium",
    subjectType: "Shipment",
    subjectId: "shp_1",
    attemptSummary: "Could not find a rate for the lane",
    blastRadius: 1,
    resolutionState: "Open",
    resolutionNotes: "",
    createdAt: 1_800_000_000,
    ...overrides,
  } as unknown as AgentExceptionRow;
}

describe("ExceptionSheet", () => {
  it("shows what happened and moves an open case on", async () => {
    const transition = vi.fn();
    const open = exception();
    render(
      wrap(
        <ExceptionActionsContext.Provider value={{ canResolve: true, transition }}>
          <ExceptionSheet exception={open} onClose={() => {}} />
        </ExceptionActionsContext.Provider>,
      ),
    );

    const sheet = await screen.findByRole("complementary", { name: "Missing Data" });
    expect(within(sheet).getByText("Could not find a rate for the lane")).toBeTruthy();
    await userEvent.click(within(sheet).getByRole("button", { name: "Mark resolved" }));
    expect(transition).toHaveBeenCalledWith(open, "Resolved");
    await userEvent.click(within(sheet).getByRole("button", { name: "Mark in review" }));
    expect(transition).toHaveBeenCalledWith(open, "InReview");
  });

  it("reopens a settled case and offers nothing to someone who may not resolve", async () => {
    const transition = vi.fn();
    const settled = exception({ resolutionState: "Resolved", resolutionNotes: "Rate added" });
    const { unmount } = render(
      wrap(
        <ExceptionActionsContext.Provider value={{ canResolve: true, transition }}>
          <ExceptionSheet exception={settled} onClose={() => {}} />
        </ExceptionActionsContext.Provider>,
      ),
    );

    expect(await screen.findByText("Rate added")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Mark resolved" })).toBeNull();
    await userEvent.click(screen.getByRole("button", { name: "Reopen" }));
    expect(transition).toHaveBeenCalledWith(settled, "Open");
    unmount();

    render(
      wrap(
        <ExceptionActionsContext.Provider value={{ canResolve: false, transition }}>
          <ExceptionSheet exception={exception()} onClose={() => {}} />
        </ExceptionActionsContext.Provider>,
      ),
    );
    expect(await screen.findByText("Could not find a rate for the lane")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Mark resolved" })).toBeNull();
  });
});
