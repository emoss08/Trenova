import type { AssistantArtifact } from "@/types/assistant";
import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it } from "vitest";
import { entityCardFrom, navigationFrom } from "../artifact-payloads";
import { EntityCardArtifact } from "../entity-card-artifact";
import { NavigationArtifact } from "../navigation-artifact";

afterEach(cleanup);

function artifact(overrides: Partial<AssistantArtifact>): AssistantArtifact {
  return {
    id: "aart_1",
    threadId: "athr_1",
    messageId: null,
    runId: null,
    proposalId: null,
    planId: null,
    kind: "navigation",
    status: "Ready",
    title: "Rate matrices",
    payload: {},
    sourceToolCallId: "call_1",
    pinned: false,
    createdAt: 1,
    updatedAt: 1,
    ...overrides,
  } as AssistantArtifact;
}

describe("a page the assistant opened", () => {
  it("reads as the page, where it sits, and the way back to it", () => {
    render(
      <MemoryRouter>
        <NavigationArtifact
          artifact={artifact({
            payload: {
              path: "/billing/configuration-files/rate-matrices",
              name: "Rate matrices",
              location: "Billing › Configuration files › Rate matrices",
            },
          })}
        />
      </MemoryRouter>,
    );

    expect(screen.getByText("Rate matrices")).toBeInTheDocument();
    expect(screen.getByText("Billing › Configuration files › Rate matrices")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Open/ })).toHaveAttribute(
      "href",
      "/billing/configuration-files/rate-matrices",
    );
  });

  it.each(["", "https://example.com", "//example.com"])(
    "offers no way out of the app for %j",
    (path) => {
      expect(navigationFrom(artifact({ payload: { path } }))).toBeNull();
    },
  );
});

describe("a record card", () => {
  it("links to the record where the app opens it", () => {
    const card = artifact({
      kind: "entity_card",
      title: "Shipment PRO-1",
      payload: {
        entity: "shipment",
        record: { id: "shp_1", proNumber: "PRO-1" },
        path: "/shipment-management/shipments?panelEntityId=shp_1&panelType=edit",
      },
    });

    render(
      <MemoryRouter>
        <EntityCardArtifact artifact={card} />
      </MemoryRouter>,
    );

    expect(screen.getByRole("link", { name: /Open/ })).toHaveAttribute(
      "href",
      "/shipment-management/shipments?panelEntityId=shp_1&panelType=edit",
    );
  });

  // A kind of record with no page of its own has no link, rather than one
  // that leads nowhere.
  it("offers no link when the record has no page", () => {
    const card = artifact({
      kind: "entity_card",
      payload: { entity: "detention_occurrence", record: { id: "det_1" } },
    });

    expect(entityCardFrom(card).path).toBe("");
    render(
      <MemoryRouter>
        <EntityCardArtifact artifact={card} />
      </MemoryRouter>,
    );
    expect(screen.queryByRole("link", { name: /Open/ })).toBeNull();
  });

  it("drops a link that would leave the app", () => {
    const card = artifact({
      kind: "entity_card",
      payload: { entity: "shipment", record: { id: "shp_1" }, path: "https://example.com" },
    });

    expect(entityCardFrom(card).path).toBe("");
  });
});
