import { describe, expect, it } from "vitest";
import { impliedOperations } from "../implied-operations";

const definitions = [
  { operation: "read", displayName: "Read", description: "", implies: [] },
  { operation: "update", displayName: "Update", description: "", implies: ["read"] },
  { operation: "approve", displayName: "Approve", description: "", implies: ["read", "update"] },
  { operation: "manage", displayName: "Manage", description: "", implies: null },
];

describe("impliedOperations", () => {
  it("names every operation a granted one brings with it, and which grants bring it", () => {
    const implied = impliedOperations(["approve", "update"], definitions);

    expect(Object.fromEntries(implied)).toEqual({ read: ["update", "approve"] });
  });

  it("leaves out operations that are granted outright", () => {
    const implied = impliedOperations(["approve", "update", "read"], definitions);

    expect(implied.size).toBe(0);
  });

  it("implies nothing for an operation the server sent no implications for", () => {
    expect(impliedOperations(["manage"], definitions).size).toBe(0);
  });

  it("implies nothing from an operation that is not granted", () => {
    expect(impliedOperations([], definitions).size).toBe(0);
  });
});
