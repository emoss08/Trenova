import { readFileSync } from "node:fs";
import { join } from "node:path";
import { documentProcessingProfileSchema } from "@trenova/shared/types/document";
import { describe, expect, it } from "vitest";

// The contract is the Go enum; every value the server can send must parse,
// or one document with a new profile fails the whole list it arrives in.
const GO_ENUM = join(
  import.meta.dirname,
  "..",
  "..",
  "..",
  "..",
  "..",
  "..",
  "services",
  "tms",
  "internal",
  "core",
  "domain",
  "document",
  "document.go",
);
const serverProfiles = [
  ...readFileSync(GO_ENUM, "utf8").matchAll(/ProcessingProfile = "([a-z_]+)"/g),
].map((match) => match[1]);

describe("documentProcessingProfileSchema", () => {
  it("finds the server's processing profiles", () => {
    expect(serverProfiles.length).toBeGreaterThanOrEqual(5);
  });

  it.each(serverProfiles)("accepts %s, which the server sends", (profile) => {
    expect(documentProcessingProfileSchema.safeParse(profile).success).toBe(true);
  });

  it("lists exactly the server's profiles", () => {
    expect([...documentProcessingProfileSchema.options].sort()).toEqual([...serverProfiles].sort());
  });
});
