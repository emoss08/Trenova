import { WorkPaneHost, WorkPaneProvider } from "@/components/work-pane";
import {
  CLOSED_WORKER_OPEN_STATE,
  type WorkerOpenState,
  type WorkerTarget,
} from "@/routes/worker/use-worker-state";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import WorkerDQFTab from "../worker-dqf-tab";

const {
  fetchDriverQualificationFile,
  markEmploymentVerificationRequested,
  recordEmploymentVerificationFollowUp,
  deleteEmploymentVerification,
  fetchWorkerCredentialSummary,
} = vi.hoisted(() => ({
  fetchDriverQualificationFile: vi.fn(),
  markEmploymentVerificationRequested: vi.fn(),
  recordEmploymentVerificationFollowUp: vi.fn(),
  deleteEmploymentVerification: vi.fn(),
  fetchWorkerCredentialSummary: vi.fn(),
}));

vi.mock("@/lib/graphql/worker-dqf", () => ({
  fetchDriverQualificationFile,
  markEmploymentVerificationRequested,
  recordEmploymentVerificationFollowUp,
  deleteEmploymentVerification,
  DRIVER_QUALIFICATION_FILE_KEY: "driver-qualification-file",
  OUTSTANDING_VERIFICATIONS_KEY: "outstanding-employment-verifications",
  DQF_RETENTION_CANDIDATES_KEY: "dqf-retention-candidates",
}));

