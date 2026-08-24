import { describe, expect, it } from "vitest";
import {
  canCommit,
  changeKindLabel,
  changesAnything,
  changeTone,
  commitWarnings,
  describeFieldChange,
  importHeadline,
  meaningfulChanges,
} from "../rate-import";
import type { RateImportBatch, RateImportChange, RateImportSummary } from "../../types/rate";

function summary(overrides: Partial<RateImportSummary> = {}): RateImportSummary {
  return { added: 0, changed: 0, removed: 0, duplicate: 0, unchanged: 0, ...overrides };
}

function batch(overrides: Partial<RateImportBatch> = {}): RateImportBatch {
  return {
    rateAgreementId: "rag_1",
    fileName: "q3-tariff.csv",
    sourceFormat: "CSV",
    status: "Parsed",
    effectiveFrom: 1,
    rowCount: 100,
    errorCount: 0,
    error: "",
    summary: summary({ added: 3, changed: 5 }),
    changes: [],
    unmappedHeaders: [],
    ...overrides,
  } as RateImportBatch;
}

function change(overrides: Partial<RateImportChange> = {}): RateImportChange {
  return {
    kind: "Changed",
    laneKey: "ST:IL>ST:GA",
    label: "IL → GA",
    ...overrides,
  } as RateImportChange;
}

describe("canCommit", () => {
  it("is true for an import that has been read and not yet applied", () => {
    expect(canCommit(batch({ status: "Parsed" }))).toBe(true);
  });

  // Committing twice would amend the agreement twice.
  it("is false once an import has been applied", () => {
    expect(canCommit(batch({ status: "Committed" }))).toBe(false);
  });

  it("is false for an import that could not be read", () => {
    expect(canCommit(batch({ status: "Failed" }))).toBe(false);
  });

  it("is false before an import exists", () => {
    expect(canCommit(undefined)).toBe(false);
  });
});

describe("changeTone", () => {
  // A lane that stops pricing is the only change where doing nothing produces
  // a shipment nobody can invoice.
  it("marks a lane that stops pricing as the thing to look at", () => {
    expect(changeTone("Removed")).toBe("danger");
  });

  it("marks a lane listed twice as worth a second look", () => {
    expect(changeTone("Duplicate")).toBe("warning");
  });

  it("keeps an unchanged lane quiet", () => {
    expect(changeTone("Unchanged")).toBe("quiet");
  });
});

describe("changeKindLabel", () => {
  it("says what a removal actually does rather than naming the operation", () => {
    expect(changeKindLabel("Removed")).toBe("Stops pricing");
  });
});

describe("meaningfulChanges", () => {
  // A monthly sheet is mostly unchanged lanes, and showing all of them buries
  // the handful that moved.
  it("hides the lanes a sheet restated unchanged", () => {
    const rows = meaningfulChanges([
      change({ laneKey: "a", kind: "Changed" }),
      change({ laneKey: "b", kind: "Unchanged" }),
      change({ laneKey: "c", kind: "Removed" }),
    ]);

    expect(rows.map((row) => row.laneKey)).toEqual(["a", "c"]);
  });

  it("has nothing to show before a sheet has been read", () => {
    expect(meaningfulChanges(null)).toEqual([]);
  });
});

describe("describeFieldChange", () => {
  it("shows what a term moved from and to", () => {
    expect(describeFieldChange({ field: "rate", before: "2.10", after: "2.35" })).toBe(
      "rate 2.10 → 2.35",
    );
  });

  // A minimum charge silently disappearing is exactly what a dry run exists to
  // catch, so it says "removed" rather than showing an empty value.
  it("says a term was removed rather than showing a blank", () => {
    expect(describeFieldChange({ field: "minCharge", before: "450", after: "" })).toBe(
      "minCharge removed (was 450)",
    );
  });

  it("says a term was set rather than showing a blank before", () => {
    expect(describeFieldChange({ field: "minCharge", before: "", after: "450" })).toBe(
      "minCharge set to 450",
    );
  });
});

describe("importHeadline", () => {
  it("states what committing would leave the agreement with", () => {
    const headline = importHeadline(batch({ summary: summary({ added: 3, changed: 5 }) }));

    expect(headline).toContain("3 new");
    expect(headline).toContain("5 changed");
  });

  it("says out loud when lanes would stop pricing", () => {
    expect(importHeadline(batch({ summary: summary({ removed: 2 }) }))).toContain(
      "would stop pricing",
    );
  });

  // Usually this means somebody uploaded last month's file again, and a screen
  // full of "no change" rows does not make that obvious.
  it("says a sheet that changes nothing may already have been imported", () => {
    expect(importHeadline(batch({ summary: summary({ unchanged: 100 }) }))).toContain(
      "already been imported",
    );
  });

  it("shows the reason a file that could not be read gives", () => {
    expect(
      importHeadline(batch({ status: "Failed", error: "this file has no rate rows in it" })),
    ).toContain("no rate rows");
  });

  it("says an applied sheet has been applied rather than describing it as pending", () => {
    expect(importHeadline(batch({ status: "Committed" }))).toContain("has been applied");
  });
});

describe("changesAnything", () => {
  it("does not count unchanged lanes as a change", () => {
    expect(changesAnything(summary({ unchanged: 100 }))).toBe(false);
  });

  // A sheet whose only effect is withdrawing lanes is very much a change.
  it("counts a withdrawal on its own as a change", () => {
    expect(changesAnything(summary({ removed: 1 }))).toBe(true);
  });
});

describe("commitWarnings", () => {
  it("says how many rows could not be read, and out of how many", () => {
    const warnings = commitWarnings(batch({ rowCount: 100, errorCount: 4 }));

    expect(warnings[0]).toContain("4 of 100");
  });

  // The withheld removals are the part somebody would otherwise not know about,
  // and it changes what the rest of the screen means.
  it("explains why no lane is shown as stopping while rows are unreadable", () => {
    expect(commitWarnings(batch({ errorCount: 4 }))[0]).toContain("named no lane");
  });

  it("names the columns nothing was read from", () => {
    const warnings = commitWarnings(batch({ unmappedHeaders: ["Sales Rep", "Notes"] }));

    expect(warnings[0]).toContain("Sales Rep");
    expect(warnings[0]).toContain("Notes");
  });

  it("says only the first of a repeated lane will be imported", () => {
    expect(commitWarnings(batch({ summary: summary({ duplicate: 2 }) }))[0]).toContain(
      "first of each",
    );
  });

  it("warns before lanes stop pricing", () => {
    expect(commitWarnings(batch({ summary: summary({ removed: 3 }) }))[0]).toContain(
      "will stop pricing",
    );
  });

  it("has nothing to warn about on a clean sheet", () => {
    expect(commitWarnings(batch())).toEqual([]);
  });
});
