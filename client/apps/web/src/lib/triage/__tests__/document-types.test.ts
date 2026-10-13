import { describe, expect, it } from "vitest";
import { documentTypeName, documentTypeNamed } from "../document-types";

const RATECON = { id: "dt_1", label: "RATECON", description: "Rate confirmation", meta: null };
const POD = { id: "dt_2", label: "POD", description: null, meta: null };

describe("documentTypeName", () => {
  it("reads the name from the description and falls back to the code", () => {
    expect(documentTypeName(RATECON)).toBe("Rate confirmation");
    expect(documentTypeName(POD)).toBe("POD");
    expect(documentTypeName({ label: "BOL", description: "  " })).toBe("BOL");
  });
});

describe("documentTypeNamed", () => {
  it("finds a type by its name or its code, ignoring case", () => {
    expect(documentTypeNamed([RATECON, POD], "rate Confirmation")).toBe(RATECON);
    expect(documentTypeNamed([RATECON, POD], "ratecon")).toBe(RATECON);
    expect(documentTypeNamed([RATECON, POD], "pod")).toBe(POD);
    expect(documentTypeNamed([RATECON, POD], "Invoice")).toBeNull();
    expect(documentTypeNamed([RATECON, POD], " ")).toBeNull();
  });
});
