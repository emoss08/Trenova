import { describe, expect, it } from "vitest";
import { formulaImportsSchema, formulaPreviewResponseSchema } from "../formula-template";

describe("formulaImportsSchema", () => {
  it("reads each import's path with the library and version it is pinned to", () => {
    expect(
      formulaImportsSchema.parse([
        { path: "org.accessorials", libraryId: "fl_01", versionNumber: 3 },
        { path: "org.fuel", libraryId: "fl_02", versionNumber: 1 },
      ]),
    ).toEqual([
      { path: "org.accessorials", libraryId: "fl_01", versionNumber: 3 },
      { path: "org.fuel", libraryId: "fl_02", versionNumber: 1 },
    ]);
  });

  it("reads an absent or null list as no imports", () => {
    expect(formulaImportsSchema.parse(undefined)).toEqual([]);
    expect(formulaImportsSchema.parse(null)).toEqual([]);
  });

  it("refuses an import without the version it is pinned to", () => {
    expect(() => formulaImportsSchema.parse([{ path: "org.fuel", libraryId: "fl_02" }])).toThrow();
  });
});

describe("formulaPreviewResponseSchema", () => {
  it("reads the server's error as text in the reader's language", () => {
    const preview = formulaPreviewResponseSchema.parse({
      diagnostics: [],
      error: { code: "eval", message: "Division by zero", range: { from: 4, to: 9 } },
    });
    expect(preview.error).toEqual({
      code: "eval",
      message: "Division by zero",
      range: { from: 4, to: 9 },
    });
    expect(preview).toMatchObject({ bill: null, refusal: null, trace: [], lookups: [], tests: [] });
  });

  it("reads an error that arrives as a catalog message, and one with no range", () => {
    const preview = formulaPreviewResponseSchema.parse({
      diagnostics: [],
      error: { code: "eval", message: { message: "The value {0} is missing", args: ["`w`"] } },
    });
    expect(preview.error).toEqual({
      code: "eval",
      message: { message: "The value {0} is missing", args: ["`w`"] },
      range: null,
    });
  });
});
