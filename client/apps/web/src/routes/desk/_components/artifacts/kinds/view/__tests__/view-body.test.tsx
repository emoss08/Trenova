import type { AssistantArtifact } from "@/types/assistant";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";
import { composedViewFrom } from "../artifact-payloads";
import { asSentence, DeskViewBody } from "../desk-bodies";

function view(payload: Record<string, unknown>): AssistantArtifact {
  return {
    id: "art_1",
    kind: "table_view",
    status: "Ready",
    title: "Shipments where status equals InTransit",
    payload,
    sourceToolCallId: "call_1",
    messageId: "msg_1",
    pinned: false,
    createdAt: 0,
    updatedAt: 0,
  } as unknown as AssistantArtifact;
}

const base = {
  display: 1,
  entity: "shipments",
  path: "/shipment-management/shipments?fieldFilters=%5B%5D",
  explanation: "shipments where status equals InTransit",
  terms: ["status equals InTransit"],
  filterCount: 1,
  unresolved: [{ phrase: "near Dallas", reason: "There is no location filter" }],
};

const counted = view({
  ...base,
  rowCount: 100,
  countCapped: true,
  recordEntity: "shipment",
  columns: [
    { key: "proNumber", label: "Pro number", type: "text" },
    { key: "customer", label: "Customer", type: "text" },
    { key: "weight", label: "Weight", type: "number" },
    { key: "status", label: "Status", type: "status" },
  ],
  rows: [
    { id: "shp_1", proNumber: "P-1001", customer: "Acme", weight: 1200, status: "InTransit" },
    { id: "shp_2", proNumber: "P-1002", customer: "Globex", weight: 900, status: "InTransit" },
  ],
});

describe("composed views", () => {
  it("reads the count and a preview down to the card's three places", () => {
    const parsed = composedViewFrom(counted);
    expect(parsed?.count).toBe(100);
    expect(parsed?.countCapped).toBe(true);
    // The first column names the row, the status closes it, and the next
    // column sits between them.
    expect(parsed?.columns.map((column) => column.key)).toEqual([
      "proNumber",
      "customer",
      "status",
    ]);
    expect(parsed?.preview).toHaveLength(2);
    expect(parsed?.preview[0]?.values.proNumber).toBe("P-1001");
  });

  it("leaves the count and preview out of a view stored before they were", () => {
    const parsed = composedViewFrom(view(base));
    expect(parsed?.count).toBeNull();
    expect(parsed?.countCapped).toBe(false);
    expect(parsed?.columns).toEqual([]);
    expect(parsed?.preview).toEqual([]);
  });

  it("is not a view without a page to open", () => {
    expect(composedViewFrom(view({ ...base, path: "" }))).toBeNull();
  });

  it("draws the explanation, the bar, the rows, what was left out and the way in", () => {
    const { container } = render(
      <MemoryRouter>
        <DeskViewBody artifact={counted} />
      </MemoryRouter>,
    );
    expect(container.querySelector(".dk-ax-term")?.textContent).toBe("status equals InTransit");
    expect(container.querySelector(".dk-ax-vbar")?.textContent).toBe(
      "Shipments1 filter100+ results",
    );
    const rows = container.querySelectorAll(".dk-ax-vrow");
    expect(rows).toHaveLength(2);
    expect(rows[0]?.querySelector(".dk-ax-id")?.textContent).toBe("P-1001");
    expect(rows[0]?.querySelector(".dk-ax-pill")).not.toBeNull();
    expect(container.querySelector(".dk-ax-warn")?.textContent).toBe(
      "Left out “near Dallas”. There is no location filter.",
    );
    const open = screen.getByRole("link", { name: /Open in Shipments/ });
    expect(open.getAttribute("href")).toBe(base.path);
  });

  it("draws an older view with no count and no preview rows", () => {
    const { container } = render(
      <MemoryRouter>
        <DeskViewBody artifact={view(base)} />
      </MemoryRouter>,
    );
    expect(container.querySelector(".dk-ax-vbar b")).toBeNull();
    expect(container.querySelectorAll(".dk-ax-vrow")).toHaveLength(0);
  });

  it("closes a reason with one full stop", () => {
    expect(asSentence("no location filter")).toBe("no location filter.");
    expect(asSentence("Only 8 filters can be applied at once.")).toBe(
      "Only 8 filters can be applied at once.",
    );
  });
});
