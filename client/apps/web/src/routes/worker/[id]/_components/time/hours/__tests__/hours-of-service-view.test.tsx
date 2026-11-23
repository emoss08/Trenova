import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import HoursOfServiceView from "../hours-of-service-view";

const mocks = vi.hoisted(() => ({
  overview: vi.fn(),
  clocks: vi.fn(),
  dailyLogs: vi.fn(),
  logs: vi.fn(),
  violations: vi.fn(),
  forms: vi.fn(),
}));

vi.mock("@/lib/queries/worker", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/queries/worker")>()),
  workerOverviewQuery: (workerId: string) => ({
    queryKey: ["worker", "overview", workerId],
    queryFn: () => mocks.overview(workerId),
  }),
}));

vi.mock("@/lib/queries/telematics", () => ({
  telematics: {
    workerHosClocks: (workerId: string) => ({
      queryKey: ["clocks", workerId],
      queryFn: () => mocks.clocks(workerId),
    }),
    workerHosDailyLogs: (workerId: string, start: string, end: string) => ({
      queryKey: ["daily", workerId, start, end],
      queryFn: () => mocks.dailyLogs(workerId),
    }),
    workerHosLogs: (workerId: string, start: number, end: number) => ({
      queryKey: ["logs", workerId, start, end],
      queryFn: () => mocks.logs(workerId, start, end),
    }),
    workerHosViolations: (workerId: string, since: number) => ({
      queryKey: ["violations", workerId, since],
      queryFn: () => mocks.violations(workerId),
    }),
    workerFormSubmissions: (workerId: string, start: number, end: number) => ({
      queryKey: ["forms", workerId, start, end],
      queryFn: () => mocks.forms(workerId),
    }),
  },
}));

vi.mock("@trenova/shared/components/link", () => ({
  Link: ({ to, children, ...props }: { to: string; children?: React.ReactNode }) => (
    <a href={to} {...props}>
      {children}
    </a>
  ),
}));

function overviewWith(hos: unknown) {
  return { hos };
}

const clocks = {
  workerId: "wrk_1",
  dutyStatus: "driving",
  dutyStatusSince: 1_800_000_000,
  driveRemainingMs: 22_320_000,
  shiftRemainingMs: 15_720_000,
  cycleRemainingMs: 112_800_000,
  cycleTomorrowMs: 120_000_000,
  breakRemainingMs: 20_040_000,
  cycleStartedAt: null,
  shiftDrivingViolationMs: 0,
  cycleViolationMs: 0,
  driveLimitMs: 39_600_000,
  shiftLimitMs: 50_400_000,
  cycleLimitMs: 252_000_000,
  breakLimitMs: 28_800_000,
  recordedAt: 1_800_008_000,
};

function renderView() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <HoursOfServiceView workerId="wrk_1" />
    </QueryClientProvider>,
  );
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("HoursOfServiceView", () => {
  it("offers the integrations page when no telematics provider is connected", async () => {
    mocks.overview.mockResolvedValue(overviewWith({ link: "NotConnected" }));
    renderView();

    expect(await screen.findByText("Samsara is not connected")).toBeInTheDocument();
    expect(screen.getByText("Open integrations").closest("a")).toHaveAttribute(
      "href",
      "/admin/integrations",
    );
    expect(mocks.clocks).not.toHaveBeenCalled();
  });

  it("says the driver is not linked and offers the Samsara integration", async () => {
    mocks.overview.mockResolvedValue(overviewWith({ link: "NotLinked" }));
    renderView();

    expect(await screen.findByText("Not linked to a Samsara driver")).toBeInTheDocument();
    expect(screen.getByText("Open the Samsara integration").closest("a")).toHaveAttribute(
      "href",
      "/admin/integrations",
    );
    expect(mocks.clocks).not.toHaveBeenCalled();
  });

  it("says nothing has been reported yet for a linked driver with no stored state", async () => {
    mocks.overview.mockResolvedValue(overviewWith({ link: "NoData" }));
    renderView();

    expect(await screen.findByText("No hours of service reported yet")).toBeInTheDocument();
    expect(screen.queryByText("Open integrations")).not.toBeInTheDocument();
  });

  it("says the plan leaves hours of service out when the overview has no section", async () => {
    mocks.overview.mockResolvedValue(overviewWith(null));
    renderView();

    expect(await screen.findByText("Hours of service are not on your plan")).toBeInTheDocument();
  });

  it("shows a failed overview read as an error, not as an empty state", async () => {
    mocks.overview.mockRejectedValue(new Error("boom"));
    renderView();

    expect(await screen.findByText("The hours of service could not be loaded")).toBeInTheDocument();
    expect(screen.queryByText("Samsara is not connected")).not.toBeInTheDocument();
  });

  it("draws the duty band and the four clocks for a reporting driver", async () => {
    mocks.overview.mockResolvedValue(overviewWith({ link: "Reporting" }));
    mocks.clocks.mockResolvedValue(clocks);
    mocks.dailyLogs.mockResolvedValue([]);
    mocks.logs.mockResolvedValue([]);
    mocks.violations.mockResolvedValue([]);
    mocks.forms.mockResolvedValue([]);
    renderView();

    expect(await screen.findByText("Driving")).toBeInTheDocument();
    expect(screen.getByRole("meter", { name: "Break" })).toBeInTheDocument();
    expect(screen.getByRole("meter", { name: "Drive" })).toHaveAttribute("aria-valuenow", "372");
    expect(screen.getByRole("meter", { name: "Shift" })).toBeInTheDocument();
    expect(screen.getByRole("meter", { name: "Cycle" })).toBeInTheDocument();
    expect(screen.getByText("of 11h driving")).toBeInTheDocument();
    expect(await screen.findByText("No violations in the last 30 days.")).toBeInTheDocument();
  });

  it("falls back to the no-data sheet when the clocks read finds no stored state", async () => {
    mocks.overview.mockResolvedValue(overviewWith({ link: "Reporting" }));
    mocks.clocks.mockResolvedValue(null);
    mocks.dailyLogs.mockResolvedValue([]);
    mocks.logs.mockResolvedValue([]);
    renderView();

    expect(await screen.findByText("No hours of service reported yet")).toBeInTheDocument();
  });
});
