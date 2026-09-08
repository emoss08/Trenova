import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import OshaLogConsole from "../osha-log-console";

const mocks = vi.hoisted(() => ({
  fetchOshaLog: vi.fn(),
  fetchOshaSummaries: vi.fn(),
  certifyOshaSummary: vi.fn(),
  uncertifyOshaSummary: vi.fn(),
  deleteWorkerInjury: vi.fn(),
}));

vi.mock("@/lib/graphql/worker-injury", () => ({
  ...mocks,
  OSHA_LOG_KEY: "osha-log",
  OSHA_SUMMARIES_KEY: "osha-summaries",
  WORKER_INJURIES_KEY: "worker-injuries",
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

vi.mock("../osha-summary-dialog", () => ({ OshaSummaryDialog: () => null }));
vi.mock("@/routes/worker/_components/safety/injury-dialog", () => ({
  InjuryDialog: () => null,
}));

vi.mock("@number-flow/react", () => ({
  default: ({ value, className, ...rest }: { value: number; className?: string }) => (
    <span className={className} {...rest}>
      {value}
    </span>
  ),
}));

const NOW = Date.UTC(2026, 8, 7, 15, 30) / 1000;
// The 2026 summary is posted February 1 to April 30, 2027.
const POST_FROM = Date.UTC(2027, 1, 1) / 1000;
const POST_THROUGH = Date.UTC(2027, 3, 30, 23, 59, 59) / 1000;

const ADA = { id: "wrk_ada", firstName: "Ada", lastName: "Byrne", profilePicUrl: "" };
const BEN = { id: "wrk_ben", firstName: "Ben", lastName: "Cole", profilePicUrl: "" };
const CARA = { id: "wrk_cara", firstName: "Cara", lastName: "Diaz", profilePicUrl: "" };
const DEV = { id: "wrk_dev", firstName: "Dev", lastName: "Ely", profilePicUrl: "" };

function totals(over: Record<string, number> = {}) {
  return {
    deaths: 0,
    daysAwayCases: 1,
    jobTransferCases: 1,
    otherRecordableCases: 1,
    totalRecordableCases: 3,
    totalDaysAway: 12,
    totalDaysRestricted: 5,
    injuryCount: 2,
    skinDisorderCount: 1,
    respiratoryCount: 0,
    poisoningCount: 0,
    hearingLossCount: 0,
    otherIllnessCount: 0,
    openCases: 1,
    ...over,
  };
}

function summary(over: Record<string, unknown> = {}) {
  return {
    id: "osum_2026",
    year: 2026,
    status: "Draft",
    naicsCode: "484121",
    averageEmployees: 120,
    totalHoursWorked: 240_000,
    executiveName: "Dana Reyes",
    executiveTitle: "VP Operations",
    executivePhone: null,
    certifiedAt: null,
    postedFrom: POST_FROM,
    postedThrough: POST_THROUGH,
    submittedAt: null,
    submissionReference: null,
    notes: null,
    version: 1,
    ...over,
  };
}

function oshaCase(over: Record<string, unknown> = {}) {
  return {
    id: "inj_1",
    workerId: ADA.id,
    caseNumber: 1,
    caseYear: 2026,
    classification: "DaysAway",
    illnessType: "Injury",
    treatment: "MedicalTreatment",
    status: "Closed",
    recordable: true,
    occurredAt: Date.UTC(2026, 2, 3) / 1000,
    reportedAt: Date.UTC(2026, 2, 3) / 1000,
    returnedToWorkAt: Date.UTC(2026, 2, 16) / 1000,
    logName: "Ada Byrne",
    location: "Yard, dock 4",
    description: "Slipped on the trailer step",
    bodyPart: "Left ankle",
    harmfulAgent: "Wet step",
    daysAway: 12,
    daysRestricted: 0,
    privacyCase: false,
    claimStatus: "Accepted",
    claimNumber: "WC-1001",
    claimCarrier: "Liberty",
    claimFiledAt: Date.UTC(2026, 2, 5) / 1000,
    claimClosedAt: null,
    safetyEventId: null,
    documentId: null,
    notes: null,
    version: 1,
    worker: ADA,
    ...over,
  };
}

const CASES = [
  oshaCase(),
  oshaCase({
    id: "inj_2",
    workerId: BEN.id,
    caseNumber: 2,
    classification: "OtherRecordable",
    status: "Open",
    logName: "Ben Cole",
    location: "Terminal B",
    description: "Strained back lifting a pallet",
    bodyPart: "Lower back",
    daysAway: 0,
    returnedToWorkAt: null,
    claimStatus: "NotFiled",
    claimNumber: null,
    claimCarrier: null,
    claimFiledAt: null,
    worker: BEN,
  }),
  oshaCase({
    id: "inj_3",
    workerId: CARA.id,
    caseNumber: 3,
    classification: "JobTransferOrRestriction",
    illnessType: "SkinDisorder",
    logName: "Privacy Case",
    location: "Wash bay",
    description: "Dermatitis from degreaser",
    bodyPart: "Hands",
    daysAway: 0,
    daysRestricted: 5,
    privacyCase: true,
    claimStatus: "NotFiled",
    worker: CARA,
  }),
  oshaCase({
    id: "inj_4",
    workerId: DEV.id,
    caseNumber: 4,
    classification: "FirstAidOnly",
    treatment: "FirstAid",
    recordable: false,
    logName: "Dev Ely",
    location: "Shop",
    description: "Paper cut, bandaged",
    bodyPart: "Right hand",
    daysAway: 0,
    claimStatus: "NotFiled",
    worker: DEV,
  }),
];

function log(over: Record<string, unknown> = {}) {
  return {
    year: 2026,
    postFrom: POST_FROM,
    postThrough: POST_THROUGH,
    totalRecordableIncidentRate: 2.5,
    daysAwayRestrictedRate: 1.67,
    totals: totals(),
    summary: summary(),
    cases: CASES,
    ...over,
  };
}

type Fixtures = {
  logs?: Record<string, unknown>;
  summaries?: unknown[];
};

function renderConsole(fixtures: Fixtures = {}) {
  const logs = fixtures.logs ?? { "2026": log() };
  mocks.fetchOshaLog.mockImplementation(
    async (year: number) =>
      logs[String(year)] ??
      log({ year, cases: [], totals: totals({ openCases: 0 }), summary: null }),
  );
  mocks.fetchOshaSummaries.mockResolvedValue(
    fixtures.summaries ?? [
      { id: "osum_2026", year: 2026, status: "Draft", certifiedAt: null, submittedAt: null },
      {
        id: "osum_2024",
        year: 2024,
        status: "Certified",
        certifiedAt: Date.UTC(2025, 0, 20) / 1000,
        submittedAt: null,
      },
    ],
  );
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <OshaLogConsole />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  vi.spyOn(Date, "now").mockReturnValue(NOW * 1000);
});

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  vi.clearAllMocks();
});

