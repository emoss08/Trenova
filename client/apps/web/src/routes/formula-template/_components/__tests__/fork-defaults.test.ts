import type { FormulaTemplate } from "@trenova/shared/types/formula-template";
import { describe, expect, it } from "vitest";
import { forkDefaultsFor } from "../fork-template-dialog";

describe("forkDefaultsFor", () => {
  it("derives the name and version from the template being forked", () => {
    const template = { name: "Per Mile", currentVersionNumber: 4 } as FormulaTemplate;
    expect(forkDefaultsFor(template)).toEqual({
      newName: "Per Mile (Fork)",
      sourceVersion: 4,
      changeMessage: "",
    });
  });

  it("is blank when no template is selected yet", () => {
    expect(forkDefaultsFor(null)).toEqual({
      newName: "",
      sourceVersion: undefined,
      changeMessage: "",
    });
  });
});
