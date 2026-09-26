import type {
  AccountingDriftFinding,
  AccountingDriftFixPreview,
  AccountingDriftOverview,
} from "@/lib/graphql/accounting-drift";
import { recordPath } from "@/config/record-links";
import { accountingDriftWithinTolerance } from "@/lib/accounting-sync";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { NuqsTestingAdapter, type UrlUpdateEvent } from "nuqs/adapters/testing";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { activeDriftBucket, driftBucketFilter } from "../_components/drift-filters";
import { DriftFindingPanel } from "../_components/drift-panel";
import { dismissDriftSchema } from "../_components/drift-schemas";
import { DriftNotices, DriftSummary } from "../_components/drift-summary";

const mocks = vi.hoisted(() => ({
  fetchAccountingDriftOverview: vi.fn(),
  fetchAccountingDriftFinding: vi.fn(),
  fetchAccountingDriftFixPreview: vi.fn(),
  resolveAccountingDrift: vi.fn(),
  dismissAccountingDrift: vi.fn(),
  checkAccountingDrift: vi.fn(),
  granted: new Set<string>(),
}));

vi.mock("@/lib/graphql/accounting-drift", () => ({
  fetchAccountingDriftOverview: mocks.fetchAccountingDriftOverview,
  fetchAccountingDriftFinding: mocks.fetchAccountingDriftFinding,
  fetchAccountingDriftFixPreview: mocks.fetchAccountingDriftFixPreview,
  resolveAccountingDrift: mocks.resolveAccountingDrift,
  dismissAccountingDrift: mocks.dismissAccountingDrift,
  checkAccountingDrift: mocks.checkAccountingDrift,
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string, operation: number) => ({
    allowed: mocks.granted.has(`${resource}:${operation}`),
    isLoading: false,
  }),
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
}));

const SYNC_UPDATE = `${Resource.AccountingSync}:${Operation.Update}`;

function finding(overrides: Partial<AccountingDriftFinding> = {}): AccountingDriftFinding {
  return {
    id: "acctdf_1",
    connectionId: "acctc_1",
    objectType: "Invoice",
    objectId: "inv_1",
    objectNumber: "INV-1001",
    partyId: "cus_1",
    partyName: "Acme Foods",
    externalId: "145",
    externalUrl: "https://books.example/invoice/145",
    kind: "AmountMismatch",
    currencyCode: "USD",
    trenovaMinor: 125_000,
    providerMinor: 120_000,
    differenceMinor: -5_000,
    trenovaState: "Posted",
    providerState: "Posted",
    detail: [],
    providerModifiedAt: 1_790_208_300,
    providerModifiedBy: "J Doe",
    status: "Open",
    resolution: null,
    resolutionNote: "",
    fixObjectType: null,
    fixObjectId: null,
    directions: ["PushTrenovaValue", "AdjustTrenova"],
    pushed: false,
    resolvedBy: null,
    resolvedAt: null,
    detectedAt: 1_790_208_400,
    lastSeenAt: 1_790_208_400,
    version: 1,
    updatedAt: 1_790_208_400,
    ...overrides,
  };
}

function overview(overrides: Partial<AccountingDriftOverview> = {}): AccountingDriftOverview {
  return {
    connectionId: "acctc_1",
    providerName: "QuickBooks Online",
    checkedAt: 1_790_208_500,
    checkError: "",
    toleranceMinor: 500,
    currencyCode: "USD",
    summary: {
      open: 4,
      amountOpen: 2,
      goneOpen: 1,
      statusOpen: 1,
      balanceOpen: 0,
      resolvedSince: 7,
    },
    ...overrides,
  };
}

function fixPreview(overrides: Partial<AccountingDriftFixPreview> = {}): AccountingDriftFixPreview {
  return {
    direction: "AdjustTrenova",
    fixObject: "CreditMemo",
    operation: null,
    amountMinor: 5_000,
    currencyCode: "USD",
    toleranceMinor: 500,
    withinTolerance: false,
    summary:
      "Post a credit memo of 50.00 USD against INV-1001, bringing it to 1200.00 USD as in QuickBooks Online.",
    ...overrides,
  };
}

function renderWith(node: ReactNode, searchParams = "") {
  const urls: UrlUpdateEvent[] = [];
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const view = render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <NuqsTestingAdapter
          searchParams={searchParams}
          hasMemory
          onUrlUpdate={(event) => urls.push(event)}
        >
          {node}
        </NuqsTestingAdapter>
      </QueryClientProvider>
    </MemoryRouter>,
  );
  return { ...view, urls, client };
}

