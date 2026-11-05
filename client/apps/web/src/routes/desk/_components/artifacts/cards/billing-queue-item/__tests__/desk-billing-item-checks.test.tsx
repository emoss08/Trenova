import { clearPermissions, seedPermissions } from "@/test/permissions";
import { installIntersectionObserver } from "@/test/intersection-observer";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import type {
  BillingCheck,
  BillingIssue,
  BillingQueueItem,
} from "@trenova/shared/types/billing-queue";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskBillingItem } from "../desk-billing-item";

const billingQueueService = vi.hoisted(() => ({
  getById: vi.fn(),
  getNeighbors: vi.fn(),
  getActivity: vi.fn(),
}));
const documentService = vi.hoisted(() => ({ getByResource: vi.fn() }));

vi.mock("@/services/billing-queue", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/services/billing-queue")>()),
  billingQueueService,
}));

vi.mock("@/services/document", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/services/document")>()),
  documentService,
}));
vi.mock("@/lib/graphql/select-options", () => ({
  fetchGraphQLSelectOptions: vi.fn().mockResolvedValue({ results: [] }),
}));

const passing = (key: BillingCheck["key"], code: string): BillingCheck => ({
  key,
  state: "ok",
  code,
  detail: "",
  issueId: null,
  facts: null,
});

function chargeIssue(id: string, label: string, amount: string): BillingIssue {
  return {
    id,
    itemId: "bqi_1",
    checkKey: "charges",
    code: "AccessorialNotOnRateCon",
    subjectKey: id,
    summary: `$${amount} ${label} isn't on the rate con`,
    reasoning: null,
    source: "Deterministic",
    flaggedChargeId: id,
    options: [
      {
        key: "keep",
        label: `Keep it · bill $${amount}`,
        amount: Number(amount.replace(",", "")),
        done: `${label} kept`,
        effect: { kind: "keep", chargeId: id },
      },
      {
        key: "drop",
        label: "Remove it",
        amount: null,
        done: `${label} removed`,
        effect: { kind: "drop", chargeId: id },
      },
    ],
    resolutionKey: null,
    resolutionText: null,
    resolvedById: null,
    resolvedAt: null,
    requestedAt: null,
    undoable: false,
    createdAt: 1_800_000_000,
  };
}

const fuel = chargeIssue("bqis_fuel", "fuel surcharge", "1,911.58");
const stopOff = chargeIssue("bqis_stop", "stop-off charge", "100.00");

/** An item in review whose charges check waits on the given issues, as the server sends it. */
function item(charges: BillingCheck, issues: BillingIssue[], needsCount: number): BillingQueueItem {
  return {
    id: "bqi_1",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    shipmentId: "shp_1",
    number: "INV2610000079",
    status: "InReview",
    billType: "Invoice",
    version: 1,
    createdAt: 1_790_000_000,
    updatedAt: 1_790_000_000,
    review: {
      checks: [
        passing("biller", "assigned"),
        charges,
        passing("pod", "not_required"),
        passing("terms", "terms"),
        passing("duplicate", "unique"),
      ],
      issues,
      needsCount,
      ready: needsCount === 0,
      blocker: needsCount > 0 ? "issue" : "",
      documents: [],
      billerName: "Priya Shah",
    },
  } as unknown as BillingQueueItem;
}

function renderCard(value: BillingQueueItem) {
  billingQueueService.getById.mockResolvedValue(value);
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskBillingItem itemId="bqi_1" fallback={<p>Reading the item</p>} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  seedPermissions({
    [Resource.BillingQueue]: Operation.Read | Operation.Update,
    [Resource.Document]: Operation.Read,
  });
  installIntersectionObserver();
  billingQueueService.getActivity.mockResolvedValue({ items: [], hasMore: false });
  billingQueueService.getNeighbors.mockResolvedValue({});
  documentService.getByResource.mockResolvedValue([]);
});

afterEach(() => {
  clearPermissions();
  vi.clearAllMocks();
});

describe("DeskBillingItem checks", () => {
  it("lists every flagged charge with its own choices, and counts each as one that needs you", async () => {
    renderCard(
      item(
        { key: "charges", state: "warn", code: "issue", detail: "", issueId: "bqis_fuel" },
        [fuel, stopOff],
        2,
      ),
    );

    expect(await screen.findAllByText("2 need you")).toHaveLength(2);
    expect(screen.getByText("$1,911.58 fuel surcharge isn't on the rate con")).toBeInTheDocument();
    expect(screen.getByText("$100.00 stop-off charge isn't on the rate con")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Keep it · bill $1,911.58" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Keep it · bill $100.00" })).toBeEnabled();
    expect(screen.getAllByRole("button", { name: "Remove it" })).toHaveLength(2);
  });

  it("offers no choices on a check its last issue settled", async () => {
    const kept = {
      ...fuel,
      resolutionKey: "keep",
      resolutionText: "Fuel surcharge kept",
      resolvedAt: 1_800_000_100,
    };
    renderCard(
      item(
        {
          key: "charges",
          state: "ok",
          code: "resolved",
          detail: "Fuel surcharge kept",
          issueId: "bqis_fuel",
        },
        [kept],
        0,
      ),
    );

    expect(await screen.findAllByText("Fuel surcharge kept")).not.toHaveLength(0);
    expect(screen.queryByRole("button", { name: "Remove it" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Keep it/u })).not.toBeInTheDocument();
  });
});
