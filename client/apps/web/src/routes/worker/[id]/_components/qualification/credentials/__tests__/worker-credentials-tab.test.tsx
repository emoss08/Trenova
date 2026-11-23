import { WorkPaneHost, WorkPaneProvider } from "@/components/work-pane";
import {
  CLOSED_WORKER_OPEN_STATE,
  type WorkerOpenState,
  type WorkerTarget,
} from "@/routes/worker/use-worker-state";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { PENDING_DOCUMENTS_REFETCH_MS } from "@/lib/resource-documents";
import { documentContentUrl } from "@/services/document";
import { afterEach, describe, expect, it, onTestFinished, vi } from "vitest";
import WorkerCredentialsTab from "../worker-credentials-tab";

const {
  fetchWorkerCredentialSummary,
  fetchWorkerCredentials,
  verifyWorkerCredential,
  archiveWorkerCredential,
} = vi.hoisted(() => ({
  fetchWorkerCredentialSummary: vi.fn(),
  fetchWorkerCredentials: vi.fn(),
  verifyWorkerCredential: vi.fn(),
  archiveWorkerCredential: vi.fn(),
}));
const paneProps = vi.hoisted(() => ({ byMode: {} as Record<string, Record<string, unknown>> }));
const permissionState = vi.hoisted(() => ({
  create: true,
  update: true,
  approve: true,
  archive: true,
}));

vi.mock("@/lib/graphql/worker-credential", () => ({
  fetchWorkerCredentialSummary,
  fetchWorkerCredentials,
  verifyWorkerCredential,
  archiveWorkerCredential,
  WORKER_CREDENTIALS_KEY: "worker-credentials",
  WORKER_CREDENTIAL_SUMMARY_KEY: "worker-credential-summary",
  CREDENTIAL_EXPIRY_FORECAST_KEY: "credential-expiry-forecast",
}));

vi.mock("@/lib/queries/worker", () => ({
  worker: {
    overview: (workerId: string) => ({
      queryKey: ["worker", "overview", workerId],
      queryFn: async () => ({ asOf: 1_700_000_000, clearance: null }),
    }),
  },
}));

vi.mock("../../../overview/clearance-runway", () => ({
  ClearanceRunway: () => <div data-testid="runway" />,
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (_resource: string, operation: number) => {
    const map: Record<number, boolean> = {
      [1 << 1]: permissionState.create,
      [1 << 2]: permissionState.update,
      [1 << 8]: permissionState.approve,
      [1 << 12]: permissionState.archive,
    };
    return { allowed: map[operation] ?? true, isLoading: false };
  },
}));

vi.mock("../credential-form-pane", () => ({
  CredentialFormPane: (props: Record<string, unknown>) => {
    paneProps.byMode[props.mode as string] = props;
    return props.open ? <div data-testid={`credential-form-${String(props.mode)}`} /> : null;
  },
}));

vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { success: vi.fn(), error: vi.fn(), info: vi.fn() },
}));

const type = (
  id: string,
  name: string,
  overrides: Partial<Record<string, unknown>> = {},
): Record<string, unknown> => ({
  id,
  businessUnitId: "bu_1",
  organizationId: "org_1",
  code: id.toUpperCase(),
  name,
  description: null,
  category: "License",
  status: "Active",
  isRequired: true,
  requiredForDriverTypes: [],
  renewalWindowDays: 30,
  validityMonths: null,
  requiresNumber: false,
  requiresDocument: false,
  profileField: null,
  isSystem: true,
  sortOrder: 10,
  activeCredentialCount: 0,
  version: 0,
  createdAt: 1,
  updatedAt: 1,
  ...overrides,
});

const credential = (
  id: string,
  typeId: string,
  overrides: Partial<Record<string, unknown>> = {},
): Record<string, unknown> => ({
  id,
  workerId: "wrk_1",
  credentialTypeId: typeId,
  status: "Active",
  number: "D-1",
  issuingAuthority: "TX",
  issuedAt: 1_700_000_000,
  expiresAt: 1_800_000_000,
  documentId: null,
  notes: null,
  verifiedById: null,
  verifiedAt: null,
  archivedById: null,
  archivedAt: null,
  archiveReason: null,
  health: "Valid",
  daysUntilExpiry: 400,
  version: 2,
  createdAt: 1,
  updatedAt: 1,
  credentialType: null,
  document: null,
  verifiedBy: null,
  ...overrides,
});

