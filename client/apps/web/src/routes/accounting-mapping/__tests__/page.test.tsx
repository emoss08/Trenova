import type {
  AccountingConnection,
  AccountingMapping,
  AccountingMappingSummary,
} from "@/lib/graphql/accounting-sync";
import { PageLayoutStub, renderAccountingPage } from "@/test/accounting-page-mocks";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AccountingMappingsPage } from "../page";

const mocks = vi.hoisted(() => ({
  fetchAccountingMappingSummary: vi.fn(),
  fetchAccountingMappings: vi.fn(),
  searchAccountingReferenceObjects: vi.fn(),
  confirmAccountingMappings: vi.fn(),
  setAccountingMapping: vi.fn(),
  rejectAccountingMapping: vi.fn(),
  clearAccountingMapping: vi.fn(),
  createAccountingReferenceRecord: vi.fn(),
  refreshAccountingReferenceData: vi.fn(),
  completeAccountingSetup: vi.fn(),
  canUpdate: true,
}));

vi.mock("@/lib/graphql/accounting-sync", () => ({
  fetchAccountingMappingSummary: mocks.fetchAccountingMappingSummary,
  fetchAccountingMappings: mocks.fetchAccountingMappings,
  searchAccountingReferenceObjects: mocks.searchAccountingReferenceObjects,
  confirmAccountingMappings: mocks.confirmAccountingMappings,
  setAccountingMapping: mocks.setAccountingMapping,
  rejectAccountingMapping: mocks.rejectAccountingMapping,
  clearAccountingMapping: mocks.clearAccountingMapping,
  createAccountingReferenceRecord: mocks.createAccountingReferenceRecord,
  refreshAccountingReferenceData: mocks.refreshAccountingReferenceData,
  completeAccountingSetup: mocks.completeAccountingSetup,
}));
vi.mock("@/components/navigation/sidebar-layout", () => ({ PageLayout: PageLayoutStub }));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: mocks.canUpdate, isLoading: false }),
}));
vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
}));

const connection: AccountingConnection = {
  id: "acctc_1",
  integrationType: "QuickBooksOnline",
  status: "Connected",
  appSource: "Instance",
  appEnvironment: "Production",
  externalCompanyName: "Peak Freight",
  externalLegalName: "Peak Freight LLC",
  externalCountry: "US",
  externalHomeCurrency: "USD",
  externalMultiCurrencyEnabled: false,
  externalBooksClosedThrough: null,
  lastCheckedAt: null,
  lastSuccessAt: null,
  lastFailureAt: null,
  consecutiveFailures: 0,
  lastErrorCategory: null,
  lastErrorMessage: "",
  lastWebhookAt: null,
  refreshTokenAbsoluteExpiresAt: 4_102_444_800,
  connectedAt: 1_780_000_000,
  disconnectedAt: null,
  setupStep: "Mappings",
  referenceRefreshStartedAt: null,
  referenceRefreshedAt: 1_780_000_100,
  referenceRefreshError: "",
  version: 1,
  updatedAt: 1_780_000_000,
};

function summary(overrides: Partial<AccountingMappingSummary> = {}): AccountingMappingSummary {
  return {
    integrationType: "QuickBooksOnline",
    providerName: "QuickBooks Online",
    connection,
    groups: [
      { targetType: "AccountRole", unmatched: 3, proposed: 2, confirmed: 1 },
      { targetType: "Customer", unmatched: 1, proposed: 4, confirmed: 0 },
    ],
    requiredTotal: 4,
    requiredConfirmed: 1,
    canCompleteSetup: false,
    ...overrides,
  };
}

const acme: AccountingMapping = {
  id: "acctm_acme",
  targetType: "Customer",
  trenovaObjectId: "cus_acme",
  trenovaKey: "",
  targetLabel: "Acme Logistics",
  providerKind: "Customer",
  externalId: "50",
  externalName: "Acme Logistics, Inc.",
  state: "Proposed",
  source: "Suggested",
  confidence: 0.97,
  reason: "Same company name",
  required: false,
  prechecked: true,
  candidates: [
    { externalId: "50", name: "Acme Logistics, Inc.", score: 0.97, reason: "Same company name" },
    { externalId: "51", name: "Acme Logistics LLC", score: 0.82, reason: "Similar name" },
  ],
  confirmedBy: null,
  confirmedAt: null,
  version: 3,
  updatedAt: 1_780_000_000,
};

