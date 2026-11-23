import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { WorkerPTO } from "@trenova/shared/types/worker";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useRowFlash } from "../../../use-row-flash";
import { RequestsPanel, type RequestAction } from "../requests-panel";

const mocks = vi.hoisted(() => ({ approveWorkerPTO: vi.fn() }));
const permissions = vi.hoisted(() => ({ approve: true, reject: true }));

vi.mock("@/lib/graphql/worker-mutations", () => ({ approveWorkerPTO: mocks.approveWorkerPTO }));

vi.mock("@/hooks/use-permission", async () => {
  const { Operation: Op } = await import("@trenova/shared/types/permission");
  return {
    usePermission: (_resource: string, operation: number) => ({
      allowed:
        operation === Op.Approve
          ? permissions.approve
          : operation === Op.Reject
            ? permissions.reject
            : true,
      isLoading: false,
    }),
  };
});

const DAY = 86_400;
const START = 1_798_070_400;

const requested = {
  id: "wrkpto_1",
  workerId: "wrk_1",
  status: "Requested",
  type: "Vacation",
  startDate: START,
  endDate: START + DAY,
  reason: "Holidays",
  days: "2",
} as unknown as WorkerPTO;

const approved = {
  id: "wrkpto_2",
  workerId: "wrk_1",
  status: "Approved",
  type: "Sick",
  startDate: START - 90 * DAY,
  endDate: START - 90 * DAY,
  reason: "Appointment",
  days: "0.5",
  approver: { id: "usr_1", name: "Dana Whitfield" },
} as unknown as WorkerPTO;

const MORE_NONE = {
  hasMore: false,
  loadMore: () => undefined,
  loadingMore: false,
  loadMoreFailed: false,
  error: null,
};

function Harness({
  entries,
  hasMore,
  onDecided,
  onReason,
  onOpenRequest,
}: {
  entries: WorkerPTO[];
  hasMore: boolean;
  onDecided: () => Promise<unknown>;
  onReason: (action: RequestAction) => void;
  onOpenRequest: (id: string) => void;
}) {
  const { flash, flashOf } = useRowFlash();
  return (
    <RequestsPanel
      entries={entries}
      total={entries.length}
      more={{ ...MORE_NONE, hasMore }}
      onOpenRequest={onOpenRequest}
      requestPane={{ preload: () => Promise.resolve() }}
      onReason={onReason}
      onDecided={onDecided}
      flashOf={flashOf}
      onChanged={flash}
    />
  );
}

function renderPanel(entries: WorkerPTO[], { hasMore = false } = {}) {
  const onDecided = vi.fn().mockResolvedValue(undefined);
  const onReason = vi.fn();
  const onOpenRequest = vi.fn();
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  render(
    <QueryClientProvider client={client}>
      <Harness
        entries={entries}
        hasMore={hasMore}
        onDecided={onDecided}
        onReason={onReason}
        onOpenRequest={onOpenRequest}
      />
    </QueryClientProvider>,
  );
  return { onDecided, onReason, onOpenRequest };
}

function rowOf(text: string): HTMLElement {
  const row = screen.getByText(text).closest("tr");
  if (!row) throw new Error(`no row for ${text}`);
  return row;
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  permissions.approve = true;
  permissions.reject = true;
});

describe("RequestsPanel", () => {
  it("approves a waiting request from its row once, refreshes the figures and marks the row", async () => {
    let settle: (value: unknown) => void = () => undefined;
    mocks.approveWorkerPTO.mockImplementation(
      () =>
        new Promise((resolve) => {
          settle = resolve;
        }),
    );
    const user = userEvent.setup();
    const { onDecided } = renderPanel([requested, approved]);

    const row = rowOf("Vacation");
    const approve = within(row).getByRole("button", { name: "Approve" });
    await user.click(approve);
    await user.click(approve);

    expect(mocks.approveWorkerPTO).toHaveBeenCalledExactlyOnceWith("wrkpto_1");
    settle({ id: "wrkpto_1" });

    await waitFor(() => expect(onDecided).toHaveBeenCalledTimes(1));
    expect(rowOf("Vacation")).toHaveAttribute("data-changed", "a");
  });

  it("hands Reject to the reason dialog rather than deciding in place", async () => {
    const user = userEvent.setup();
    const { onReason } = renderPanel([requested]);

    await user.click(within(rowOf("Vacation")).getByRole("button", { name: "Reject" }));

    expect(onReason).toHaveBeenCalledExactlyOnceWith({ kind: "reject", pto: requested });
    expect(mocks.approveWorkerPTO).not.toHaveBeenCalled();
  });

  it("offers no Approve without the approve permission, and shows decided rows' deciders", () => {
    permissions.approve = false;
    renderPanel([requested, approved]);

    expect(screen.queryByRole("button", { name: "Approve" })).not.toBeInTheDocument();
    expect(within(rowOf("Vacation")).getByRole("button", { name: "Reject" })).toBeInTheDocument();
    expect(within(rowOf("Sick")).getByText("Dana Whitfield")).toBeInTheDocument();
    expect(within(rowOf("Sick")).getByText("0.5")).toBeInTheDocument();
    expect(screen.getByText("1 waiting on a decision")).toBeInTheDocument();
  });

  it("never says a decided request is waiting, and dots every status badge", () => {
    permissions.approve = false;
    permissions.reject = false;
    renderPanel([
      requested,
      { ...approved, id: "wrkpto_3", type: "Personal", approver: null, autoApproved: true },
      { ...approved, id: "wrkpto_4", type: "Bereavement", approver: null, autoApproved: false },
      {
        ...approved,
        id: "wrkpto_5",
        type: "Holiday",
        status: "Rejected",
        approver: null,
        rejector: { id: "usr_2", name: "Marcus Bell" },
        rejectionReason: "Peak week",
      },
    ] as unknown as WorkerPTO[]);

    expect(within(rowOf("Vacation")).getByText("Waiting")).toBeInTheDocument();
    expect(within(rowOf("Personal")).getByText("Approved automatically")).toBeInTheDocument();
    expect(within(rowOf("Bereavement")).getByText("—")).toBeInTheDocument();
    expect(within(rowOf("Holiday")).getByText("Marcus Bell")).toBeInTheDocument();
    expect(within(rowOf("Holiday")).getByText("Peak week")).toBeInTheDocument();
    expect(screen.getAllByText("Waiting")).toHaveLength(1);
    expect(document.querySelectorAll('[data-slot="badge-dot"]')).toHaveLength(4);
  });

  it("opens a request from anywhere on its row, by pointer or keyboard, but not from its buttons", async () => {
    const user = userEvent.setup();
    const { onOpenRequest } = renderPanel([requested, approved]);

    await user.click(within(rowOf("Sick")).getByText("Dana Whitfield"));
    expect(onOpenRequest).toHaveBeenLastCalledWith("wrkpto_2");

    rowOf("Vacation").focus();
    await user.keyboard("{Enter}");
    expect(onOpenRequest).toHaveBeenLastCalledWith("wrkpto_1");

    await user.click(within(rowOf("Vacation")).getByRole("button", { name: "Reject" }));
    expect(onOpenRequest).toHaveBeenCalledTimes(2);
  });

  it("says how many wait only once every request has been read", () => {
    renderPanel([requested, approved], { hasMore: true });
    expect(screen.queryByText("1 waiting on a decision")).not.toBeInTheDocument();
    expect(screen.queryByText("Nothing waiting")).not.toBeInTheDocument();
  });

  it("draws the empty table with its headings when nothing has been requested", () => {
    renderPanel([]);
    expect(screen.getByText("No time off on record")).toBeInTheDocument();
  });
});
