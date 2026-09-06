import type {
  ShipmentEventFieldsFragment,
  ShipmentEventType as GraphQLShipmentEventType,
} from "@trenova/graphql/generated/graphql";
import { describe, expect, expectTypeOf, it } from "vitest";
import {
  type ShipmentEvent,
  type ShipmentEventType,
  shipmentEventSchema,
  shipmentEventTypeSchema,
  shipmentEventTypenameSchema,
} from "../shipment-event";

describe("shipmentEventTypeSchema", () => {
  it("is exactly the generated GraphQL ShipmentEventType union", () => {
    expectTypeOf<ShipmentEventType>().toEqualTypeOf<GraphQLShipmentEventType>();
  });

  it("parses every tender and carrier event type the server emits", () => {
    const emitted: GraphQLShipmentEventType[] = [
      "CarrierAssigned",
      "CarrierUnassigned",
      "TenderOffered",
      "TenderAccepted",
      "TenderDeclined",
      "TenderExpired",
      "TenderWithdrawn",
      "TenderNeedsReview",
      "RoutingGuideExhausted",
      "TenderLateResponse",
      "TenderDeliveryFailed",
      "TenderEntrySkipped",
      "TenderEntryWarned",
    ];

    for (const value of emitted) {
      expect(shipmentEventTypeSchema.safeParse(value).success).toBe(true);
    }
  });

  it("stays strict — an unknown type is rejected rather than passed through", () => {
    expect(shipmentEventTypeSchema.safeParse("NotAShipmentEventType").success).toBe(false);
  });
});

describe("shipmentEventSchema", () => {
  it("discriminates on exactly the concrete __typename set the fragment can return", () => {
    expectTypeOf<ShipmentEvent["__typename"]>().toEqualTypeOf<
      ShipmentEventFieldsFragment["__typename"]
    >();
  });

  it("has one union member per concrete typename", () => {
    const options = shipmentEventSchema.options.map(
      (option) => option.shape.__typename.value as string,
    );
    expect([...options].sort()).toEqual([...shipmentEventTypenameSchema.options].sort());
    expect(new Set(options).size).toBe(options.length);
  });
});
