import { describe, expect, it } from "vitest";
import { recordPresence, viewPresence } from "../presence-scopes";

describe("presence scopes", () => {
  it("names a table's scope and join path the way the server builds them", () => {
    expect(viewPresence("customer")).toEqual({
      scope: "view:customer",
      joinPath: "/realtime/presence/customer/",
    });
  });

  it("names a record's scope by resource and id and encodes both in the path", () => {
    expect(recordPresence("shipment", "shp_1")).toEqual({
      scope: "record:shipment:shp_1",
      joinPath: "/realtime/presence/shipment/shp_1/",
    });
    expect(recordPresence("shipment", "a/b").joinPath).toBe("/realtime/presence/shipment/a%2Fb/");
  });
});
