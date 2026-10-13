import type { WatchtowerItem } from "@/lib/graphql/watchtower";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";
import { WatchtowerItemRow } from "../item-row";

const NOW = 1_800_000_000;

function item(overrides: Partial<WatchtowerItem> = {}): WatchtowerItem {
  return {
    __typename: "WatchtowerItem",
    id: "wt_1",
    sourceKind: "MissedPickup",
    sourceId: "stp_1",
    severity: "Warning",
    title: "Missed pickup on PRO-7",
    summary: "The pickup window closed and no arrival has been recorded.",
    subjectType: "Shipment",
    subjectId: "shp_1",
    eventKind: null,
    path: "/shipments/shp_1",
    occurredAt: NOW - 3600,
    resolvedAt: null,
    seen: true,
    kindLabel: "Missed pickup",
    version: 2,
    status: "Open",
    assigneeId: null,
    assignee: null,
    dueAt: NOW + 2 * 3600,
    overdue: false,
    snoozedUntil: null,
    acknowledgedAt: null,
    resolutionCode: null,
    resolutionNote: null,
    resolvedById: null,
    resolvedBy: null,
    sourceClearedAt: null,
    escalationLevel: 0,
    lastEscalatedAt: null,
    ...overrides,
  } as WatchtowerItem;
}

function renderRow(value: WatchtowerItem) {
  const handlers = { onHandOff: vi.fn(), onDismiss: vi.fn(), onOpen: vi.fn() };
  render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter>
        <ul>
          <WatchtowerItemRow item={value} unseen={false} now={NOW} {...handlers} />
        </ul>
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return handlers;
}

describe("WatchtowerItemRow", () => {
  it("shows an unowned open exception with its due time", () => {
    renderRow(item());

    expect(screen.getByText("Open")).toBeDefined();
    expect(screen.getByText("Unassigned")).toBeDefined();
    expect(screen.getByText("Due in 2h")).toBeDefined();
  });

  it("names the owner and how late an overdue exception is", () => {
    renderRow(
      item({
        status: "InProgress",
        assigneeId: "usr_1",
        assignee: { id: "usr_1", name: "Dana Ortiz" },
        dueAt: NOW - 30 * 60,
        overdue: true,
      }),
    );

    expect(screen.getByText("In progress")).toBeDefined();
    expect(screen.getByText("Dana Ortiz")).toBeDefined();
    expect(screen.getByText("Overdue 30m")).toBeDefined();
  });

  it("opens the exception from its title", () => {
    const value = item();
    const handlers = renderRow(value);

    fireEvent.click(screen.getByRole("button", { name: value.title }));

    expect(handlers.onOpen).toHaveBeenCalledWith(value);
  });
});
