import type { AssistantArtifact } from "@/types/assistant";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";
import { entityCardFrom } from "../artifact-payloads";
import { DeskRecordBody } from "../desk-bodies";
import { localMoment } from "../desk-record-view";

function card(payload: Record<string, unknown>): AssistantArtifact {
  return {
    id: "art_1",
    kind: "entity_card",
    status: "Ready",
    title: "Shipment SEED-SHP-008",
    payload,
    sourceToolCallId: "call_1",
    messageId: "msg_1",
    pinned: false,
    createdAt: 0,
    updatedAt: 0,
  } as unknown as AssistantArtifact;
}

const shipment = card({
  entity: "shipment",
  fields: [{ key: "proNumber", label: "Pro number", type: "text", value: "SEED-SHP-008" }],
  path: "/shipment-management/shipments?panelType=edit&panelEntityId=shp_1",
  view: {
    type: "shipment",
    status: "InTransit",
    subtitle: "Sunbelt Materials",
    from: { city: "Miami, FL", place: "Fleet Yard", when: "departed", at: "2026-10-01T12:50" },
    to: { city: "Chicago, IL", place: "Chicago DC", when: "scheduled", at: "2026-10-03T12:05" },
    progress: 0.5,
    facts: [
      { key: "weight", value: 18000 },
      { key: "bol", value: "BOL-2026-0008" },
      { key: "bogus", value: { nested: true } },
    ],
  },
});

describe("record views", () => {
  it("reads a shipment's view and drops facts it cannot show", () => {
    const parsed = entityCardFrom(shipment);
    expect(parsed.view?.from?.city).toBe("Miami, FL");
    expect(parsed.view?.facts.map((fact) => fact.key)).toEqual(["weight", "bol"]);
  });

  it("draws a shipment as its route and its facts, values as written", () => {
    render(
      <MemoryRouter>
        <DeskRecordBody artifact={shipment} />
      </MemoryRouter>,
    );
    expect(screen.getByText("SEED-SHP-008")).toBeInTheDocument();
    expect(screen.getByText("Sunbelt Materials")).toBeInTheDocument();
    expect(screen.getByText("Miami, FL")).toBeInTheDocument();
    expect(screen.getByText("Chicago, IL")).toBeInTheDocument();
    expect(screen.getByText("18,000 lb")).toBeInTheDocument();
    expect(screen.getByText("BOL-2026-0008")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Open shipment/u })).toBeInTheDocument();
  });

  it("says a time today as the time alone", () => {
    const now = new Date(2026, 9, 1, 8, 0);
    expect(localMoment("2026-10-01T12:50", now)).not.toContain("Oct");
    expect(localMoment("2026-10-03T12:05", now)).toContain("Oct");
  });
});