vi.mock("@/lib/graphql/worker-credential", () => ({
  fetchWorkerCredentialSummary,
  WORKER_CREDENTIAL_SUMMARY_KEY: "worker-credential-summary",
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

vi.mock("../employer-form-pane", () => ({
  EmployerFormPane: ({
    open,
    verification,
  }: {
    open: boolean;
    verification?: { id: string } | null;
  }) => (open ? <div data-testid="employer-pane">{verification?.id ?? "new"}</div> : null),
}));

vi.mock("@trenova/shared/components/ui/toast", () => ({
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

const credentialSlot = (
  code: string,
  typeId: string,
  credentialId: string | null,
  verified: boolean,
) => ({
  health: "Valid",
  daysUntilExpiry: 100,
  required: true,
  credentialType: {
    id: typeId,
    code,
    name: code === "CDL" ? "Commercial driver's license" : "Medical examiner's certificate",
  },
  credential: credentialId ? { id: credentialId, verifiedAt: verified ? 1 : null } : null,
});

const credentialSummary = {
  workerId: "wrk_1",
  complianceStatus: "Pending",
  requiredCount: 2,
  validCount: 2,
  expiringCount: 1,
  expiredCount: 0,
  missingCount: 0,
  items: [
    credentialSlot("MEDICAL", "wct_med", "wcred_med", true),
    credentialSlot("CDL", "wct_cdl", "wcred_cdl", false),
  ],
};

function renderTab(
  data: unknown = file,
  open: Partial<WorkerOpenState> = {},
  summary: unknown = credentialSummary,
) {
  fetchDriverQualificationFile.mockResolvedValue(data);
  fetchWorkerCredentialSummary.mockResolvedValue(summary);
  const onOpen = vi.fn<(target: WorkerTarget) => void>();
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  render(
    <QueryClientProvider client={client}>
      <WorkPaneProvider>
        <WorkPaneHost>
          <WorkerDQFTab
            workerId="wrk_1"
            open={{ ...CLOSED_WORKER_OPEN_STATE, ...open }}
            onOpen={onOpen}
          />
        </WorkPaneHost>
      </WorkPaneProvider>
    </QueryClientProvider>,
  );
  return onOpen;
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

const tones = (element: HTMLElement) =>
  Array.from(element.querySelectorAll("[data-tone]")).map((node) => node.getAttribute("data-tone"));

describe("WorkerDQFTab", () => {
  it("draws the verdict and a spine segment for every row of each section", async () => {
    renderTab();

    const header = await screen.findByTestId("dqf-header");
    expect(within(header).getByText("Incomplete")).toBeInTheDocument();

    const spine = within(screen.getByTestId("dqf-spine"));
    const credentials = spine.getByTestId("dqf-spine-Credentials");
    expect(tones(credentials)).toEqual(["warning", "success"]);
    expect(credentials).toHaveTextContent("2 of 2");
    expect(credentials).toHaveTextContent("1 expiring soon");

    const documents = spine.getByTestId("dqf-spine-Documents");
    expect(tones(documents)).toEqual(["danger"]);
    expect(documents).toHaveTextContent("0 of 1");
    expect(documents).toHaveTextContent("1 outstanding");

    // Employers are read from the employers listed, both late against the deadline.
    const employers = spine.getByTestId("dqf-spine-SafetyHistory");
    expect(tones(employers)).toEqual(["danger", "danger"]);
    expect(employers).toHaveTextContent("0 of 2");

    expect(tones(spine.getByTestId("dqf-spine-DrugAlcohol"))).toEqual(["success"]);
  });

  it("brings a section into view and flashes it when its spine block is pressed", async () => {
    renderTab();

    fireEvent.click(await screen.findByTestId("dqf-spine-Documents"));
    const section = screen.getByRole("region", { name: "Documents" });
    expect(section.parentElement).toHaveAttribute("data-changed", "a");
  });

  // The office reads the top of the list and does that. A gap that stops the
  // file comes before a chase, a chase before a verification, and that before
  // a mere warning.
  it("lists the next steps worst first, each with the action that does it", async () => {
    const onOpen = renderTab();

    await waitFor(() =>
      expect(
        screen.getAllByTestId(/^dqf-step-/).map((step) => step.getAttribute("data-testid")),
      ).toEqual([
        "dqf-step-item:Documents:APPLICATION",
        "dqf-step-verification:ev_acme",
        "dqf-step-verification:ev_beta",
        "dqf-step-verify:wcred_cdl",
        "dqf-step-item:Credentials:MEDICAL",
      ]),
    );

    const first = screen.getByTestId("dqf-step-item:Documents:APPLICATION");
    fireEvent.click(within(first).getByRole("button", { name: "Open documents" }));
    expect(onOpen).toHaveBeenCalledWith({ tab: "files", view: "documents" });

    const renew = screen.getByTestId("dqf-step-item:Credentials:MEDICAL");
    fireEvent.click(within(renew).getByRole("button", { name: "Renew" }));
    expect(onOpen).toHaveBeenCalledWith({
      tab: "qualification",
      view: "credentials",
      form: "renew-credential",
      credential: "wcred_med",
    });
  });

  it("sends the request straight from the step and chases from the employer's menu", async () => {
    markEmploymentVerificationRequested.mockResolvedValue(
      employer("ev_acme", "Acme Freight", { status: "Requested" }),
    );
    recordEmploymentVerificationFollowUp.mockResolvedValue(
      employer("ev_beta", "Beta Logistics", { status: "Requested", followUpCount: 2 }),
    );
    renderTab();

    const step = await screen.findByTestId("dqf-step-verification:ev_acme");
    fireEvent.click(within(step).getByRole("button", { name: "Send the request" }));
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

  it("asks before removing an employer", async () => {
    deleteEmploymentVerification.mockResolvedValue(true);
    renderTab();

    const user = userEvent.setup();
    const acme = await screen.findByTestId("dqf-employer-ev_acme");
    await user.click(within(acme).getByRole("button", { name: "Actions for Acme Freight" }));
    await user.click(await screen.findByRole("menuitem", { name: "Remove" }));
    const dialog = await screen.findByRole("alertdialog", { name: "Remove this employer?" });
    expect(deleteEmploymentVerification).not.toHaveBeenCalled();
    await user.click(within(dialog).getByRole("button", { name: "Remove employer" }));
    await waitFor(() =>
      expect(deleteEmploymentVerification).toHaveBeenCalledExactlyOnceWith("ev_acme"),
    );
  });

  it("draws the employer's request as four stops", async () => {
    renderTab();

    const beta = await screen.findByTestId("dqf-employer-ev_beta");
    const stops = within(beta).getAllByRole("listitem");
    expect(stops.map((stop) => stop.getAttribute("data-state"))).toEqual([
      "done",
      "done",
      "current",
      "ahead",
    ]);
    expect(within(beta).getByText("Late")).toBeInTheDocument();
  });

  it("opens the employer panes from the address", async () => {
    renderTab(file, { form: "edit-employer", employer: "ev_beta" });
    expect(await screen.findByTestId("employer-pane")).toHaveTextContent("ev_beta");
  });

  it("says plainly when the file is complete", async () => {
    renderTab(
      {
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
      },
      {},
      { ...credentialSummary, items: [] },
    );

    expect(await screen.findByText("Complete")).toBeInTheDocument();
    expect(
      await screen.findByText("Nothing left to do. The file is complete."),
    ).toBeInTheDocument();
    expect(screen.queryAllByTestId(/^dqf-step-/)).toHaveLength(0);
  });

  it("says a failed read plainly", async () => {
    fetchWorkerCredentialSummary.mockResolvedValue(credentialSummary);
    fetchDriverQualificationFile.mockRejectedValue(new Error("boom"));
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <WorkPaneProvider>
          <WorkPaneHost>
            <WorkerDQFTab workerId="wrk_1" open={CLOSED_WORKER_OPEN_STATE} onOpen={vi.fn()} />
          </WorkPaneHost>
        </WorkPaneProvider>
      </QueryClientProvider>,
    );
    expect(
      await screen.findByText("The qualification file could not be loaded"),
    ).toBeInTheDocument();
  });
});