const cdl = type("wct_cdl", "Commercial Driver's License", { requiresNumber: true });
const med = type("wct_med", "DOT Medical Card", { sortOrder: 20, requiresDocument: true });
const forklift = type("wct_fork", "Forklift Certification", { isRequired: false, sortOrder: 90 });

const cdlCred = credential("wcred_cdl", "wct_cdl", {
  health: "ExpiringSoon",
  daysUntilExpiry: 12,
  credentialType: cdl,
});
const forkCred = credential("wcred_fork", "wct_fork", {
  health: "Valid",
  verifiedAt: 1_700_000_100,
  verifiedBy: { id: "usr_1", name: "Ada" },
  credentialType: forklift,
});
const archived = credential("wcred_old", "wct_cdl", {
  status: "Archived",
  health: "Expired",
  daysUntilExpiry: -200,
  archivedAt: 1_750_000_000,
  archiveReason: "Superseded by renewal",
  credentialType: cdl,
});

const summary = {
  workerId: "wrk_1",
  complianceStatus: "NonCompliant",
  requiredCount: 2,
  validCount: 1,
  expiringCount: 1,
  expiredCount: 0,
  missingCount: 1,
  items: [
    {
      health: "ExpiringSoon",
      daysUntilExpiry: 12,
      required: true,
      credentialType: cdl,
      credential: cdlCred,
    },
    {
      health: "Missing",
      daysUntilExpiry: null,
      required: true,
      credentialType: med,
      credential: null,
    },
    {
      health: "Valid",
      daysUntilExpiry: 400,
      required: false,
      credentialType: forklift,
      credential: forkCred,
    },
  ],
};

/** Opens a row's actions menu and returns the user driving it. */
async function openActions(scope: HTMLElement, name: string) {
  const user = userEvent.setup();
  await user.click(within(scope).getByRole("button", { name }));
  return user;
}

function renderTab(open: Partial<WorkerOpenState> = {}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const onOpen = vi.fn<(target: WorkerTarget) => void>();
  const view = render(
    <QueryClientProvider client={queryClient}>
      <WorkPaneProvider>
        <WorkPaneHost>
          <WorkerCredentialsTab
            workerId="wrk_1"
            open={{ ...CLOSED_WORKER_OPEN_STATE, ...open }}
            onOpen={onOpen}
          />
        </WorkPaneHost>
      </WorkPaneProvider>
    </QueryClientProvider>,
  );
  return { onOpen, queryClient, view };
}

function loads(credentials: unknown[] = [cdlCred, forkCred, archived]) {
  fetchWorkerCredentialSummary.mockResolvedValue(summary);
  fetchWorkerCredentials.mockResolvedValue(credentials);
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  paneProps.byMode = {};
  permissionState.create = true;
  permissionState.update = true;
  permissionState.approve = true;
  permissionState.archive = true;
});