/** The form's mark and the figure beside it, as a reader of the paper 300A would pair them. */
function figure(scope: HTMLElement, name: string): [string, string] {
  const group = within(scope).getByRole("group", { name });
  const mark = group.firstElementChild?.firstElementChild?.textContent ?? "";
  const value = group.lastElementChild?.textContent ?? "";
  return [mark, value];
}

describe("OshaLogConsole", () => {
  // The loading state is the loaded page drawn in grey: the year picker, the
  // KPI strip, the 300A beside its track and the Form 300 table, at the sizes
  // they take once the log lands, so the page does not reflow when it does.
  it("draws the page's own shape while the log is still being read", () => {
    mocks.fetchOshaLog.mockReturnValue(new Promise(() => {}));
    mocks.fetchOshaSummaries.mockReturnValue(new Promise(() => {}));
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <MemoryRouter>
        <QueryClientProvider client={client}>
          <OshaLogConsole />
        </QueryClientProvider>
      </MemoryRouter>,
    );

    const loading = screen.getByLabelText("Loading the log");
    expect(loading).toHaveAttribute("aria-busy", "true");
    const hidden = { hidden: true } as const;
    const summary = within(loading).getByRole("region", { name: "Form 300A", ...hidden });
    expect(
      within(summary).getByRole("region", { name: "Number of cases", ...hidden }),
    ).toBeInTheDocument();
    const log = within(loading).getByRole("region", { name: "Form 300", ...hidden });
    expect(within(log).getAllByRole("row", hidden).length).toBeGreaterThan(1);
  });

  it("heads the year with the totals counted from the log, in the form's own columns", async () => {
    renderConsole();

    expect(
      await screen.findByLabelText("Recordable cases", { selector: "span" }),
    ).toHaveTextContent("3");
    expect(screen.getByText("1 still accruing days, 1 kept off the log")).toBeInTheDocument();
    expect(screen.getByLabelText("Incident rate")).toHaveTextContent("2.50");
    expect(screen.getByLabelText("DART rate")).toHaveTextContent("1.67");
    expect(screen.getByLabelText("Days lost", { selector: "span" })).toHaveTextContent("17");

    const form = screen.getByRole("region", { name: "Form 300A" });
    expect(figure(form, "Death")).toEqual(["G", "0"]);
    expect(figure(form, "Days away from work")).toEqual(["H", "1"]);
    expect(figure(form, "Job transfer or restriction")).toEqual(["I", "1"]);
    expect(figure(form, "Other recordable cases")).toEqual(["J", "1"]);
    expect(figure(form, "Skin disorders")).toEqual(["(2)", "1"]);
    expect(figure(form, "Total days of job transfer or restriction")).toEqual(["L", "5"]);
    expect(within(form).getByText("484121")).toBeInTheDocument();
    expect(within(form).getByText("240,000")).toBeInTheDocument();
  });

  it("captions each year with its summary and reads another year on request", async () => {
    const user = userEvent.setup();
    renderConsole({
      logs: {
        "2026": log(),
        "2024": log({
          year: 2024,
          cases: [oshaCase({ id: "inj_old", caseYear: 2024, logName: "Old Case" })],
          totals: totals({ openCases: 0 }),
          summary: summary({
            id: "osum_2024",
            year: 2024,
            status: "Certified",
            certifiedAt: Date.UTC(2025, 0, 20) / 1000,
          }),
        }),
      },
    });

    const years = await screen.findByRole("radiogroup", { name: "Log year" });
    await waitFor(() =>
      expect(within(years).getByRole("radio", { name: /2024/ })).toHaveTextContent("Certified"),
    );
    expect(within(years).getByRole("radio", { name: /2026/ })).toHaveTextContent("Draft");
    expect(within(years).getByRole("radio", { name: /2025/ })).toHaveTextContent("No summary");
    expect(within(years).getByRole("radio", { name: /2022/ })).toBeInTheDocument();

    await user.click(within(years).getByRole("radio", { name: /2024/ }));

    expect(await screen.findByRole("row", { name: "Case 2024-1" })).toBeInTheDocument();
    expect(mocks.fetchOshaLog).toHaveBeenCalledWith(2024, expect.anything());
    expect(screen.getByRole("button", { name: "Reopen" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Certify" })).not.toBeInTheDocument();
  });

  it("holds certification while a case is still accruing days, and certifies once it is not", async () => {
    const user = userEvent.setup();
    renderConsole({
      logs: {
        "2026": log(),
        "2025": log({ year: 2025, totals: totals({ openCases: 0 }), cases: [oshaCase()] }),
      },
    });

    const certify = await screen.findByRole("button", { name: "Certify" });
    expect(certify).toBeDisabled();
    expect(certify).toHaveAccessibleDescription(/Close the 1 case still accruing days first/);

    const track = screen.getByRole("list", { name: "Where 2026 stands" });
    const active = within(track)
      .getAllByRole("listitem")
      .find((item) => item.getAttribute("aria-current") === "step");
    expect(active).toHaveTextContent("Every case closed");
    expect(active).toHaveTextContent("1 case still accruing days");

    await user.click(screen.getByRole("radio", { name: /2025/ }));
    const ready = await screen.findByRole("button", { name: "Certify" });
    await waitFor(() => expect(ready).toBeEnabled());
    expect(screen.getByText("Ready to certify.")).toBeInTheDocument();

    mocks.certifyOshaSummary.mockResolvedValue(summary({ year: 2025, status: "Certified" }));
    await user.click(ready);
    await waitFor(() => expect(mocks.certifyOshaSummary).toHaveBeenCalledWith(2025));
  });

  it("filters the log to the open cases and searches the posted name", async () => {
    const user = userEvent.setup();
    renderConsole();

    expect(await screen.findAllByRole("row", { name: /^Case 2026-/ })).toHaveLength(4);
    expect(screen.getByRole("row", { name: "Case 2026-4" })).toHaveTextContent("Off the log");

    await user.click(screen.getByRole("radio", { name: /Still open/ }));
    expect(screen.getAllByRole("row", { name: /^Case 2026-/ })).toHaveLength(1);
    expect(screen.getByRole("row", { name: "Case 2026-2" })).toHaveTextContent("Ben Cole");

    await user.click(screen.getByRole("radio", { name: /Every case/ }));
    await user.type(screen.getByRole("searchbox", { name: "Search the log" }), "cara");
    expect(screen.getByText("No case matches")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Show every case" }));
    expect(screen.getAllByRole("row", { name: /^Case 2026-/ })).toHaveLength(4);
    expect(screen.getByRole("searchbox", { name: "Search the log" })).toHaveValue("");

    await user.type(screen.getByRole("searchbox", { name: "Search the log" }), "wash bay");
    expect(screen.getAllByRole("row", { name: /^Case 2026-/ })).toHaveLength(1);
    expect(screen.getByRole("row", { name: "Case 2026-3" })).toHaveTextContent("Privacy Case");
  });

  it("shows an empty page of the log for a year with nothing recorded", async () => {
    renderConsole({ logs: {} });

    expect(await screen.findByText("Nothing recorded for 2026")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open the workers list" })).toHaveAttribute(
      "href",
      "/hr/workers",
    );
    expect(screen.getByText("Nothing recorded this year")).toBeInTheDocument();
    expect(screen.queryByRole("radiogroup", { name: "Which cases" })).not.toBeInTheDocument();
  });

  it("opens a case to show who is behind a privacy case and where the claim stands", async () => {
    const user = userEvent.setup();
    renderConsole();

    await user.click(await screen.findByRole("button", { name: "2026-3" }));

    const sheet = await screen.findByRole("dialog");
    expect(within(sheet).getByText("Case 2026-3")).toBeInTheDocument();
    expect(within(sheet).getByText("Cara Diaz")).toBeInTheDocument();
    expect(within(sheet).getByRole("link", { name: /Cara Diaz/ })).toHaveAttribute(
      "href",
      "/hr/workers?entityId=wrk_cara&modType=edit&tab=safety",
    );
    expect(within(sheet).getByText(/Privacy case\. The posted log reads/)).toBeInTheDocument();
    expect(within(sheet).getByText("No claim has been filed.")).toBeInTheDocument();
    expect(within(sheet).getByRole("region", { name: "Outcome" })).toHaveTextContent("L5");
  });

  it("asks before deleting a case and then takes it off the log", async () => {
    const user = userEvent.setup();
    mocks.deleteWorkerInjury.mockResolvedValue(true);
    renderConsole();

    await user.click(await screen.findByRole("button", { name: "2026-1" }));
    await user.click(await screen.findByRole("button", { name: "Delete case" }));

    const confirm = await screen.findByRole("alertdialog");
    expect(within(confirm).getByText("Delete case 2026-1?")).toBeInTheDocument();
    expect(mocks.deleteWorkerInjury).not.toHaveBeenCalled();

    await user.click(within(confirm).getByRole("button", { name: "Delete case" }));
    await waitFor(() => expect(mocks.deleteWorkerInjury).toHaveBeenCalledWith("inj_1"));
  });
});
