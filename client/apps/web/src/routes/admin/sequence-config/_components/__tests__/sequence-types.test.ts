import { sequenceConfigDocumentSchema, sequenceTypes } from "@/types/sequence-config";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { defaultConfigForType, sidebarGroups } from "../sequence-config-constants";

const TENANT_DOMAIN = resolve(
  __dirname,
  "../../../../../../../../../services/tms/internal/modules/identity/domain/tenant",
);

/**
 * The sequence types the server sends and requires on save, read from its
 * catalog: `requiredSequenceTypes` names the constants, and enums.go gives each
 * constant its wire value.
 */
function serverRequiredSequenceTypes(): string[] {
  const enums = readFileSync(resolve(TENANT_DOMAIN, "enums.go"), "utf8");
  const wireValues = new Map(
    Array.from(enums.matchAll(/(SequenceType\w+)\s*=\s*SequenceType\("([a-z_]+)"\)/g), (m) => [
      m[1],
      m[2],
    ]),
  );

  const catalog = readFileSync(resolve(TENANT_DOMAIN, "sequencecatalog.go"), "utf8");
  const block = /var requiredSequenceTypes = \[\]SequenceType\{([^}]*)\}/.exec(catalog)?.[1];
  if (!block) {
    throw new Error("requiredSequenceTypes not found in sequencecatalog.go");
  }

  return Array.from(block.matchAll(/SequenceType\w+/g), ([name]) => {
    const value = wireValues.get(name);
    if (!value) {
      throw new Error(`${name} has no wire value in enums.go`);
    }
    return value;
  });
}

// The page parses the server's document against this list and requires exactly
// one config per type, so a type the server adds without the client breaks the
// whole page rather than one row of it.
describe("sequence types", () => {
  it("are exactly the ones the server requires", () => {
    expect([...sequenceTypes].sort()).toEqual(serverRequiredSequenceTypes().sort());
  });

  it("each appear in exactly one group of the list", () => {
    const listed = sidebarGroups.flatMap((group) => group.items);
    expect([...listed].sort()).toEqual([...sequenceTypes].sort());
  });

  it("accept a document holding one config for every type", () => {
    const base = {
      id: "",
      organizationId: "",
      businessUnitId: "",
      version: 1,
      createdAt: 1,
      updatedAt: 1,
    };
    const document = {
      organizationId: "org_1",
      businessUnitId: "bu_1",
      configs: sequenceTypes.map((type) => defaultConfigForType(type, base)),
    };

    expect(sequenceConfigDocumentSchema.safeParse(document).success).toBe(true);
  });
});
