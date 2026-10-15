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

function approvedItem(): BillingQueueItem {
  return {
    id: "bqi_1",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    shipmentId: "shp_1",
    number: "INV2610000079",
    status: "Approved",
    billType: "Invoice",
    allocatedTotalAmount: 1250,
    version: 2,
    createdAt: 1_790_000_000,
    updatedAt: 1_790_000_000,
  } as BillingQueueItem;
}

function draw() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskBillingItem itemId="bqi_1" fallback={<p>Reading the item</p>} onSelect={vi.fn()} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  installIntersectionObserver();
  billingQueueService.getById.mockResolvedValue(approvedItem());
  billingQueueService.getActivity.mockResolvedValue({ items: [], hasMore: false });
  billingQueueService.getNeighbors.mockResolvedValue({});
  documentService.getByResource.mockResolvedValue([]);
});

afterEach(() => {
  clearPermissions();
  vi.unstubAllGlobals();
});

/*
Posting books the invoice to the ledger and sends it to the customer, so it is
its own permission on invoices. A biller who may work the queue but not post
sees who finishes the item instead of a button the server would refuse.
*/
describe("DeskBillingItem posting", () => {
  it("offers posting an approved item to someone who may post invoices", async () => {
    seedPermissions({
      [Resource.BillingQueue]: Operation.Read | Operation.Update,
      [Resource.Invoice]: Operation.Read | Operation.Post,
      [Resource.Document]: Operation.Read,
    });
    draw();
    await screen.findByText("INV2610000079");

    expect(screen.getByRole("button", { name: /^Post / })).toBeEnabled();
    expect(screen.queryByText("Someone who can post invoices posts this one.")).toBeNull();
  });

  it("tells a biller who may not post invoices who posts the approved item", async () => {
    seedPermissions({
      [Resource.BillingQueue]: Operation.Read | Operation.Update,
      [Resource.Invoice]: Operation.Read | Operation.Update,
      [Resource.Document]: Operation.Read,
    });
    draw();
    await screen.findByText("INV2610000079");

    expect(screen.queryByRole("button", { name: /^Post / })).toBeNull();
    expect(screen.getByText("Someone who can post invoices posts this one.")).toBeInTheDocument();
  });
});