function renderPanel(row: AccountingDriftFinding | null, searchParams = "", toleranceMinor = 500) {
  return renderWith(
    <DriftFindingPanel
      open
      onOpenChange={() => undefined}
      mode="edit"
      row={row}
      providerName="QuickBooks Online"
      toleranceMinor={toleranceMinor}
    />,
    searchParams,
  );
}

beforeEach(() => {
  for (const mock of Object.values(mocks)) {
    if (typeof mock === "function") mock.mockReset();
  }
  mocks.granted = new Set([SYNC_UPDATE]);
});

describe("drift buckets", () => {
  it("round-trips each bucket, including the two-field ones", () => {
    for (const bucket of ["open", "amount", "gone", "settled"] as const) {
      expect(activeDriftBucket(driftBucketFilter(bucket))).toBe(bucket);
    }
    expect(driftBucketFilter("gone")).toEqual([
      { field: "status", operator: "in", value: ["Open"] },
      { field: "kind", operator: "in", value: ["DeletedInProvider", "VoidedInProvider"] },
    ]);
  });
});

describe("accountingDriftWithinTolerance", () => {
  it("counts only amount differences, either sign, up to the tolerance", () => {
    expect(
      accountingDriftWithinTolerance({ kind: "AmountMismatch", differenceMinor: -500 }, 500),
    ).toBe(true);
    expect(
      accountingDriftWithinTolerance({ kind: "AmountMismatch", differenceMinor: 501 }, 500),
    ).toBe(false);
    expect(
      accountingDriftWithinTolerance({ kind: "CustomerBalanceMismatch", differenceMinor: 10 }, 500),
    ).toBe(true);
    expect(
      accountingDriftWithinTolerance({ kind: "DeletedInProvider", differenceMinor: null }, 500),
    ).toBe(false);
    expect(accountingDriftWithinTolerance({ kind: "AmountMismatch", differenceMinor: 1 }, 0)).toBe(
      false,
    );
  });
});

describe("dismissDriftSchema", () => {
  it("needs a note and trims it", () => {
    expect(dismissDriftSchema.safeParse({ note: "   " }).success).toBe(false);
    expect(dismissDriftSchema.parse({ note: "  Rounding  " })).toEqual({ note: "Rounding" });
  });
});

describe("DriftSummary", () => {
  it("counts open differences and filters the table to deleted or voided ones", async () => {
    const { urls } = renderWith(<DriftSummary overview={overview()} />);

    expect(screen.getByRole("button", { name: /Open differences/ })).toHaveTextContent("4");
    expect(screen.getByRole("button", { name: /Settled/ })).toHaveTextContent("7");

    await userEvent.click(screen.getByRole("button", { name: /Deleted or voided/ }));
    await waitFor(() => expect(urls.length).toBeGreaterThan(0));
    expect(JSON.parse(urls.at(-1)!.searchParams.get("fieldFilters") ?? "[]")).toEqual(
      driftBucketFilter("gone"),
    );
  });
});

describe("DriftNotices", () => {
  it("says the last check could not read the books and why", () => {
    renderWith(
      <DriftNotices
        overview={overview({ checkError: "The authorization was revoked." })}
        providerName="QuickBooks Online"
      />,
    );

    expect(screen.getByText("The last check could not read QuickBooks Online")).toBeInTheDocument();
    expect(screen.getByText("The authorization was revoked.")).toBeInTheDocument();
  });

  it("says nothing when the last check read the books", () => {
    const { container } = renderWith(
      <DriftNotices overview={overview()} providerName="QuickBooks Online" />,
    );
    expect(container).toBeEmptyDOMElement();
  });
});