describe("AccountingMappingsPage", () => {
  beforeEach(() => {
    for (const mock of Object.values(mocks)) {
      if (typeof mock === "function") mock.mockReset();
    }
    mocks.canUpdate = true;
    mocks.fetchAccountingMappingSummary.mockResolvedValue(summary());
    mocks.fetchAccountingMappings.mockResolvedValue({
      mappings: [acme],
      hasNextPage: false,
      endCursor: null,
    });
    mocks.searchAccountingReferenceObjects.mockResolvedValue([]);
  });

  it("sends a person to the integrations page before anything is connected", async () => {
    mocks.fetchAccountingMappingSummary.mockResolvedValue(summary({ connection: null }));

    renderAccountingPage(<AccountingMappingsPage />);

    expect(await screen.findByText("QuickBooks Online is not connected")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open integrations" })).toHaveAttribute(
      "href",
      "/admin/integrations?type=QuickBooksOnline",
    );
    expect(mocks.fetchAccountingMappings).not.toHaveBeenCalled();
  });

  it("totals the mappings and filters to the proposals when that figure is chosen", async () => {
    const user = userEvent.setup();
    renderAccountingPage(<AccountingMappingsPage />);

    expect(await screen.findByText("1 of 4")).toBeInTheDocument();
    expect(screen.getByText("6")).toBeInTheDocument();
    await user.click(screen.getByText("Proposed", { selector: "span" }));

    await waitFor(() =>
      expect(mocks.fetchAccountingMappings).toHaveBeenLastCalledWith(
        expect.objectContaining({
          filter: expect.objectContaining({ states: ["Proposed"] }),
        }),
        expect.anything(),
      ),
    );
  });

  it("maps to another record Trenova considered", async () => {
    mocks.setAccountingMapping.mockResolvedValue({ ...acme, externalId: "51", state: "Confirmed" });
    const user = userEvent.setup();
    renderAccountingPage(<AccountingMappingsPage />);

    await user.click(await screen.findByRole("button", { name: /Acme Logistics/ }));
    expect(screen.getByRole("link", { name: "Acme Logistics" })).toHaveAttribute(
      "href",
      "/billing/configuration-files/customers?panelType=edit&panelEntityId=cus_acme",
    );
    expect(screen.getByText("Acme Logistics LLC")).toBeInTheDocument();
    await user.click(screen.getAllByRole("button", { name: "Use" })[0]);

    await waitFor(() =>
      expect(mocks.setAccountingMapping).toHaveBeenCalledWith({
        mappingId: "acctm_acme",
        externalId: "51",
      }),
    );
  });

  it("creates the customer in QuickBooks under the name a person gives", async () => {
    mocks.createAccountingReferenceRecord.mockResolvedValue({ ...acme, state: "Confirmed" });
    const user = userEvent.setup();
    renderAccountingPage(<AccountingMappingsPage />);

    await user.click(await screen.findByRole("button", { name: /Acme Logistics/ }));
    const name = screen.getByRole("textbox", { name: "Name in QuickBooks Online" });
    await user.clear(name);
    await user.type(name, "Acme Logistics West");
    await user.click(screen.getByRole("button", { name: "Create in QuickBooks Online" }));

    await waitFor(() =>
      expect(mocks.createAccountingReferenceRecord).toHaveBeenCalledWith({
        mappingId: "acctm_acme",
        name: "Acme Logistics West",
      }),
    );
  });

  it("shows the mappings without changing anything to someone who may only read them", async () => {
    mocks.canUpdate = false;
    const user = userEvent.setup();
    renderAccountingPage(<AccountingMappingsPage />);

    await user.click(await screen.findByRole("button", { name: /Acme Logistics/ }));

    expect(screen.queryByRole("button", { name: "Confirm" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Use" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Confirm \d+ checked/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Finish setup" })).not.toBeInTheDocument();
  });

  it("waits to finish setup until the required mappings are confirmed", async () => {
    renderAccountingPage(<AccountingMappingsPage />);

    expect(await screen.findByRole("button", { name: "Finish setup" })).toBeDisabled();
  });
});
