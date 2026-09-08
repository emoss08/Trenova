import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import WorkerDQFTab from "../worker-dqf-tab";

const {
  fetchDriverQualificationFile,
  markEmploymentVerificationRequested,
  recordEmploymentVerificationFollowUp,
} = vi.hoisted(() => ({
  fetchDriverQualificationFile: vi.fn(),
  markEmploymentVerificationRequested: vi.fn(),
  recordEmploymentVerificationFollowUp: vi.fn(),
}));

vi.mock("@/lib/graphql/worker-dqf", () => ({
  fetchDriverQualificationFile,
  markEmploymentVerificationRequested,
  recordEmploymentVerificationFollowUp,
  deleteEmploymentVerification: vi.fn(),
  DRIVER_QUALIFICATION_FILE_KEY: "driver-qualification-file",
  OUTSTANDING_VERIFICATIONS_KEY: "outstanding-employment-verifications",
  DQF_RETENTION_CANDIDATES_KEY: "dqf-retention-candidates",
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

vi.mock("../employer-dialog", () => ({
  EmployerDialog: ({ open, verification }: { open: boolean; verification?: { id: string } }) =>
    open ? <div data-testid="employer-dialog">{verification?.id ?? "new"}</div> : null,
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn(), info: vi.fn() },
}));

const day = 86400;
const now = Math.floor(Date.now() / 1000);

const item = (section: string, code: string, name: string, status: string, detail = "") => ({
  section,
  code,
  name,
  status,
  detail,
  regulation: "49 CFR 391.51",
  expiresAt: null,
});

const employer = (id: string, name: string, overrides: Record<string, unknown> = {}) => ({
  id,
  workerId: "wrk_1",
  employerName: name,
  employerDotNumber: null,
  employerMcNumber: null,
  contactName: null,
  contactPhone: null,
  contactEmail: null,
  employedFrom: 1_600_000_000,
  employedTo: 1_650_000_000,
  wasDotRegulated: true,
  status: "Pending",
  method: "Email",
  requestedAt: null,
  responseReceivedAt: null,
  lastFollowUpAt: null,
  followUpCount: 0,
  drugAlcoholResponseReceivedAt: null,
  hadAccidents: false,
  accidentCount: 0,
  hadDrugAlcoholViolations: false,
  findings: null,
  notes: null,
  documentId: null,
  version: 1,
  ...overrides,
});

const file = {
  workerId: "wrk_1",
  complete: false,
  missingRequired: 1,
  expiringSoon: 1,
  expired: 0,
  outstanding: 1,
  hireDate: now - 40 * day,
  terminationDate: null,
  safetyHistoryDueAt: now - 10 * day,
  safetyHistoryLate: true,
  retentionExpiresAt: null,
  purgeEligible: false,
  items: [
    item(
      "Credentials",
      "MEDICAL",
      "Medical examiner's certificate",
      "ExpiringSoon",
      "Expires soon",
    ),
    item("Credentials", "CDL", "Commercial driver's license", "Satisfied", "On file"),
    item("Documents", "APPLICATION", "Application for employment", "Missing", "Not on file"),
    item("SafetyHistory", "SAFETY_HISTORY", "Previous employer safety history", "Outstanding"),
    item("DrugAlcohol", "CLEARINGHOUSE", "Clearinghouse query", "Satisfied", "Run this year"),
  ],
  verifications: [
    employer("ev_acme", "Acme Freight"),
    employer("ev_beta", "Beta Logistics", {
      status: "Requested",
      requestedAt: now - 30 * day,
      lastFollowUpAt: now - 20 * day,
      followUpCount: 1,
    }),
  ],
};

function renderTab(data: unknown = file, onOpenTab = vi.fn()) {
  fetchDriverQualificationFile.mockResolvedValue(data);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <WorkerDQFTab workerId="wrk_1" onOpenTab={onOpenTab} />
    </QueryClientProvider>,
  );
  return onOpenTab;
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("WorkerDQFTab", () => {
  it("reads the verdict, the deadline, and the spine of the file", async () => {
    renderTab();

    const header = await screen.findByTestId("dqf-header");
    expect(within(header).getByText("Incomplete")).toBeInTheDocument();
    expect(within(header).getByText(/investigation was due/i)).toBeInTheDocument();
    expect(within(header).getByText(/late/i)).toBeInTheDocument();

    const spine = within(screen.getByTestId("dqf-spine"));
    expect(spine.getByTestId("dqf-spine-Credentials")).toHaveTextContent("1/2");
    expect(spine.getByTestId("dqf-spine-Documents")).toHaveTextContent("0/1");
  });

  // The office reads the top of the list and does that. A gap that stops the
  // file comes before a chase, and a chase before a mere warning.
  it("lists the next steps worst first and sends each to where it is fixed", async () => {
    const onOpenTab = renderTab();

    const steps = await screen.findAllByTestId(/^dqf-step-/);
    expect(steps.map((step) => step.getAttribute("data-testid"))).toEqual([
      "dqf-step-item:Documents:APPLICATION",
      "dqf-step-verification:ev_acme",
      "dqf-step-verification:ev_beta",
      "dqf-step-item:Credentials:MEDICAL",
    ]);

    fireEvent.click(steps[0]);
    expect(onOpenTab).toHaveBeenCalledWith("documents");
  });

  it("sends the request straight from the step and chases from the employer's menu", async () => {
    markEmploymentVerificationRequested.mockResolvedValue(
      employer("ev_acme", "Acme Freight", { status: "Requested" }),
    );
    recordEmploymentVerificationFollowUp.mockResolvedValue(
      employer("ev_beta", "Beta Logistics", { status: "Requested", followUpCount: 2 }),
    );
    renderTab();

    fireEvent.click(await screen.findByTestId("dqf-step-verification:ev_acme"));
    await waitFor(() =>
      expect(markEmploymentVerificationRequested).toHaveBeenCalledExactlyOnceWith("ev_acme"),
    );

    const user = userEvent.setup();
    const beta = screen.getByTestId("dqf-employer-ev_beta");
    await user.click(within(beta).getByRole("button", { name: "Actions for Beta Logistics" }));
    await user.click(await screen.findByRole("menuitem", { name: "Chase again" }));
    await waitFor(() =>
      expect(recordEmploymentVerificationFollowUp).toHaveBeenCalledExactlyOnceWith("ev_beta"),
    );
  });

  it("says plainly when the file is complete", async () => {
    renderTab({
      ...file,
      complete: true,
      missingRequired: 0,
      expiringSoon: 0,
      outstanding: 0,
      safetyHistoryLate: false,
      items: file.items.map((row) => ({ ...row, status: "Satisfied" })),
      verifications: [
        employer("ev_acme", "Acme Freight", {
          status: "Received",
          requestedAt: now - 30 * day,
          responseReceivedAt: now - 20 * day,
          drugAlcoholResponseReceivedAt: now - 20 * day,
        }),
      ],
    });

    expect(await screen.findByText("Complete")).toBeInTheDocument();
    expect(screen.getByText(/nothing left to do/i)).toBeInTheDocument();
    expect(screen.queryAllByTestId(/^dqf-step-/)).toHaveLength(0);
  });
});
