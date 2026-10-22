import { describe, expect, it } from "vitest";
import { EMPTY_LIBRARY, libraryFormValues, saveDemotesLibrary } from "../library-form";

const saved = { name: "accessorials", source: 'library "org.accessorials"\n' };

describe("saveDemotesLibrary", () => {
  it("is true when an approved or in-review library's source or name changes", () => {
    for (const status of ["Active", "InReview"] as const) {
      const library = { ...saved, status };
      expect(saveDemotesLibrary(library, { ...saved, source: `${saved.source}\n// more` })).toBe(
        true,
      );
      expect(saveDemotesLibrary(library, { ...saved, name: "fees" })).toBe(true);
    }
  });

  it("is false when only whitespace around the name changes, or nothing material does", () => {
    const library = { ...saved, status: "Active" as const };
    expect(saveDemotesLibrary(library, { ...saved, name: " accessorials " })).toBe(false);
    expect(saveDemotesLibrary(library, saved)).toBe(false);
  });

  it("is false for a draft or archived library, and before one is saved", () => {
    const changed = { ...saved, source: 'library "org.other"\n' };
    expect(saveDemotesLibrary({ ...saved, status: "Draft" }, changed)).toBe(false);
    expect(saveDemotesLibrary({ ...saved, status: "Inactive" }, changed)).toBe(false);
    expect(saveDemotesLibrary(null, changed)).toBe(false);
  });
});

describe("libraryFormValues", () => {
  it("reads the form's fields from a saved library, a missing description as empty", () => {
    expect(
      libraryFormValues({
        name: "accessorials",
        description: undefined as unknown as string,
        source: saved.source,
        status: "Active",
      }),
    ).toEqual({ name: "accessorials", description: "", source: saved.source, status: "Active" });
  });

  it("starts a new library as an empty draft", () => {
    expect(EMPTY_LIBRARY).toEqual({ name: "", description: "", source: "", status: "Draft" });
  });
});
