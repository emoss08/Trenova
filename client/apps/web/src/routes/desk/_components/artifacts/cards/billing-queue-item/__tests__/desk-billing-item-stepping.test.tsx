import { clearPermissions, seedPermissions } from "@/test/permissions";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { installIntersectionObserver } from "@/test/intersection-observer";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import type { BillingQueueItem } from "@trenova/shared/types/billing-queue";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskBillingItem } from "../desk-billing-item";

const billingQueueService = vi.hoisted(() => ({
  getById: vi.fn(),
  getNeighbors: vi.fn(),
  getActivity: vi.fn(),
}));
const documentService = vi.hoisted(() => ({ getByResource: vi.fn() }));

vi.mock("@/services/api", () => ({ apiService: { billingQueueService, documentService } }));
vi.mock("@/lib/graphql/select-options", () => ({
  fetchGraphQLSelectOptions: vi.fn().mockResolvedValue({ results: [] }),
}));

/** A billing queue item as the queue returns it, before its review has been read. */
function item(fields: Partial<BillingQueueItem> = {}): BillingQueueItem {
  return {
    id: "bqi_1",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    shipmentId: "shp_1",
    number: "INV2610000079",
    status: "ReadyForReview",
    billType: "Invoice",
    version: 1,
    createdAt: 1_790_000_000,
    updatedAt: 1_790_000_000,
    ...fields,
  } as BillingQueueItem;
}

beforeEach(() => {
  seedPermissions({
    [Resource.BillingQueue]: Operation.Read | Operation.Update,
    [Resource.Document]: Operation.Read,
  });
  installIntersectionObserver();
  billingQueueService.getById.mockReset();
  billingQueueService.getActivity.mockResolvedValue({ items: [], hasMore: false });
  billingQueueService.getNeighbors.mockResolvedValue({});
  documentService.getByResource.mockResolvedValue([]);
});

afterEach(() => {
  clearPermissions();
  vi.unstubAllGlobals();
});

/*
A biller steps from one queue item to the next. The card's buttons act on the
item it was asked for, so until that item is read the card shows its outline,
never the item it left: a click there would land on an item no one can see.
*/
describe("DeskBillingItem stepping", () => {
  it("shows the next item's outline, not the item it left, while the next one is read", async () => {
    billingQueueService.getById.mockResolvedValue(item());
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const card = (itemId: string) => (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <DeskBillingItem itemId={itemId} fallback={<p>Reading the item</p>} onSelect={vi.fn()} />
        </MemoryRouter>
      </QueryClientProvider>
    );
    const { rerender } = render(card("bqi_1"));
    await screen.findByText("INV2610000079");

    billingQueueService.getById.mockReturnValue(new Promise(() => {}));
    rerender(card("bqi_2"));

    expect(screen.queryByText("INV2610000079")).toBeNull();
    expect(screen.getByText("Reading the item")).toBeInTheDocument();
  });
});
