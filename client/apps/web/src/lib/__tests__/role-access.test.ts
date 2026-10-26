import type { ResourceDefinition } from "@/lib/role-api";
import type { AddPermission } from "@trenova/shared/types/role";
import { describe, expect, it } from "vitest";
import {
  presetLevels,
  resourceAccess,
  setAccessLevel,
  setDataScope,
  toggleOperation,
} from "../role-access";

// Written from the Go registry: invoice disputes carry renamed operations,
// each of which implies the read; a reference resource offers only a read.
const invoiceDispute: ResourceDefinition = {
  resource: "invoice_dispute",
  displayName: "Invoice Dispute",
  description: "Dispute cases raised against posted invoices",
  category: "Billing",
  defaultSensitivity: "restricted",
  operations: [
    { operation: "read", displayName: "View", description: "View invoice disputes", implies: [] },
    { operation: "create", displayName: "Open", description: "Open a dispute", implies: ["read"] },
    { operation: "approve", displayName: "Resolve", description: "Resolve", implies: ["read"] },
    { operation: "cancel", displayName: "Withdraw", description: "Withdraw", implies: null },
  ],
};

const locationCategory: ResourceDefinition = {
  resource: "location_category",
  displayName: "Location Category",
  description: "Location categories",
  category: "Dispatch",
  defaultSensitivity: "internal",
  operations: [{ operation: "read", displayName: "View", description: "View", implies: [] }],
};

const signal: ResourceDefinition = {
  resource: "service_signal",
  displayName: "Service Signal",
  description: "A resource with no read",
  category: "Operations",
  defaultSensitivity: "internal",
  operations: [{ operation: "resolve", displayName: "Resolve", description: "Resolve" }],
};

describe("resourceAccess", () => {
  it("reads as no access with no operations", () => {
    expect(resourceAccess(invoiceDispute, [])).toEqual({ level: "none", reached: 0, total: 4 });
  });

  it("reads as view only with the read alone", () => {
    expect(resourceAccess(invoiceDispute, ["read"])).toEqual({
      level: "view",
      reached: 1,
      total: 4,
    });
  });

  it("counts an implied read toward what the grant reaches", () => {
    expect(resourceAccess(invoiceDispute, ["create"])).toEqual({
      level: "custom",
      reached: 2,
      total: 4,
    });
  });

  it("reads as full access when every operation is held or implied", () => {
    expect(resourceAccess(invoiceDispute, ["create", "approve", "cancel"])).toEqual({
      level: "full",
      reached: 4,
      total: 4,
    });
  });
});

describe("presetLevels", () => {
  it("offers view only on a resource that does more than read", () => {
    expect(presetLevels(invoiceDispute)).toEqual(["none", "view", "full"]);
  });

  it("offers no separate view only on a read-only resource or one with no read", () => {
    expect(presetLevels(locationCategory)).toEqual(["none", "full"]);
    expect(presetLevels(signal)).toEqual(["none", "full"]);
  });
});

describe("setAccessLevel", () => {
  const held: AddPermission[] = [
    { resource: "customer", operations: ["read"], dataScope: "own" },
    { resource: "invoice_dispute", operations: ["read", "create"], dataScope: "team" },
  ];

  it("replaces a held resource's operations and keeps its scope and place", () => {
    expect(setAccessLevel(held, [invoiceDispute], "full")).toEqual([
      { resource: "customer", operations: ["read"], dataScope: "own" },
      {
        resource: "invoice_dispute",
        operations: ["read", "create", "approve", "cancel"],
        dataScope: "team",
      },
    ]);
  });

  it("adds a resource the role did not hold at organization scope", () => {
    expect(setAccessLevel(held, [locationCategory], "full")).toEqual([
      ...held,
      { resource: "location_category", operations: ["read"], dataScope: "organization" },
    ]);
  });

  it("drops a resource set to no access", () => {
    expect(setAccessLevel(held, [invoiceDispute], "none")).toEqual([held[0]]);
  });

  it("leaves a resource with no read alone when the level is view only", () => {
    expect(setAccessLevel(held, [invoiceDispute, signal], "view")).toEqual([
      held[0],
      { resource: "invoice_dispute", operations: ["read"], dataScope: "team" },
    ]);
  });
});

describe("toggleOperation", () => {
  it("adds an operation to a held resource", () => {
    expect(
      toggleOperation(
        [{ resource: "invoice_dispute", operations: ["read"], dataScope: "own" }],
        "invoice_dispute",
        "approve",
      ),
    ).toEqual([{ resource: "invoice_dispute", operations: ["read", "approve"], dataScope: "own" }]);
  });

  it("drops the resource when its last operation is removed", () => {
    expect(
      toggleOperation(
        [{ resource: "invoice_dispute", operations: ["read"], dataScope: "own" }],
        "invoice_dispute",
        "read",
      ),
    ).toEqual([]);
  });

  it("grants a resource the role did not hold", () => {
    expect(toggleOperation([], "invoice_dispute", "create")).toEqual([
      { resource: "invoice_dispute", operations: ["create"], dataScope: "organization" },
    ]);
  });
});

describe("setDataScope", () => {
  it("changes only the named resource's scope", () => {
    expect(
      setDataScope(
        [
          { resource: "customer", operations: ["read"], dataScope: "organization" },
          { resource: "invoice_dispute", operations: ["read"], dataScope: "organization" },
        ],
        "customer",
        "own",
      ),
    ).toEqual([
      { resource: "customer", operations: ["read"], dataScope: "own" },
      { resource: "invoice_dispute", operations: ["read"], dataScope: "organization" },
    ]);
  });
});
