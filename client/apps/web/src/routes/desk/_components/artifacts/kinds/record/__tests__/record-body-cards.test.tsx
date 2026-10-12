import type { AssistantArtifact } from "@/types/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";
import { deskArtKind, deskArtKindName } from "../../../desk-art-kinds";
import { DeskRecordBody } from "../desk-record-body";

vi.mock("../../../cards/formula-template/desk-formula-template", () => ({
  DeskFormulaTemplate: () => null,
  DeskFormulaTemplateArtifact: ({ recordId }: { recordId: string }) => (
    <p>Formula template {recordId}</p>
  ),
}));
vi.mock("../../../cards/shipment/desk-shipment", () => ({
  DeskShipment: () => null,
  DeskShipmentArtifact: ({ recordId }: { recordId: string }) => <p>Shipment {recordId}</p>,
}));
vi.mock("../../../cards/customer/desk-customer", () => ({
  DeskCustomer: () => null,
  DeskCustomerArtifact: ({ recordId }: { recordId: string }) => <p>Customer {recordId}</p>,
}));
vi.mock("../../../cards/rate-agreement/desk-rate-agreement", () => ({
  DeskRateAgreement: () => null,
  DeskRateAgreementArtifact: ({ recordId }: { recordId: string }) => (
    <p>Rate agreement {recordId}</p>
  ),
}));
vi.mock("../../../cards/trailer/desk-trailer", () => ({
  DeskTrailer: () => null,
  DeskTrailerArtifact: ({ recordId }: { recordId: string }) => <p>Trailer {recordId}</p>,
}));
vi.mock("../../../cards/location/desk-location", () => ({
  DeskLocation: () => null,
  DeskLocationArtifact: ({ recordId }: { recordId: string }) => <p>Location {recordId}</p>,
}));
vi.mock("../../../cards/worker/desk-worker", () => ({
  DeskWorker: () => null,
  DeskWorkerArtifact: ({ recordId }: { recordId: string }) => <p>Worker {recordId}</p>,
}));
vi.mock("../../../cards/tractor/desk-tractor", () => ({
  DeskTractor: () => null,
  DeskTractorArtifact: ({ recordId }: { recordId: string }) => <p>Tractor {recordId}</p>,
}));
vi.mock("../../../cards/billing-queue-item/desk-billing-record", () => ({
  DeskBillingRecord: () => null,
  DeskBillingArtifact: ({ recordId, fallback }: { recordId: string; fallback: ReactNode }) => (
    <div>
      <p>Billing item {recordId}</p>
      <div data-testid="fallback">{fallback}</div>
    </div>
  ),
}));

/** A record card as the server keeps a get_ tool's result: the entity, its fields and its id. */
function card(entity: string, payload: Record<string, unknown> = {}): AssistantArtifact {
  return {
    id: "art_1",
    kind: "entity_card",
    status: "Ready",
    title: "A record",
    payload: {
      display: 1,
      entity,
      fields: [{ key: "name", label: "Name", type: "text", value: "Reefer per mile" }],
      ...payload,
    },
    sourceToolCallId: "call_1",
    messageId: "msg_1",
    pinned: false,
    createdAt: 0,
    updatedAt: 0,
  } as unknown as AssistantArtifact;
}

