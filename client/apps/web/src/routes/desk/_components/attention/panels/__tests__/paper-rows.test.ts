import type { ShipmentBillingRequirement } from "@trenova/shared/types/shipment";
import { describe, expect, it } from "vitest";
import { paperRows } from "../paper-rows";

function requirement(overrides: Partial<ShipmentBillingRequirement>): ShipmentBillingRequirement {
  return {
    documentTypeId: "dt_pod",
    documentTypeCode: "POD",
    documentTypeName: "Proof of delivery",
    satisfied: false,
    documentCount: 0,
    documentIds: [],
    ineligibleDocuments: [],
    ...overrides,
  };
}

/** The checklist is the shipment's own once it is read; until then it is what the Watchtower named. */
describe("the papers a load waits on", () => {
  it("reads each document billing asks for, and where it stands", () => {
    const rows = paperRows(
      [{ name: "Proof of delivery", onFile: false }],
      [
        requirement({
          documentTypeId: "dt_bol",
          documentTypeName: "Bill of lading",
          satisfied: true,
          documentCount: 1,
        }),
        requirement({ documentTypeId: "dt_pod", documentCount: 1 }),
        requirement({ documentTypeId: "dt_lump", documentTypeName: "Lumper receipt" }),
      ],
    );

    expect(rows).toEqual([
      { key: "dt_bol", name: "Bill of lading", documentTypeId: "dt_bol", state: "onFile" },
      { key: "dt_pod", name: "Proof of delivery", documentTypeId: "dt_pod", state: "waiting" },
      { key: "dt_lump", name: "Lumper receipt", documentTypeId: "dt_lump", state: "missing" },
    ]);
  });

  it("falls back to what the Watchtower named before the checklist is read, or when billing asks for nothing", () => {
    const named = [{ name: "Proof of delivery", onFile: false }];
    const expected = [
      {
        key: "Proof of delivery",
        name: "Proof of delivery",
        documentTypeId: null,
        state: "missing",
      },
    ];

    expect(paperRows(named, undefined)).toEqual(expected);
    expect(paperRows(named, [])).toEqual(expected);
  });
});
