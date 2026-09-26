import { InboundPaymentsLink } from "@/components/accounting-sync/inbound-payments-link";
import type {
  AccountingInboundApplyPreview,
  AccountingInboundChange,
  AccountingInboundOverview,
} from "@/lib/graphql/accounting-inbound";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { NuqsTestingAdapter, type UrlUpdateEvent } from "nuqs/adapters/testing";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { activeInboundBucket, inboundBucketFilter } from "../_components/inbound-filters";
import { InboundChangePanel } from "../_components/inbound-panel";
import { ignoreInboundSchema } from "../_components/inbound-schemas";
import { InboundNotices, InboundSummary } from "../_components/inbound-summary";

const mocks = vi.hoisted(() => ({
  fetchAccountingInboundOverview: vi.fn(),
  fetchAccountingInboundChange: vi.fn(),
  fetchAccountingInboundApplyPreview: vi.fn(),
  applyAccountingInboundChange: vi.fn(),
  ignoreAccountingInboundChange: vi.fn(),
  granted: new Set<string>(),
}));

vi.mock("@/lib/graphql/accounting-inbound", () => ({
  fetchAccountingInboundOverview: mocks.fetchAccountingInboundOverview,
  fetchAccountingInboundChange: mocks.fetchAccountingInboundChange,
  fetchAccountingInboundApplyPreview: mocks.fetchAccountingInboundApplyPreview,
  applyAccountingInboundChange: mocks.applyAccountingInboundChange,
  ignoreAccountingInboundChange: mocks.ignoreAccountingInboundChange,
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

function change(overrides: Partial<AccountingInboundChange> = {}): AccountingInboundChange {
  return {
    id: "acctic_1",
    kind: "CustomerPayment",
    status: "Proposed",
    reason: "PolicyPropose",
    resolution: "Pays INV-1001 and INV-1002 in full.",
    externalId: "301",
    externalNumber: "10442",
    externalUrl: "https://books.example/payment/301",
    providerModifiedAt: 1_790_208_300,
    providerModifiedBy: "J Doe",
    txnDate: 1_790_208_000,
    amountMinor: 150_025,
    currencyCode: "USD",
    partyName: "Acme Foods",
    partyObjectId: "cus_1",
    referenceNumber: "CHK-88",
    methodName: "Check",
    unappliedMinor: 0,
    lines: [
      {
        documentKind: "Invoice",
        documentExternalId: "145",
        amountMinor: 100_025,
        objectType: "Invoice",
        objectId: "inv_1",
        objectNumber: "INV-1001",
        openMinor: 100_025,
      },
      {
        documentKind: "Invoice",
        documentExternalId: "146",
        amountMinor: 50_000,
        objectType: "Invoice",
        objectId: "inv_2",
        objectNumber: "INV-1002",
        openMinor: 50_000,
      },
    ],
    appliedObjects: [],
    decidedBy: null,
    decidedAt: null,
    note: "",
    detectedAt: 1_790_208_400,
    version: 1,
    updatedAt: 1_790_208_400,
    ...overrides,
  };
}

function preview(
  overrides: Partial<AccountingInboundApplyPreview> = {},
): AccountingInboundApplyPreview {
  return {
    canApply: true,
    blocker: "",
    paidAt: 1_790_208_000,
    cashMinor: 150_025,
    unappliedMinor: 0,
    lines: [
      {
        objectType: "Invoice",
        objectId: "inv_1",
        objectNumber: "INV-1001",
        amountMinor: 100_025,
        openMinor: 100_025,
      },
      {
        objectType: "Invoice",
        objectId: "inv_2",
        objectNumber: "INV-1002",
        amountMinor: 50_000,
        openMinor: 50_000,
      },
    ],
    ...overrides,
  };
}

function overview(overrides: Partial<AccountingInboundOverview> = {}): AccountingInboundOverview {
  return {
    connectionId: "acctc_1",
    policy: "Propose",
    changesReadAt: 1_790_208_500,
    changesError: "",
    summary: { detected: 1, proposed: 3, appliedSince: 12, ignoredSince: 2 },
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

function renderPanel(row: AccountingInboundChange | null, searchParams = "") {
  return renderWith(
    <InboundChangePanel
      open
      onOpenChange={() => undefined}
      mode="edit"
      row={row}
      providerName="QuickBooks Online"
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

describe("inbound buckets", () => {
  it("round-trips each bucket through its filter", () => {
    for (const bucket of ["proposed", "detected", "applied", "ignored"] as const) {
      expect(activeInboundBucket(inboundBucketFilter(bucket))).toBe(bucket);
    }
  });

  it("recognises the status column's own single-value filter", () => {
    expect(activeInboundBucket([{ field: "status", operator: "eq", value: "Proposed" }])).toBe(
      "proposed",
    );
  });

  it("is no bucket for any other filter", () => {
    expect(activeInboundBucket([])).toBeNull();
    expect(
      activeInboundBucket([{ field: "status", operator: "in", value: ["Proposed", "Detected"] }]),
    ).toBeNull();
    expect(
      activeInboundBucket([{ field: "status", operator: "eq", value: "Superseded" }]),
    ).toBeNull();
    expect(
      activeInboundBucket([{ field: "reason", operator: "eq", value: "Proposed" }]),
    ).toBeNull();
    expect(
      activeInboundBucket([...inboundBucketFilter("proposed"), ...inboundBucketFilter("applied")]),
    ).toBeNull();
  });
});

describe("ignoreInboundSchema", () => {
  it("needs a note that says why, and trims it", () => {
    expect(ignoreInboundSchema.safeParse({ note: "   " }).success).toBe(false);
    expect(ignoreInboundSchema.safeParse({ note: "x".repeat(501) }).success).toBe(false);
    const parsed = ignoreInboundSchema.safeParse({ note: "  Keyed in by AR  " });
    expect(parsed.success).toBe(true);
    expect(parsed.data?.note).toBe("Keyed in by AR");
  });
});

describe("InboundSummary", () => {
  it("shows what waits and filters the table to it", async () => {
    const { urls } = renderWith(<InboundSummary overview={overview()} />);

    const waiting = screen.getByRole("button", { name: /Waiting for you/ });
    expect(waiting).toHaveTextContent("3");
    expect(screen.getByRole("button", { name: /Applied/ })).toHaveTextContent("12");

    await userEvent.click(waiting);
    await waitFor(() => expect(urls.length).toBeGreaterThan(0));
    expect(JSON.parse(urls.at(-1)!.searchParams.get("fieldFilters") ?? "[]")).toEqual([
      { field: "status", operator: "in", value: ["Proposed"] },
    ]);

    await userEvent.click(screen.getByRole("button", { name: /Waiting for you/ }));
    await waitFor(() => expect(urls.at(-1)!.searchParams.get("fieldFilters") ?? "").toBe(""));
  });
});

describe("InboundNotices", () => {
  it("says the books could not be read and why", () => {
    renderWith(
      <InboundNotices
        overview={overview({ changesError: "The connection was refused." })}
        system="QuickBooksOnline"
        providerName="QuickBooks Online"
      />,
    );

    expect(
      screen.getByText("Trenova could not read the latest changes from QuickBooks Online"),
    ).toBeInTheDocument();
    expect(screen.getByText("The connection was refused.")).toBeInTheDocument();
  });

  it("says payments are left out when the policy is off", () => {
    renderWith(
      <InboundNotices
        overview={overview({ policy: "Off" })}
        system="QuickBooksOnline"
        providerName="QuickBooks Online"
      />,
    );

    expect(
      screen.getByText(
        "Payments recorded in QuickBooks Online are left out of Trenova. Turn them on in the integration settings.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Integration settings" })).toHaveAttribute(
      "href",
      "/admin/integrations?type=QuickBooksOnline",
    );
  });
});

describe("InboundChangePanel", () => {
  it("shows what applying posts and applies it", async () => {
    mocks.fetchAccountingInboundApplyPreview.mockResolvedValue(
      preview({
        unappliedMinor: 5_000,
        lines: [
          {
            objectType: "Invoice",
            objectId: "inv_1",
            objectNumber: "INV-1001",
            amountMinor: 100_025,
            openMinor: 100_025,
          },
        ],
      }),
    );
    mocks.applyAccountingInboundChange.mockResolvedValue(change({ status: "Applied" }));
    renderPanel(change());

    expect(await screen.findByText("INV-1001, open $1,000.25")).toBeInTheDocument();
    expect(
      screen.getByText(/\$50\.00 stays on the account as unapplied cash\./),
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "INV-1002" })).toHaveAttribute(
      "href",
      "/billing/invoices?item=inv_2",
    );
    expect(mocks.fetchAccountingInboundApplyPreview).toHaveBeenCalledWith(
      "acctic_1",
      expect.anything(),
    );

    await userEvent.click(screen.getByRole("button", { name: "Apply in Trenova" }));
    await waitFor(() =>
      expect(mocks.applyAccountingInboundChange).toHaveBeenCalledWith("acctic_1"),
    );
  });

  it("says why it cannot be applied and offers no apply", async () => {
    mocks.fetchAccountingInboundApplyPreview.mockResolvedValue(
      preview({
        canApply: false,
        blocker: "INV-1001 was paid in Trenova after the payment was read.",
        lines: [],
      }),
    );
    renderPanel(change());

    expect(
      await screen.findByText("INV-1001 was paid in Trenova after the payment was read."),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Apply in Trenova" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Ignore" })).toBeInTheDocument();
  });

  it("ignores a payment that cannot be applied, with a note", async () => {
    mocks.ignoreAccountingInboundChange.mockResolvedValue(change({ status: "Ignored" }));
    renderPanel(
      change({
        reason: "Overpayment",
        resolution: "Pays $1,500.25 on INV-1001, which has $1,000.25 open in Trenova.",
      }),
    );

    expect(
      screen.getByText("Pays $1,500.25 on INV-1001, which has $1,000.25 open in Trenova."),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Apply in Trenova" })).not.toBeInTheDocument();
    expect(mocks.fetchAccountingInboundApplyPreview).not.toHaveBeenCalled();

    await userEvent.click(screen.getByRole("button", { name: "Ignore" }));
    await userEvent.click(screen.getByRole("button", { name: "Ignore payment" }));
    expect(
      await screen.findByText("Say why this payment stays out of Trenova"),
    ).toBeInTheDocument();
    expect(mocks.ignoreAccountingInboundChange).not.toHaveBeenCalled();

    await userEvent.type(
      screen.getByRole("textbox", { name: "Why does this payment stay out of Trenova?" }),
      "  Refund keyed by AR  ",
    );
    await userEvent.click(screen.getByRole("button", { name: "Ignore payment" }));
    await waitFor(() =>
      expect(mocks.ignoreAccountingInboundChange).toHaveBeenCalledWith({
        id: "acctic_1",
        note: "Refund keyed by AR",
      }),
    );
  });

  it("offers nothing without update access", async () => {
    mocks.granted = new Set();
    mocks.fetchAccountingInboundApplyPreview.mockResolvedValue(preview());
    renderPanel(change());

    expect(await screen.findByText("INV-1001, open $1,000.25")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Apply in Trenova" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Ignore" })).not.toBeInTheDocument();
  });

  it("links what was posted and who decided", () => {
    renderPanel(
      change({
        kind: "BillPayment",
        status: "Applied",
        reason: null,
        resolution: "",
        partyName: "Swift Haul LLC",
        lines: [
          {
            documentKind: "Bill",
            documentExternalId: "88",
            amountMinor: 150_025,
            objectType: "CarrierBill",
            objectId: "cset_1",
            objectNumber: "CS-0042",
            openMinor: 150_025,
          },
          {
            documentKind: "Other",
            documentExternalId: "90",
            amountMinor: 0,
            objectType: null,
            objectId: null,
            objectNumber: "",
            openMinor: 0,
          },
        ],
        appliedObjects: [{ type: "CarrierSettlement", id: "cset_1" }],
        decidedBy: { id: "usr_1", name: "Dana Whitfield" },
        decidedAt: 1_790_209_000,
      }),
    );

    expect(screen.getByRole("link", { name: "Carrier settlement" })).toHaveAttribute(
      "href",
      "/carrier-settlements/settlements?panelType=edit&panelEntityId=cset_1",
    );
    expect(screen.getByText("Not a document Trenova sent")).toBeInTheDocument();
    expect(screen.getByText(/Dana Whitfield/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Ignore" })).not.toBeInTheDocument();
    expect(mocks.fetchAccountingInboundApplyPreview).not.toHaveBeenCalled();
  });

  it("loads the payment a link opens", async () => {
    mocks.fetchAccountingInboundChange.mockResolvedValue(
      change({
        status: "Ignored",
        reason: "AlreadyPaid",
        note: "Duplicate",
        decidedBy: { id: "usr_1", name: "Dana Whitfield" },
        decidedAt: 1_790_209_000,
      }),
    );
    renderPanel(null, "?panelType=edit&panelEntityId=acctic_1");

    expect(await screen.findByText(/: Duplicate/)).toBeInTheDocument();
    expect(mocks.fetchAccountingInboundChange).toHaveBeenCalledWith("acctic_1", expect.anything());
  });
});

describe("InboundPaymentsLink", () => {
  it("counts the payments waiting to be applied", async () => {
    mocks.fetchAccountingInboundOverview.mockResolvedValue(overview());
    renderWith(<InboundPaymentsLink system="QuickBooksOnline" />);

    expect(await screen.findByRole("link", { name: "Payments to apply (3)" })).toHaveAttribute(
      "href",
      "/accounting/sync/inbound",
    );
  });

  it("is hidden when payments are left out and none wait", async () => {
    mocks.fetchAccountingInboundOverview.mockResolvedValue(
      overview({
        policy: "Off",
        summary: { detected: 0, proposed: 0, appliedSince: 4, ignoredSince: 0 },
      }),
    );
    const { container, client } = renderWith(<InboundPaymentsLink system="QuickBooksOnline" />);

    await waitFor(() =>
      expect(
        client.getQueryCache().findAll({ predicate: (query) => query.state.status === "success" }),
      ).toHaveLength(1),
    );
    expect(container).toBeEmptyDOMElement();
  });
});