function draw(artifact: AssistantArtifact) {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter>
        <DeskRecordBody artifact={artifact} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

const t = (message: string) => message;

/*
A record that has a card of its own is drawn as that card when the agent looks
it up, not as a list of its fields: a formula template as its formula, a
billing item as the item a biller reviews.
*/
describe("a looked-up record with a card of its own", () => {
  it("draws a formula template as the formula template card", async () => {
    draw(card("formula_template", { recordId: "ft_1" }));

    expect(await screen.findByText("Formula template ft_1")).toBeInTheDocument();
  });

  it("draws a billing item as its card, with the server's view while the item is read", async () => {
    draw(
      card("billing_queue_item", {
        recordId: "bqi_1",
        view: { type: "billing_queue_item", status: "ReadyForReview", facts: [] },
      }),
    );

    expect(await screen.findByText("Billing item bqi_1")).toBeInTheDocument();
    expect(screen.getByTestId("fallback")).not.toBeEmptyDOMElement();
  });

  it("holds a billing item with the card's outline when the server sent no view", async () => {
    draw(card("billing_queue_item", { recordId: "bqi_1" }));

    expect(await screen.findByText("Billing item bqi_1")).toBeInTheDocument();
    expect(screen.getByTestId("fallback").querySelector(".dk-rec-skel")).not.toBeNull();
  });

  it("draws a record without a card as its fields", () => {
    draw(card("commodity", { recordId: "com_1" }));

    expect(screen.getByText("Reefer per mile")).toBeInTheDocument();
    expect(screen.queryByText(/Formula template|Billing item/)).not.toBeInTheDocument();
  });

  it("draws a formula template card with no record id as its fields", () => {
    draw(card("formula_template"));

    expect(screen.getByText("Reefer per mile")).toBeInTheDocument();
  });

  it("names a formula template card by its kind", () => {
    const artifact = card("formula_template", { recordId: "ft_1" });

    expect(deskArtKind(artifact)).toBe("formula");
    expect(deskArtKindName("formula", t as never)).toBe("Formula template");
    expect(deskArtKind(card("billing_queue_item"))).toBe("bill");
    expect(deskArtKind(card("commodity"))).toBe("record");
  });

  it("draws a shipment the agent looked up as the shipment card, with its own mark", async () => {
    const artifact = card("shipment", { recordId: "shp_1", tool: "get_shipment" });

    draw(artifact);

    expect(await screen.findByText("Shipment shp_1")).toBeInTheDocument();
    expect(deskArtKind(artifact)).toBe("shipment");
    expect(deskArtKindName("shipment", t as never)).toBe("Shipment");
  });

  it("draws a customer the agent looked up as the customer card, with its own mark", async () => {
    const artifact = card("customer", { recordId: "cus_1", tool: "get_customer" });

    draw(artifact);

    expect(await screen.findByText("Customer cus_1")).toBeInTheDocument();
    expect(deskArtKind(artifact)).toBe("customer");
    expect(deskArtKindName("customer", t as never)).toBe("Customer");
  });

  it("draws a rate agreement the agent looked up as the agreement card, with its own mark", async () => {
    const artifact = card("rate_agreement", { recordId: "ra_1", tool: "get_rate_agreement" });

    draw(artifact);

    expect(await screen.findByText("Rate agreement ra_1")).toBeInTheDocument();
    expect(deskArtKind(artifact)).toBe("agreement");
    expect(deskArtKindName("agreement", t as never)).toBe("Rate agreement");
  });

  it("draws a tractor the agent looked up as the tractor card, with its own mark", async () => {
    const artifact = card("tractor", { recordId: "tr_1", tool: "get_tractor" });

    draw(artifact);

    expect(await screen.findByText("Tractor tr_1")).toBeInTheDocument();
    expect(deskArtKind(artifact)).toBe("tractor");
    expect(deskArtKindName("tractor", t as never)).toBe("Tractor");
  });

  it("draws a trailer the agent looked up as the trailer card, with its own mark", async () => {
    const artifact = card("trailer", { recordId: "trl_1", tool: "get_trailer" });

    draw(artifact);

    expect(await screen.findByText("Trailer trl_1")).toBeInTheDocument();
    expect(deskArtKind(artifact)).toBe("trailer");
    expect(deskArtKindName("trailer", t as never)).toBe("Trailer");
  });

  it("draws a worker the agent looked up as the worker card, with its own mark", async () => {
    const artifact = card("worker", { recordId: "wrk_1", tool: "get_worker" });

    draw(artifact);

    expect(await screen.findByText("Worker wrk_1")).toBeInTheDocument();
    expect(deskArtKind(artifact)).toBe("worker");
    expect(deskArtKindName("worker", t as never)).toBe("Worker");
  });

  it("draws a location the agent looked up as the location card, with its own mark", async () => {
    const artifact = card("location", { recordId: "loc_1", tool: "get_location" });

    draw(artifact);

    expect(await screen.findByText("Location loc_1")).toBeInTheDocument();
    expect(deskArtKind(artifact)).toBe("location");
    expect(deskArtKindName("location", t as never)).toBe("Location");
  });

  it("reads only the record kinds it names, never a name every object has", () => {
    expect(deskArtKind(card("constructor"))).toBe("record");
    expect(deskArtKind(card("toString"))).toBe("record");
  });
});
