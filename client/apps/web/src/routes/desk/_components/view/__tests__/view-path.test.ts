import { withOpenWork } from "@/stores/desk-margin-store";
import { describe, expect, it } from "vitest";
import { viewPath } from "../view-path";

describe("viewPath", () => {
  it("opens a view under the Desk by its conversation and its name there", () => {
    expect(viewPath("athr_1", "shipments-at-risk")).toBe("/desk/v/athr_1/shipments-at-risk");
  });

  it("escapes a name that is not safe in a path", () => {
    expect(viewPath("athr_1", "a/b c")).toBe("/desk/v/athr_1/a%2Fb%20c");
  });
});

describe("open work", () => {
  it("keeps a view's own path beside the cases, newest first and each once", () => {
    const view = {
      kind: "view",
      id: "athr_1/risk",
      label: "Shipments at risk",
      path: "/desk/v/athr_1/risk",
    };
    const shipment = { kind: "shipment", id: "shp_1", label: "S-1" };

    expect(withOpenWork(withOpenWork([shipment], view), view)).toEqual([view, shipment]);
  });
});
