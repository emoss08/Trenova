import { documentContentUrl } from "@/services/document";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { Operation, Resource } from "@trenova/shared/types/permission";
import {
  FakeIntersectionObserver,
  installIntersectionObserver,
} from "@/test/intersection-observer";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen } from "@testing-library/react";
import type { BillingQueueItem } from "@trenova/shared/types/billing-queue";
import type { Document } from "@trenova/shared/types/document";
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

function proofOfDelivery(): Document {
  return {
    id: "doc_pod",
    originalName: "pod-1042.pdf",
    fileType: "application/pdf",
    fileSize: 120_000,
    createdAt: 1_790_000_000,
    isCurrentVersion: true,
    status: "Active",
    approvedAt: 1_790_000_100,
    previewStatus: "Ready",
    contentStatus: "Indexed",
    shipmentDraftStatus: "Unavailable",
    processingProfile: "none",
    documentType: { name: "Proof of delivery", code: "POD" },
  } as Document;
}

function draw() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskBillingItem itemId="bqi_1" fallback={<p>Reading the item</p>} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  seedPermissions({ [Resource.Document]: Operation.Read });
  installIntersectionObserver();
  billingQueueService.getById.mockReset();
  billingQueueService.getActivity.mockResolvedValue({ items: [], hasMore: false });
  billingQueueService.getNeighbors.mockResolvedValue({});
  documentService.getByResource.mockReset();
  documentService.getByResource.mockResolvedValue([proofOfDelivery()]);
});

afterEach(() => {
  clearPermissions();
  vi.unstubAllGlobals();
});

/*
A biller bills from the shipment's documents, filed on the shipment. The
billing item's card shows them at its foot in the same rail as every card,
read only when the biller scrolls down to them.
*/
describe("DeskBillingItem's documents", () => {
  it("shows the shipment's documents in the rail once the biller scrolls to them", async () => {
    billingQueueService.getById.mockResolvedValue(item());
    const { container } = draw();

    await screen.findByRole("heading", { name: "Documents" });
    expect(documentService.getByResource).not.toHaveBeenCalled();

    act(() => FakeIntersectionObserver.show(container.querySelector(".dk-rec-docs-body")!));

    const pod = await screen.findByRole("link", { name: /Proof of delivery/ });
    expect(documentService.getByResource).toHaveBeenCalledWith("shipment", "shp_1", "", {
      includeDocumentType: "true",
    });
    expect(pod).toHaveAttribute("href", documentContentUrl("doc_pod", "view"));
    expect(screen.getByRole("link", { name: /Manage/ })).toHaveAttribute(
      "href",
      "/shipment-management/shipments?expanded=shp_1&panelType=edit&panelEntityId=shp_1",
    );
  });

  it("has no documents to show for an item with no shipment", async () => {
    billingQueueService.getById.mockResolvedValue(item({ shipmentId: "" }));
    draw();

    await screen.findByText("INV2610000079");
    expect(screen.queryByRole("heading", { name: "Documents" })).not.toBeInTheDocument();
  });
});
