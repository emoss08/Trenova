import type { AssistantArtifact } from "@/types/assistant";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { TableViewArtifact } from "../table-view-artifact";

const writeText = vi.fn();

beforeEach(() => {
  writeText.mockResolvedValue(undefined);
  Object.defineProperty(navigator, "clipboard", {
    value: { writeText },
    configurable: true,
  });
});

afterEach(() => {
  cleanup();
  writeText.mockReset();
});

function tableArtifact(rowCount: number, extra: Record<string, unknown> = {}): AssistantArtifact {
  const rows = Array.from({ length: rowCount }, (_, index) => ({
    id: `shp_${index}`,
    proNumber: `PRO-${index + 1}`,
    status: index % 2 === 0 ? "InTransit" : "Delivered",
    weight: (index + 1) * 100,
  }));

  return {
    id: "aart_1",
    threadId: "athr_1",
    messageId: "",
    runId: "",
    proposalId: "",
    planId: "",
    kind: "table_view",
    status: "Ready",
    title: "Shipments",
    payload: {
      display: 1,
      tool: "list_shipments",
      entity: "shipments",
      recordEntity: "shipment",
      columns: [
        { key: "proNumber", label: "PRO number", type: "text" },
        { key: "status", label: "Status", type: "status" },
        { key: "weight", label: "Weight", type: "number" },
      ],
      rows,
      rowCount,
      searchedFor: ["status is InTransit"],
      ...extra,
    },
    sourceToolCallId: "call_1",
    pinned: false,
    lineageId: "",
    lineageSeq: 1,
    createdAt: 1,
    updatedAt: 1,
  } as AssistantArtifact;
}

function renderTable(artifact: AssistantArtifact) {
  return render(
    <MemoryRouter>
      <TableViewArtifact artifact={artifact} />
    </MemoryRouter>,
  );
}

describe("a table artifact", () => {
  it("says how many rows and what was searched for", () => {
    renderTable(tableArtifact(3));

    expect(screen.getByText(/3 rows · searched for: status is InTransit/)).toBeInTheDocument();
  });

  it("copies the table as CSV", async () => {
    renderTable(tableArtifact(2));

    await userEvent.click(screen.getByRole("button", { name: "Copy as CSV" }));

    expect(writeText).toHaveBeenCalledWith(
      "PRO number,Status,Weight\r\nPRO-1,In transit,100\r\nPRO-2,Delivered,200",
    );
    expect(await screen.findByRole("button", { name: "Copied" })).toBeInTheDocument();
  });

  // Eight rows are read at a glance; a longer table earns a filter, and a
  // short one keeps the space for its rows.
  it("offers a filter only once there is enough to filter", () => {
    const short = renderTable(tableArtifact(8));
    expect(screen.queryByRole("searchbox", { name: "Filter rows" })).toBeNull();
    short.unmount();

    renderTable(tableArtifact(9));
    expect(screen.getByRole("searchbox", { name: "Filter rows" })).toBeInTheDocument();
  });

  it("narrows the rows to what matches, counts them, and copies only those", async () => {
    renderTable(tableArtifact(10));

    await userEvent.type(screen.getByRole("searchbox", { name: "Filter rows" }), "delivered");

    const body = screen.getAllByRole("rowgroup")[1];
    expect(within(body).getAllByRole("row")).toHaveLength(5);
    expect(screen.getByText(/5 of 10 rows/)).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Copy as CSV" }));
    expect(writeText.mock.calls[0][0].split("\r\n")).toHaveLength(6);
  });

  it("links each row to its record through the registry", () => {
    renderTable(tableArtifact(1));

    expect(screen.getByRole("link", { name: "PRO-1" })).toHaveAttribute(
      "href",
      expect.stringContaining("shp_0"),
    );
  });
});
