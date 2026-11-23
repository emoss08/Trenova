import { WorkPaneProvider } from "@/components/work-pane";
import { CLOSED_WORKER_OPEN_STATE, type WorkerRecord } from "@/routes/worker/use-worker-state";
import { clearPermissions, permissionManifest } from "@/test/permissions";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { TestRouter } from "@trenova/shared/testing/test-router";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource, type UnavailableFeature } from "@trenova/shared/types/permission";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { clearanceItemName } from "../clearance-runway";
import OverviewTab from "../overview-tab";
import { translate } from "@trenova/shared/i18n/runtime";
import { AS_OF, DAY, overviewFixture } from "./overview-fixtures";

const fetchWorkerOverview = vi.hoisted(() => vi.fn());

vi.mock("@/lib/graphql/worker-overview", () => ({ fetchWorkerOverview }));

const READ = Operation.Read;
const ALL_READS: Record<string, number> = Object.fromEntries(
  [
    Resource.Worker,
    Resource.WorkerCredential,
    Resource.Qualification,
    Resource.WorkerDOTTest,
    Resource.WorkerTraining,
    Resource.PerformanceReview,
    Resource.WorkerSafetyEvent,
    Resource.WorkerChecklist,
    Resource.WorkerEmploymentEvent,
    Resource.WorkerPTO,
    Resource.WorkerLeave,
    Resource.DriverPayProfile,
    Resource.BenefitPlan,
    Resource.DriverPortal,
  ].map((resource) => [resource, READ]),
);

function signIn(unavailable: string[] = []) {
  const manifest = permissionManifest(ALL_READS);
  manifest.unavailableFeatures = unavailable.map((feature): UnavailableFeature => ({
    feature,
    featureName: feature,
    addOn: "",
    addOnName: "",
  }));
  usePermissionStore.setState({ manifest, lastFetched: Date.now(), isLoading: false });
}

function draw() {
  const onOpen = vi.fn();
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <TestRouter>
        <WorkPaneProvider>
          <OverviewTab
            workerId="wrk_1"
            worker={{ id: "wrk_1" } as WorkerRecord}
            view={null}
            open={CLOSED_WORKER_OPEN_STATE}
            onOpen={onOpen}
            shown
          />
        </WorkPaneProvider>
      </TestRouter>
    </QueryClientProvider>,
  );
  return { onOpen };
}

beforeEach(() => signIn());
afterEach(() => {
  vi.useRealTimers();
  clearPermissions();
  fetchWorkerOverview.mockReset();
});

