import type { CaptureBatchRow } from "@/lib/graphql/capture";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { CaptureStacksPanel } from "../capture-panel";

const capture = vi.hoisted(() => ({ fetchCaptureBatches: vi.fn() }));

vi.mock("@/lib/graphql/capture", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/capture")>()),
  fetchCaptureBatches: capture.fetchCaptureBatches,
}));

function row(id: string, jobName: string): CaptureBatchRow {
  return {
    id,
    source: "Scan",
    status: "Filed",
    sourceName: "Front desk scanner",
    jobName,
    receivedPageCount: 3,
    createdAt: 1_700_000_000,
  } as unknown as CaptureBatchRow;
}

function renderPanel() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <CaptureStacksPanel kind="shipment" recordId="shp_1" />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("CaptureStacksPanel", () => {
  beforeEach(() => capture.fetchCaptureBatches.mockReset());

  it("lists the record's stacks, each opening in Intake", async () => {
    capture.fetchCaptureBatches.mockResolvedValue({
      batches: [row("cbat_1", "Delivery receipts")],
      endCursor: null,
      hasNextPage: false,
    });
    renderPanel();

    const link = await screen.findByRole("link", { name: "Delivery receipts" });
    expect(link).toHaveAttribute("href", "/intake?view=all&batch=cbat_1");
    expect(capture.fetchCaptureBatches).toHaveBeenCalledWith(
      expect.objectContaining({ targetType: "shipment", targetId: "shp_1" }),
      expect.anything(),
    );
  });

  it("shows nothing for a record with no stacks", async () => {
    capture.fetchCaptureBatches.mockResolvedValue({
      batches: [],
      endCursor: null,
      hasNextPage: false,
    });
    const { container } = renderPanel();

    await vi.waitFor(() => expect(capture.fetchCaptureBatches).toHaveBeenCalled());
    expect(container).toBeEmptyDOMElement();
  });
});
