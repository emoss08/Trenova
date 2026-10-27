import { describe, expect, it } from "vitest";
import type { CapabilityBundle } from "../role-api";
import {
  applyBundle,
  bundleState,
  bundlesIncluding,
  grantsToPermissions,
  removeBundles,
} from "../role-bundles";

const board: CapabilityBundle = {
  id: "dispatch.board",
  module: "Dispatch",
  label: "Dispatch board",
  description: "",
  grants: [
    {
      resource: "shipment",
      displayName: "Shipment",
      operations: ["read", "update"],
      dataScope: "organization",
    },
    { resource: "worker", displayName: "Worker", operations: ["read"], dataScope: "organization" },
  ],
};

const booking: CapabilityBundle = {
  id: "dispatch.shipments",
  module: "Dispatch",
  label: "Book shipments",
  description: "",
  grants: [
    {
      resource: "shipment",
      displayName: "Shipment",
      operations: ["read", "create"],
      dataScope: "own",
    },
  ],
};

describe("bundleState", () => {
  it("is full only when every operation of every grant is held", () => {
    expect(bundleState(board, grantsToPermissions(board.grants))).toBe("full");
    expect(
      bundleState(board, [
        { resource: "shipment", operations: ["read"], dataScope: "organization" },
      ]),
    ).toBe("partial");
    expect(bundleState(board, [])).toBe("none");
  });
});

describe("applyBundle", () => {
  it("adds operations to grants already there and keeps the wider scope", () => {
    const permissions = applyBundle(
      [{ resource: "shipment", operations: ["export"], dataScope: "own" }],
      board,
    );

    expect(permissions).toEqual([
      { resource: "shipment", operations: ["export", "read", "update"], dataScope: "organization" },
      { resource: "worker", operations: ["read"], dataScope: "organization" },
    ]);
  });
});

describe("removeBundles", () => {
  it("takes away the bundle's operations but keeps what another full bundle grants", () => {
    const permissions = applyBundle(applyBundle([], board), booking);

    const remaining = removeBundles(permissions, [board], [board, booking]);

    expect(remaining).toEqual([
      { resource: "shipment", operations: ["create", "read"], dataScope: "organization" },
    ]);
  });

  it("drops a resource left with no operations", () => {
    expect(removeBundles(grantsToPermissions(board.grants), [board], [board])).toEqual([]);
  });

  it("removes a bundle held inside another when both are removed together", () => {
    const viewing: CapabilityBundle = {
      ...board,
      id: "dispatch.view",
      label: "View the board",
      grants: [board.grants[1]],
    };
    const permissions = applyBundle([], board);

    expect(removeBundles(permissions, [viewing, board], [viewing, board, booking])).toEqual([]);
  });
});

describe("bundlesIncluding", () => {
  it("names the fully selected bundles that grant a resource", () => {
    const permissions = applyBundle([], board);

    expect(bundlesIncluding("shipment", [board, booking], permissions).map((b) => b.id)).toEqual([
      "dispatch.board",
    ]);
  });
});