describe("OverviewTab", () => {
  it("draws its own skeleton while the overview is read", () => {
    fetchWorkerOverview.mockReturnValue(new Promise(() => {}));
    draw();
    expect(screen.getByLabelText("Loading the overview")).toHaveAttribute("aria-busy", "true");
  });

  it("says the read failed rather than drawing an empty overview", async () => {
    fetchWorkerOverview.mockRejectedValue(new Error("boom"));
    draw();
    expect(await screen.findByText("The overview could not be loaded")).toBeInTheDocument();
    expect(screen.queryByText("Nothing needs attention right now.")).toBeNull();
  });

  it("gives the clearance verdict with what lapses first and how much comes due before it", async () => {
    fetchWorkerOverview.mockResolvedValue(overviewFixture());
    const { onOpen } = draw();
    const band = await screen.findByRole("region", { name: "Clearance" });
    expect(within(band).getByText("Cleared to drive")).toBeInTheDocument();
    expect(within(band).getByText("84 days")).toBeInTheDocument();
    expect(
      within(band).getByText(
        "Medical card lapses first. 2 things come due before then, and none of them stops driving.",
      ),
    ).toBeInTheDocument();
    fireEvent.click(within(band).getByRole("button", { name: "Renew Medical card" }));
    expect(onOpen).toHaveBeenCalledWith({
      tab: "qualification",
      view: "credentials",
      form: "renew-credential",
      credential: "cred_med",
    });
  });

  it("opens a marker's record from the runway", async () => {
    fetchWorkerOverview.mockResolvedValue(overviewFixture());
    const { onOpen } = draw();
    const marker = await screen.findByRole("button", { name: /^Hazmat awareness refresher,/ });
    fireEvent.click(marker);
    expect(onOpen).toHaveBeenCalledWith({
      tab: "development",
      view: "training",
      training: "trn_1",
    });
  });

  it("says plainly that no verdict is given when credentials are withheld", async () => {
    fetchWorkerOverview.mockResolvedValue(overviewFixture({ clearance: null }));
    draw();
    expect(await screen.findByTestId("clearance-withheld")).toHaveTextContent(
      "You don't have access to credentials, so this page gives no verdict on whether the driver is cleared to drive.",
    );
    expect(screen.queryByText("Cleared to drive")).toBeNull();
    expect(screen.queryByText("Not cleared to drive")).toBeNull();
  });

  it("turns to the danger tone with the reasons when the driver is not cleared", async () => {
    // Days not cleared are counted to the reader's today.
    vi.useFakeTimers({ toFake: ["Date"] });
    vi.setSystemTime(AS_OF * 1000);
    const lapsedAt = AS_OF - 3 * DAY;
    const base = overviewFixture();
    fetchWorkerOverview.mockResolvedValue(
      overviewFixture({
        clearance: {
          ...base.clearance!,
          cleared: false,
          clearedThrough: null,
          notClearedSince: lapsedAt,
          blocks: ["RequiredCredentialLapsed"],
          firstStop: null,
          items: base.clearance!.items.map((item) =>
            item.key === "medical" ? { ...item, dueAt: lapsedAt, daysUntil: -3 } : item,
          ),
        },
      }),
    );
    draw();
    const band = await screen.findByRole("region", { name: "Clearance" });
    expect(within(band).getByText("Not cleared to drive")).toBeInTheDocument();
    expect(within(band).getByText("3 days")).toBeInTheDocument();
    expect(within(band).getByText("A required credential has lapsed.")).toBeInTheDocument();
    expect(within(band).queryByText("Cleared to drive")).toBeNull();
    expect(within(band).getByRole("button", { name: "Renew Medical card" })).toBeInTheDocument();
  });

  it("says nothing needs attention when there is nothing", async () => {
    fetchWorkerOverview.mockResolvedValue(overviewFixture({ concerns: [], concernCount: 0 }));
    draw();
    expect(await screen.findByText("Nothing needs attention right now.")).toBeInTheDocument();
  });

  it("lists concerns under their group with the first row's step solid, and counts what was cut", async () => {
    fetchWorkerOverview.mockResolvedValue(overviewFixture({ concernCount: 5 }));
    const { onOpen } = draw();
    const panel = await screen.findByRole("region", { name: "Needs attention" });
    expect(within(panel).getByRole("region", { name: "Needs action" })).toHaveTextContent(
      "Ridgeline Freight has not answered the safety history request",
    );
    expect(within(panel).getByRole("region", { name: "Coming due" })).toHaveTextContent(
      "Medical card expires in 84 days",
    );
    expect(
      within(panel).getByText("3 more are not shown here. Each tab lists its own."),
    ).toBeInTheDocument();
    fireEvent.click(within(panel).getByRole("button", { name: "Renew" }));
    expect(onOpen).toHaveBeenCalledWith({
      tab: "qualification",
      view: "credentials",
      form: "renew-credential",
      credential: "cred_med",
    });
  });

  it("lays the file out only for the parts on the tenant's plan", async () => {
    signIn(["workforce.safety", "tms.settlement", "workforce.benefits"]);
    fetchWorkerOverview.mockResolvedValue(overviewFixture());
    const { onOpen } = draw();
    const file = await screen.findByRole("region", { name: "The file" });
    const cells = Array.from(file.querySelectorAll("[data-file-cell]")).map((cell) =>
      cell.getAttribute("data-file-cell"),
    );
    expect(cells).toEqual([
      "credentials",
      "dq-file",
      "drug-and-alcohol",
      "training",
      "reviews",
      "checklists",
      "time-off",
      "leave",
      "driver-portal",
    ]);
    fireEvent.click(within(file).getByRole("button", { name: /Leave/ }));
    expect(onOpen).toHaveBeenCalledWith({ tab: "time", view: "leave" });
  });

  it("draws the driver's day from stored hours: duty, tractor, three clocks and the break", async () => {
    fetchWorkerOverview.mockResolvedValue(overviewFixture());
    draw();
    const panel = await screen.findByRole("region", { name: "Right now" });
    expect(panel).toHaveTextContent(/since .+ · Tractor 4417/);
    expect(within(panel).getByText("Driving")).toBeInTheDocument();
    const meters = within(panel).getAllByRole("meter");
    expect(
      meters.map((meter) => [
        meter.getAttribute("aria-label"),
        meter.getAttribute("aria-valuetext"),
      ]),
    ).toEqual([
      ["Drive", "6h 12m left"],
      ["Shift", "4h 22m left"],
      ["Cycle", "31h 20m left"],
    ]);
    expect(within(panel).getByText("left of 70h")).toBeInTheDocument();
    expect(within(panel).getByText("Break due in 5h 34m")).toBeInTheDocument();
  });

  it("says a clock in words when its limit is not stored, and leaves the tractor out when none is matched", async () => {
    const base = overviewFixture();
    fetchWorkerOverview.mockResolvedValue(
      overviewFixture({ hos: { ...base.hos!, cycleLimitMs: null, tractorCode: null } }),
    );
    draw();
    const panel = await screen.findByRole("region", { name: "Right now" });
    expect(within(panel).getAllByRole("meter")).toHaveLength(2);
    expect(within(panel).getByText("Cycle: 31h 20m left")).toBeInTheDocument();
    expect(panel).not.toHaveTextContent("Tractor");
  });

  it("opens the next time off on Time › Time off", async () => {
    fetchWorkerOverview.mockResolvedValue(overviewFixture());
    const { onOpen } = draw();
    const panel = await screen.findByRole("region", { name: "Right now" });
    fireEvent.click(within(panel).getByRole("button", { name: /^Dec 23 to 24 · requested$/ }));
    expect(onOpen).toHaveBeenCalledWith({ tab: "time", view: "time-off", timeOff: "pto_1" });
  });

  it("names the Clearinghouse query on the runway rather than echoing the server's generic name", async () => {
    fetchWorkerOverview.mockResolvedValue(overviewFixture());
    draw();
    expect(
      await screen.findByRole("button", { name: /^Clearinghouse annual query,/ }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^Clearinghouse query,/ })).toBeNull();
  });

  it("says the driver's state from stored hours and never shows an id", async () => {
    fetchWorkerOverview.mockResolvedValue(
      overviewFixture({
        hos: {
          link: "NotLinked",
          dutyStatus: null,
          dutyStatusSince: null,
          driveRemainingMs: null,
          driveLimitMs: null,
          shiftRemainingMs: null,
          shiftLimitMs: null,
          cycleRemainingMs: null,
          cycleLimitMs: null,
          breakRemainingMs: null,
          breakLimitMs: null,
          tractorCode: null,
          recordedAt: null,
        },
      }),
    );
    draw();
    const panel = await screen.findByRole("region", { name: "Right now" });
    expect(panel).toHaveTextContent("This driver is not linked to a driver in telematics yet.");
    expect(document.body.textContent).not.toContain("cred_med");
    expect(document.body.textContent).not.toContain("wrk_1");
  });
});

describe("clearanceItemName", () => {
  it("names the generic items from their source and keeps a credential's or course's own name", () => {
    expect(
      clearanceItemName({ source: "PerformanceReview", name: "Performance review" }, translate),
    ).toBe("Performance review");
    expect(
      clearanceItemName({ source: "ClearinghouseQuery", name: "Clearinghouse query" }, translate),
    ).toBe("Clearinghouse annual query");
    expect(clearanceItemName({ source: "Credential", name: "Medical card" }, translate)).toBe(
      "Medical card",
    );
    expect(clearanceItemName({ source: "Training", name: "Defensive driving" }, translate)).toBe(
      "Defensive driving",
    );
  });
});