describe("WorkerCredentialsTab", () => {
  it("draws every required slot and the held optional credential in one grouped table", async () => {
    loads();
    renderTab();

    const overview = await screen.findByTestId("credential-overview");
    expect(within(overview).getByText("1 of 2")).toBeInTheDocument();
    expect(within(overview).getByText("Non-compliant")).toBeInTheDocument();
    expect(within(overview).getByText("1 unverified")).toBeInTheDocument();

    const table = screen.getByRole("table", { name: "Credentials on file" });
    expect(within(table).getByText("Required")).toBeInTheDocument();
    expect(within(table).getByText("Other")).toBeInTheDocument();

    const cdlRow = screen.getByTestId("credential-row-wct_cdl");
    expect(cdlRow).toHaveAttribute("data-health", "ExpiringSoon");
    expect(within(cdlRow).getByText("in 12 days")).toBeInTheDocument();
    expect(within(cdlRow).getByText("Unverified")).toBeInTheDocument();

    const medRow = screen.getByTestId("credential-row-wct_med");
    expect(within(medRow).getByText("Nothing on file")).toBeInTheDocument();
    expect(within(medRow).getByText("Missing")).toBeInTheDocument();

    const forkRow = screen.getByTestId("credential-row-wct_fork");
    expect(within(forkRow).getByText("Ada")).toBeInTheDocument();
    expect(screen.queryByTestId("credential-row-wcred_old")).not.toBeInTheDocument();
  });

  it("opens a held credential's read pane from its row", async () => {
    loads();
    const { onOpen } = renderTab();

    fireEvent.click(await screen.findByTestId("credential-row-wct_cdl"));
    expect(onOpen).toHaveBeenCalledExactlyOnceWith({
      tab: "qualification",
      view: "credentials",
      credential: "wcred_cdl",
    });
  });

  it("opens the add form with the type of a missing slot", async () => {
    loads();
    const { onOpen } = renderTab();

    fireEvent.click(await screen.findByTestId("credential-row-wct_med"));
    expect(onOpen).toHaveBeenCalledExactlyOnceWith({
      tab: "qualification",
      view: "credentials",
      form: "add-credential",
      kind: "wct_med",
    });
  });

  it("opens the add pane from the address with the type the link names", async () => {
    loads();
    renderTab({ form: "add-credential", kind: "wct_med" });

    expect(await screen.findByTestId("credential-form-create")).toBeInTheDocument();
    expect(paneProps.byMode.create).toMatchObject({
      open: true,
      mode: "create",
      workerId: "wrk_1",
      credentialTypeId: "wct_med",
    });
  });

  it("opens the renew pane from the address with the credential it renews", async () => {
    loads();
    renderTab({ form: "renew-credential", credential: "wcred_cdl" });

    expect(await screen.findByTestId("credential-form-renew")).toBeInTheDocument();
    expect(paneProps.byMode.renew).toMatchObject({
      credentialTypeId: "wct_cdl",
      credential: expect.objectContaining({ id: "wcred_cdl", version: 2 }),
    });
    expect(screen.queryByTestId("credential-form-create")).toBeNull();
  });

  it("renews from a row's menu through the address", async () => {
    loads();
    const { onOpen } = renderTab();

    const row = await screen.findByTestId("credential-row-wct_cdl");
    const user = await openActions(row, "Actions for Commercial Driver's License");
    await user.click(await screen.findByRole("menuitem", { name: "Renew" }));
    expect(onOpen).toHaveBeenCalledWith({
      tab: "qualification",
      view: "credentials",
      credential: "wcred_cdl",
      form: "renew-credential",
    });
  });

  it("reads the credential in the pane the address opens and verifies it, flashing its row", async () => {
    loads();
    verifyWorkerCredential.mockResolvedValue({ ...cdlCred, verifiedAt: 5 });
    renderTab({ credential: "wcred_cdl" });

    const pane = await screen.findByRole("dialog", { name: "Commercial Driver's License" });
    expect(within(pane).getByText("D-1")).toBeInTheDocument();
    expect(within(pane).getByText("Not verified")).toBeInTheDocument();
    expect(within(pane).getByText("Superseded by renewal")).toBeInTheDocument();

    fireEvent.click(within(pane).getByRole("button", { name: "Verify" }));
    await waitFor(() =>
      expect(verifyWorkerCredential).toHaveBeenCalledExactlyOnceWith("wcred_cdl", 2),
    );
    // Opening it from the address already washed the row once ("a"); the verify replays it.
    await waitFor(() =>
      expect(screen.getByTestId("credential-row-wct_cdl")).toHaveAttribute("data-changed", "b"),
    );
    await waitFor(() => expect(fetchWorkerCredentialSummary).toHaveBeenCalledTimes(2));
  });

  it("reads the credentials again while a document's preview is drawn, and stops once it is ready", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    onTestFinished(() => {
      vi.useRealTimers();
    });
    const scan = {
      id: "doc_cdl",
      fileName: "doc_cdl.jpg",
      originalName: "cdl-front.jpg",
      fileType: "image/jpeg",
      fileSize: 88_000,
      previewStatus: "Pending",
      createdAt: 1,
    };
    const pending = { ...cdlCred, documentId: "doc_cdl", document: scan };
    const ready = { ...pending, document: { ...scan, previewStatus: "Ready" } };
    fetchWorkerCredentialSummary.mockResolvedValue(summary);
    fetchWorkerCredentials
      .mockResolvedValueOnce([pending, forkCred, archived])
      .mockResolvedValue([ready, forkCred, archived]);
    renderTab({ credential: "wcred_cdl" });

    const pane = await screen.findByRole("dialog", { name: "Commercial Driver's License" });
    expect(within(pane).getByTitle("Generating thumbnail...")).toBeInTheDocument();
    expect(within(pane).queryByAltText("cdl-front.jpg")).not.toBeInTheDocument();
    expect(fetchWorkerCredentials).toHaveBeenCalledTimes(1);

    await act(() => vi.advanceTimersByTimeAsync(PENDING_DOCUMENTS_REFETCH_MS));

    await waitFor(() =>
      expect(within(pane).getByAltText("cdl-front.jpg")).toHaveAttribute(
        "src",
        documentContentUrl("doc_cdl", "preview"),
      ),
    );
    expect(fetchWorkerCredentials).toHaveBeenCalledTimes(2);

    await act(() => vi.advanceTimersByTimeAsync(PENDING_DOCUMENTS_REFETCH_MS * 3));
    expect(fetchWorkerCredentials).toHaveBeenCalledTimes(2);
    expect(fetchWorkerCredentialSummary).toHaveBeenCalledTimes(1);
  });

  it("reads the credentials once when no document's preview is waited on", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    onTestFinished(() => {
      vi.useRealTimers();
    });
    loads();
    renderTab();

    await screen.findByTestId("credential-row-wct_cdl");
    await act(() => vi.advanceTimersByTimeAsync(PENDING_DOCUMENTS_REFETCH_MS * 3));

    expect(fetchWorkerCredentials).toHaveBeenCalledTimes(1);
  });

  it("asks for a reason before archiving and sends it", async () => {
    loads();
    archiveWorkerCredential.mockResolvedValue({ ...cdlCred, status: "Archived" });
    const { onOpen } = renderTab({ credential: "wcred_cdl" });

    const pane = await screen.findByRole("dialog", { name: "Commercial Driver's License" });
    fireEvent.click(within(pane).getByRole("button", { name: "Archive" }));
    const dialog = await screen.findByRole("dialog", { name: "Archive this credential?" });
    fireEvent.click(within(dialog).getByRole("button", { name: "Archive" }));
    expect(await within(dialog).findByText("Say why")).toBeInTheDocument();
    expect(archiveWorkerCredential).not.toHaveBeenCalled();

    fireEvent.change(within(dialog).getByRole("textbox"), {
      target: { value: "Endorsement surrendered" },
    });
    fireEvent.click(within(dialog).getByRole("button", { name: "Archive" }));
    await waitFor(() =>
      expect(archiveWorkerCredential).toHaveBeenCalledExactlyOnceWith({
        id: "wcred_cdl",
        version: 2,
        reason: "Endorsement surrendered",
      }),
    );
    await waitFor(() =>
      expect(onOpen).toHaveBeenCalledWith({ tab: "qualification", view: "credentials" }),
    );
  });

  it("hides what the reader may not do", async () => {
    permissionState.create = false;
    permissionState.approve = false;
    loads([cdlCred, forkCred]);
    renderTab();

    const medRow = await screen.findByTestId("credential-row-wct_med");
    expect(within(medRow).queryByRole("button", { name: /Actions for/ })).toBeNull();
    expect(screen.queryByRole("button", { name: "Add credential" })).toBeNull();
    const cdlRow = screen.getByTestId("credential-row-wct_cdl");
    await openActions(cdlRow, "Actions for Commercial Driver's License");
    expect(await screen.findByRole("menuitem", { name: "Edit" })).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Verify" })).toBeNull();
    expect(screen.queryByRole("menuitem", { name: "Renew" })).toBeNull();
  });

  it("keeps archived credentials in a collapsed history", async () => {
    loads();
    renderTab();

    const toggle = await screen.findByRole("button", { name: /History/ });
    expect(toggle).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByTestId("credential-history-wcred_old")).not.toBeInTheDocument();
    fireEvent.click(toggle);
    const row = await screen.findByTestId("credential-history-wcred_old");
    expect(within(row).getByText("Superseded by renewal")).toBeInTheDocument();
  });

  it("says a failed read plainly instead of drawing an empty file", async () => {
    fetchWorkerCredentialSummary.mockRejectedValue(new Error("boom"));
    fetchWorkerCredentials.mockResolvedValue([]);
    renderTab();

    expect(await screen.findByText("The credentials could not be loaded")).toBeInTheDocument();
    expect(screen.queryByRole("table")).toBeNull();
  });
});
