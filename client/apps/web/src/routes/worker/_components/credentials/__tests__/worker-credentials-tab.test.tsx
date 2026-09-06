import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import WorkerCredentialsTab from "../../worker-credentials-tab";

const { fetchWorkerCredentialSummary, fetchWorkerCredentials, verifyWorkerCredential } = vi.hoisted(
  () => ({
    fetchWorkerCredentialSummary: vi.fn(),
    fetchWorkerCredentials: vi.fn(),
    verifyWorkerCredential: vi.fn(),
  }),
);
const dialogProps = vi.hoisted(() => ({ last: null as Record<string, unknown> | null }));
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
  archiveWorkerCredential: vi.fn(),
  WORKER_CREDENTIALS_KEY: "worker-credentials",
  WORKER_CREDENTIAL_SUMMARY_KEY: "worker-credential-summary",
  CREDENTIAL_EXPIRY_FORECAST_KEY: "credential-expiry-forecast",
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

vi.mock("../credential-form-dialog", () => ({
  CredentialFormDialog: (props: Record<string, unknown>) => {
    dialogProps.last = props;
    return props.open ? <div role="dialog">credential-form</div> : null;
  },
}));

vi.mock("sonner", () => ({
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

function renderTab() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <WorkerCredentialsTab workerId="wrk_1" />
    </QueryClientProvider>,
  );
  return queryClient;
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  dialogProps.last = null;
  permissionState.create = true;
  permissionState.update = true;
  permissionState.approve = true;
  permissionState.archive = true;
});

describe("WorkerCredentialsTab", () => {
  it("renders every required slot, the held optional credential, and the roll-up", async () => {
    fetchWorkerCredentialSummary.mockResolvedValue(summary);
    fetchWorkerCredentials.mockResolvedValue([cdlCred, forkCred, archived]);
    renderTab();

    const overview = await screen.findByTestId("credential-overview");
    expect(within(overview).getByText("1/2")).toBeInTheDocument();
    expect(within(overview).getByText("Non-compliant")).toBeInTheDocument();

    const cdlSlot = screen.getByTestId("credential-slot-wct_cdl");
    expect(cdlSlot).toHaveAttribute("data-health", "ExpiringSoon");
    expect(within(cdlSlot).getByText("12 days left")).toBeInTheDocument();

    const medSlot = screen.getByTestId("credential-slot-wct_med");
    expect(medSlot).toHaveAttribute("data-health", "Missing");
    expect(
      within(medSlot).getByRole("button", { name: "Add DOT Medical Card" }),
    ).toBeInTheDocument();

    const forkSlot = screen.getByTestId("credential-slot-wct_fork");
    expect(within(forkSlot).getByText("Verified")).toBeInTheDocument();
    expect(screen.queryByTestId("credential-slot-wcred_old")).not.toBeInTheDocument();
  });

  it("adds from a missing slot with the type preselected", async () => {
    fetchWorkerCredentialSummary.mockResolvedValue(summary);
    fetchWorkerCredentials.mockResolvedValue([cdlCred, forkCred]);
    renderTab();

    fireEvent.click(await screen.findByRole("button", { name: "Add DOT Medical Card" }));
    await waitFor(() => expect(screen.getByRole("dialog")).toBeInTheDocument());
    expect(dialogProps.last).toMatchObject({
      mode: "create",
      workerId: "wrk_1",
      credentialTypeId: "wct_med",
    });
  });

  it("renews from a slot's action menu carrying the current credential", async () => {
    fetchWorkerCredentialSummary.mockResolvedValue(summary);
    fetchWorkerCredentials.mockResolvedValue([cdlCred, forkCred]);
    renderTab();

    const cdlSlot = await screen.findByTestId("credential-slot-wct_cdl");
    fireEvent.click(
      within(cdlSlot).getByRole("button", { name: "Renew Commercial Driver's License" }),
    );
    await waitFor(() => expect(screen.getByRole("dialog")).toBeInTheDocument());
    expect(dialogProps.last).toMatchObject({
      mode: "renew",
      credentialTypeId: "wct_cdl",
      credential: expect.objectContaining({ id: "wcred_cdl" }),
    });
  });

  it("verifies straight from the slot and refreshes", async () => {
    fetchWorkerCredentialSummary.mockResolvedValue(summary);
    fetchWorkerCredentials.mockResolvedValue([cdlCred, forkCred]);
    verifyWorkerCredential.mockResolvedValue({ ...cdlCred, verifiedAt: 5 });
    renderTab();

    const cdlSlot = await screen.findByTestId("credential-slot-wct_cdl");
    fireEvent.click(
      within(cdlSlot).getByRole("button", { name: "Verify Commercial Driver's License" }),
    );
    await waitFor(() =>
      expect(verifyWorkerCredential).toHaveBeenCalledExactlyOnceWith("wcred_cdl", 2),
    );
    await waitFor(() => expect(fetchWorkerCredentialSummary).toHaveBeenCalledTimes(2));
  });

  it("hides mutations without permission", async () => {
    permissionState.create = false;
    permissionState.approve = false;
    fetchWorkerCredentialSummary.mockResolvedValue(summary);
    fetchWorkerCredentials.mockResolvedValue([cdlCred, forkCred]);
    renderTab();

    const medSlot = await screen.findByTestId("credential-slot-wct_med");
    expect(within(medSlot).queryByRole("button", { name: "Add DOT Medical Card" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Add credential" })).toBeNull();
    const cdlSlot = screen.getByTestId("credential-slot-wct_cdl");
    expect(
      within(cdlSlot).queryByRole("button", { name: "Verify Commercial Driver's License" }),
    ).toBeNull();
    expect(
      within(cdlSlot).queryByRole("button", { name: "Renew Commercial Driver's License" }),
    ).toBeNull();
  });

  it("keeps archived credentials in a collapsed history", async () => {
    fetchWorkerCredentialSummary.mockResolvedValue(summary);
    fetchWorkerCredentials.mockResolvedValue([cdlCred, forkCred, archived]);
    renderTab();

    const toggle = await screen.findByRole("button", { name: /History \(1\)/ });
    expect(screen.queryByTestId("credential-history-wcred_old")).not.toBeInTheDocument();
    fireEvent.click(toggle);
    const row = await screen.findByTestId("credential-history-wcred_old");
    expect(within(row).getByText("Superseded by renewal")).toBeInTheDocument();
  });
});
