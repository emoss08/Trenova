import type { SupportLogs } from "@/lib/graphql/support-logs";
import { renderAccountingPage } from "@/test/accounting-page-mocks";
import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { SendLogsSection } from "../_components/send-logs-section";

const mocks = vi.hoisted(() => ({ fetchSupportLogs: vi.fn(), requestSupportLogs: vi.fn() }));

vi.mock("@/lib/graphql/support-logs", () => ({
  fetchSupportLogs: mocks.fetchSupportLogs,
  requestSupportLogs: mocks.requestSupportLogs,
}));

const NOV_14 = 1_794_657_600;
const NOV_15 = 1_794_744_000;

// The shape of the supportLogRequests query (services/tms/internal/api/graphql/schema/support_access.graphqls).
function logs(overrides: Partial<SupportLogs> = {}): SupportLogs {
  return {
    managed: true,
    inFlight: false,
    remainingToday: 5,
    requests: [],
    ...overrides,
  } as SupportLogs;
}

const SENT = {
  id: "slr_01JTESTSENT0000000000000000",
  period: "Week",
  status: "Sent",
  requestedByName: "Ada Lovelace",
  requestedAt: NOV_14,
  finishedAt: NOV_15,
  failureMessage: null,
} as const;

const FAILED = {
  id: "slr_01JTESTFAILED00000000000000",
  period: "Hour",
  status: "Failed",
  requestedByName: "Grace Hopper",
  requestedAt: NOV_14,
  finishedAt: NOV_14 + 1800,
  failureMessage:
    "The instance did not start collecting the logs within 30 minutes. Try again, and contact Trenova support if it happens again.",
} as const;

const COLLECTING = {
  id: "slr_01JTESTCOLLECTING0000000000",
  period: "Day",
  status: "Collecting",
  requestedByName: "Ada Lovelace",
  requestedAt: NOV_15,
  finishedAt: null,
  failureMessage: null,
} as const;

afterEach(() => {
  cleanup();
  mocks.fetchSupportLogs.mockReset();
  mocks.requestSupportLogs.mockReset();
});

describe("Send logs to Trenova support", () => {
  it("says what is sent and how long it is kept, and offers the period and the button", async () => {
    mocks.fetchSupportLogs.mockResolvedValue(logs());

    renderAccountingPage(<SendLogsSection />);

    const panel = await screen.findByRole("region", { name: "Send logs to Trenova support" });
    expect(
      within(panel).getByText(
        "Trenova support receives this instance's logs for the period you choose and keeps them for 30 days.",
      ),
    ).toBeInTheDocument();
    expect(within(panel).getByRole("combobox")).toHaveTextContent("Last 24 hours");
    expect(within(panel).getByRole("button", { name: "Send logs" })).toBeEnabled();
    expect(within(panel).queryByRole("list")).not.toBeInTheDocument();
  });

  it("sends the chosen period and reads the list again", async () => {
    mocks.fetchSupportLogs.mockResolvedValue(logs());
    mocks.requestSupportLogs.mockResolvedValue(undefined);

    renderAccountingPage(<SendLogsSection />);

    fireEvent.click(await screen.findByRole("button", { name: "Send logs" }));

    await waitFor(() => expect(mocks.requestSupportLogs).toHaveBeenCalledWith("Day"));
    expect(mocks.requestSupportLogs).toHaveBeenCalledTimes(1);
    await waitFor(() => expect(mocks.fetchSupportLogs).toHaveBeenCalledTimes(2));
  });

  it("lists each request with its state, who asked and why one failed, and no ids", async () => {
    mocks.fetchSupportLogs.mockResolvedValue(logs({ requests: [SENT, FAILED] }));

    renderAccountingPage(<SendLogsSection />);

    const list = await screen.findByRole("list", { name: "Recent log requests" });
    const rows = within(list).getAllByRole("listitem");
    expect(rows).toHaveLength(2);
    expect(within(rows[0]!).getByText("Last 7 days")).toBeInTheDocument();
    expect(within(rows[0]!).getByText("Sent Nov 15")).toBeInTheDocument();
    expect(within(rows[0]!).getByText(/^Asked by Ada Lovelace on Nov 14/)).toBeInTheDocument();
    expect(within(rows[1]!).getByText("Last hour")).toBeInTheDocument();
    expect(within(rows[1]!).getByText("Failed")).toBeInTheDocument();
    expect(within(rows[1]!).getByText(FAILED.failureMessage)).toBeInTheDocument();
    expect(list).not.toHaveTextContent("slr_");
  });

  it("holds the button while a request is in flight and says why", async () => {
    mocks.fetchSupportLogs.mockResolvedValue(logs({ inFlight: true, requests: [COLLECTING] }));

    renderAccountingPage(<SendLogsSection />);

    expect(await screen.findByRole("button", { name: "Send logs" })).toBeDisabled();
    expect(
      screen.getByText(
        "Logs are on their way to Trenova support. You can send more once they arrive.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByText("Collecting")).toBeInTheDocument();
  });

  it("holds the button once the daily limit is reached", async () => {
    mocks.fetchSupportLogs.mockResolvedValue(logs({ remainingToday: 0 }));

    renderAccountingPage(<SendLogsSection />);

    expect(await screen.findByRole("button", { name: "Send logs" })).toBeDisabled();
    expect(
      screen.getByText(
        "This instance has sent logs 5 times in the last 24 hours. Try again later.",
      ),
    ).toBeInTheDocument();
  });

  it("shows nothing on a self-hosted install", async () => {
    mocks.fetchSupportLogs.mockResolvedValue(logs({ managed: false }));

    renderAccountingPage(<SendLogsSection />);

    await waitFor(() => expect(mocks.fetchSupportLogs).toHaveBeenCalled());
    await waitFor(() => expect(screen.queryByLabelText("Loading log requests")).toBeNull());
    expect(screen.queryByRole("region")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Send logs" })).not.toBeInTheDocument();
  });

  it("draws its skeleton while loading and an error it can retry", async () => {
    mocks.fetchSupportLogs.mockReturnValueOnce(new Promise(() => {}));

    renderAccountingPage(<SendLogsSection />);
    expect(screen.getByLabelText("Loading log requests")).toHaveAttribute("aria-busy", "true");

    cleanup();
    mocks.fetchSupportLogs.mockRejectedValue(new Error("down"));
    renderAccountingPage(<SendLogsSection />);
    expect(await screen.findByText("Log requests could not be loaded")).toBeInTheDocument();
  });
});