describe("DriftFindingPanel", () => {
  it("shows both sides, the editor and the tolerance", () => {
    renderPanel(finding());

    expect(screen.getByText("$1,250.00")).toBeInTheDocument();
    expect(screen.getByText("$1,200.00")).toBeInTheDocument();
    expect(screen.getByText("-$50.00")).toBeInTheDocument();
    expect(screen.getByText("$5.00")).toBeInTheDocument();
    expect(screen.getByText(/by J Doe/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "INV-1001" })).toHaveAttribute(
      "href",
      recordPath("invoice", "inv_1"),
    );
    expect(screen.queryByText("Within tolerance")).not.toBeInTheDocument();
  });

  it("previews a fix before making it, then makes it", async () => {
    mocks.fetchAccountingDriftFixPreview.mockResolvedValue(fixPreview());
    mocks.resolveAccountingDrift.mockResolvedValue(
      finding({ status: "Resolved", resolution: "AdjustedTrenova" }),
    );
    renderPanel(finding());

    await userEvent.click(screen.getByRole("button", { name: "Adjust Trenova" }));
    expect(
      await screen.findByText(
        "Post a credit memo of 50.00 USD against INV-1001, bringing it to 1200.00 USD as in QuickBooks Online.",
      ),
    ).toBeInTheDocument();
    expect(mocks.fetchAccountingDriftFixPreview).toHaveBeenCalledWith(
      { id: "acctdf_1", direction: "AdjustTrenova" },
      expect.anything(),
    );
    expect(mocks.resolveAccountingDrift).not.toHaveBeenCalled();

    await userEvent.click(screen.getByRole("button", { name: "Adjust Trenova" }));
    await waitFor(() =>
      expect(mocks.resolveAccountingDrift).toHaveBeenCalledWith({
        id: "acctdf_1",
        direction: "AdjustTrenova",
      }),
    );
  });

  it("says why a fix cannot be made and does not offer to make it", async () => {
    mocks.fetchAccountingDriftFixPreview.mockRejectedValue(
      new Error("INV-1001 already has a change on its way to QuickBooks Online."),
    );
    renderPanel(finding());

    await userEvent.click(screen.getByRole("button", { name: "Push Trenova's value" }));
    expect(await screen.findByText("This fix cannot be made now")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Push Trenova's value" })).toBeDisabled();
  });

  it("offers only the directions the finding lists", () => {
    renderPanel(
      finding({
        objectType: "CarrierBill",
        directions: ["PushTrenovaValue"],
      }),
    );

    expect(screen.getByRole("button", { name: "Push Trenova's value" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Adjust Trenova" })).not.toBeInTheDocument();
  });

  it("dismisses with a note", async () => {
    mocks.dismissAccountingDrift.mockResolvedValue(finding({ status: "Dismissed" }));
    renderPanel(finding({ differenceMinor: -300 }));

    expect(screen.getByText("Within tolerance")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Dismiss" }));
    expect(
      screen.getByText("Both sides stay as they are. The difference is within the tolerance."),
    ).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Dismiss difference" }));
    expect(await screen.findByText("Say why both sides stay as they are")).toBeInTheDocument();
    expect(mocks.dismissAccountingDrift).not.toHaveBeenCalled();

    await userEvent.type(
      screen.getByRole("textbox", { name: "Why do both sides stay as they are?" }),
      "  Rounding on their side  ",
    );
    await userEvent.click(screen.getByRole("button", { name: "Dismiss difference" }));
    await waitFor(() =>
      expect(mocks.dismissAccountingDrift).toHaveBeenCalledWith({
        id: "acctdf_1",
        note: "Rounding on their side",
      }),
    );
  });

  it("waits on a pushed value and offers nothing more", () => {
    renderPanel(finding({ pushed: true, fixObjectType: "SyncRecord", fixObjectId: "acctsr_9" }));

    expect(screen.getByText("Trenova's value was sent")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Dismiss" })).not.toBeInTheDocument();
  });

  it("offers nothing without update access", () => {
    mocks.granted = new Set();
    renderPanel(finding());

    expect(screen.queryByRole("button", { name: "Dismiss" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Adjust Trenova" })).not.toBeInTheDocument();
  });

  it("lists the documents inside a customer balance difference", () => {
    renderPanel(
      finding({
        objectType: "Customer",
        objectId: "cus_1",
        objectNumber: "Acme Foods",
        kind: "CustomerBalanceMismatch",
        directions: [],
        detail: [
          {
            objectType: "Invoice",
            objectId: "inv_2",
            objectNumber: "INV-1002",
            trenovaMinor: 40_000,
            providerMinor: 15_000,
          },
        ],
      }),
    );

    expect(screen.getByText("Documents that differ")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "INV-1002" })).toBeInTheDocument();
    expect(
      screen.getByText("$400.00 in Trenova, $150.00 in QuickBooks Online"),
    ).toBeInTheDocument();
  });

  it("loads the finding a link opens", async () => {
    mocks.fetchAccountingDriftFinding.mockResolvedValue(finding());
    renderPanel(null, "?panelType=edit&panelEntityId=acctdf_1");

    expect(await screen.findByText("$1,250.00")).toBeInTheDocument();
    expect(mocks.fetchAccountingDriftFinding).toHaveBeenCalledWith("acctdf_1", expect.anything());
  });
});
