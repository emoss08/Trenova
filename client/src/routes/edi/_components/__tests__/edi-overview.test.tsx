import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { EDIOverview } from "../overview/edi-overview";
import * as summaryHook from "../overview/use-edi-summary";

function renderOverview() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <EDIOverview />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

const summary = {
  ediSummary: {
    deliveryStatusCounts: [
      { status: "Sent", count: 12 },
      { status: "Failed", count: 2 },
      { status: "DeadLettered", count: 3 },
    ],
    ackStatusCounts: [
      { status: "Pending", count: 4 },
      { status: "Rejected", count: 1 },
    ],
    inboundFileStatusCounts: [
      { status: "Quarantined", count: 5 },
      { status: "PartiallyProcessed", count: 2 },
    ],
    inboundTransferStatusCounts: [
      { status: "MappingRequired", count: 7 },
      { status: "PendingApproval", count: 1 },
    ],
    overdueAckCount: 6,
    attentionItems: [
      {
        kind: "Message" as const,
        id: "edimsg_1",
        partnerId: "edip_1",
        partnerName: "Acme Carrier",
        partnerCode: "ACME",
        reference: "204 0042",
        error: "connection refused",
        occurredAt: 1_767_193_200,
      },
      {
        kind: "InboundFile" as const,
        id: "ediinf_1",
        partnerId: null,
        partnerName: null,
        partnerCode: null,
        reference: "bad-file.edi",
        error: "inbound file does not contain any X12 segments",
        occurredAt: 1_767_193_100,
      },
    ],
  },
};

describe("EDIOverview", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it("renders headline counts from the summary", () => {
    vi.spyOn(summaryHook, "useEDISummary").mockReturnValue({
      data: summary,
      isLoading: false,
      isError: false,
    } as ReturnType<typeof summaryHook.useEDISummary>);

    renderOverview();

    expect(screen.getByText("Dead-lettered messages")).toBeDefined();
    expect(screen.getByText("3")).toBeDefined();
    expect(screen.getByText("Quarantined files")).toBeDefined();
    expect(screen.getByText("5")).toBeDefined();
    expect(screen.getByText("Stuck transfers")).toBeDefined();
    expect(screen.getByText("7")).toBeDefined();
    expect(screen.getByText("Overdue acknowledgments")).toBeDefined();
    expect(screen.getByText("6")).toBeDefined();
  });

  it("renders attention items with deep links", () => {
    vi.spyOn(summaryHook, "useEDISummary").mockReturnValue({
      data: summary,
      isLoading: false,
      isError: false,
    } as ReturnType<typeof summaryHook.useEDISummary>);

    renderOverview();

    const messageRow = screen.getByText("204 0042").closest("a");
    expect(messageRow?.getAttribute("href")).toBe(
      "/edi/messages?panelType=edit&panelEntityId=edimsg_1",
    );
    const fileRow = screen.getByText("bad-file.edi").closest("a");
    expect(fileRow?.getAttribute("href")).toBe(
      "/edi/inbound-files?panelType=edit&panelEntityId=ediinf_1",
    );
  });

  it("shows an empty state when there are no failures", () => {
    vi.spyOn(summaryHook, "useEDISummary").mockReturnValue({
      data: {
        ediSummary: {
          ...summary.ediSummary,
          attentionItems: [],
        },
      },
      isLoading: false,
      isError: false,
    } as unknown as ReturnType<typeof summaryHook.useEDISummary>);

    renderOverview();

    expect(screen.getByText(/pipeline is healthy/i)).toBeDefined();
  });
});
