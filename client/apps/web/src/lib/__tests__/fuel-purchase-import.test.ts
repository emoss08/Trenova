import type { FuelPurchaseImportBatch } from "@/lib/graphql/fuel-purchase-import";
import { describe, expect, it } from "vitest";
import {
  canCommitImport,
  commitLabel,
  discardImportNotice,
  IMPORT_ROW_FILTER_STATUSES,
  importHeadline,
  importStep,
  importWarnings,
  needsDiscardConfirmation,
} from "../fuel-purchase-import";

function batch(overrides: Partial<FuelPurchaseImportBatch> = {}): FuelPurchaseImportBatch {
  return {
    id: "fpib_1",
    businessUnitId: "bu_1",
    organizationId: "org_1",
    provider: "Comdata",
    origin: "Upload",
    feedReference: null,
    documentId: "doc_1",
    fileName: "statement.csv",
    sourceFormat: "CSV",
    status: "Parsed",
    defaultFuelType: "Diesel",
    defaultFuelCardId: null,
    defaultCurrency: "USD",
    mapping: null,
    unmappedHeaders: [],
    summary: {
      rowCount: 10,
      newCount: 7,
      duplicateInFileCount: 1,
      alreadyImportedCount: 1,
      errorCount: 1,
      totalGallons: "812.450",
      totalAmount: "3104.22",
      byFuelType: null,
      byJurisdiction: null,
      earliestPurchasedAt: 1_700_000_000,
      latestPurchasedAt: 1_700_500_000,
    },
    rowCount: 10,
    errorCount: 1,
    committedCount: 0,
    error: null,
    uploadedById: "usr_1",
    stagedAt: 1_700_600_000,
    committedAt: null,
    committedById: null,
    version: 2,
    createdAt: 1_700_600_000,
    updatedAt: 1_700_600_000,
    document: null,
    defaultFuelCard: null,
    ...overrides,
  };
}

describe("importStep", () => {
  it("is set-up until a statement has been staged", () => {
    expect(importStep(undefined)).toBe("setup");
    expect(importStep(batch({ status: "Pending", summary: null }))).toBe("setup");
  });

  it("is review for a parsed or failed batch, and done once committed or discarded", () => {
    expect(importStep(batch())).toBe("review");
    expect(importStep(batch({ status: "Failed", error: "no header row" }))).toBe("review");
    expect(importStep(batch({ status: "Committed", committedCount: 7 }))).toBe("done");
    expect(importStep(batch({ status: "Discarded" }))).toBe("done");
  });
});

describe("canCommitImport", () => {
  it("allows commit only for a parsed batch with at least one new row", () => {
    expect(canCommitImport(batch())).toBe(true);
    expect(canCommitImport(undefined)).toBe(false);
    expect(canCommitImport(batch({ status: "Pending" }))).toBe(false);
    expect(canCommitImport(batch({ status: "Failed" }))).toBe(false);
    expect(canCommitImport(batch({ status: "Committed" }))).toBe(false);
    expect(canCommitImport(batch({ summary: { ...batch().summary!, newCount: 0 } }))).toBe(false);
    expect(canCommitImport(batch({ summary: null }))).toBe(false);
  });
});

describe("commitLabel", () => {
  it("names how many purchases the commit records", () => {
    expect(commitLabel(batch())).toBe("Import 7 purchases");
    expect(commitLabel(batch({ summary: { ...batch().summary!, newCount: 1 } }))).toBe(
      "Import 1 purchase",
    );
    expect(commitLabel(undefined)).toBe("Import purchases");
  });
});

describe("importHeadline", () => {
  it("says what committing would record, in gallons and money", () => {
    expect(importHeadline(batch())).toBe(
      "Importing would record 7 purchases for 812.450 gallons and $3,104.22.",
    );
  });

  it("uses the singular for one row", () => {
    expect(
      importHeadline(
        batch({
          summary: { ...batch().summary!, newCount: 1, totalGallons: "50.000", totalAmount: "200" },
        }),
      ),
    ).toBe("Importing would record 1 purchase for 50.000 gallons and $200.00.");
  });

  it("says plainly when nothing in the statement is new", () => {
    expect(importHeadline(batch({ summary: { ...batch().summary!, newCount: 0 } }))).toBe(
      "Nothing in this statement is new: every row is a duplicate, already on file, or could not be read.",
    );
  });

  it("reports the other states", () => {
    expect(importHeadline(batch({ status: "Pending", summary: null }))).toBe(
      "Waiting for the statement to be read.",
    );
    expect(importHeadline(batch({ status: "Failed", error: "no header row" }))).toBe(
      "The statement could not be read.",
    );
    expect(importHeadline(batch({ status: "Committed", committedCount: 7 }))).toBe(
      "Recorded 7 purchases.",
    );
    expect(importHeadline(batch({ status: "Discarded" }))).toBe(
      "This import was discarded; nothing was recorded.",
    );
  });

  it("formats the amount in the batch's currency", () => {
    expect(importHeadline(batch({ defaultCurrency: "CAD" }))).toContain("CA$3,104.22");
  });
});

describe("importWarnings", () => {
  it("lists what the commit will leave out, one line each", () => {
    expect(importWarnings(batch({ unmappedHeaders: ["Driver Name", "Memo"] }))).toEqual([
      "Columns ignored: Driver Name, Memo.",
      "1 row repeats a reference earlier in this statement and will be skipped.",
      "1 row matches a purchase already on file and will be skipped.",
      "1 row could not be read and will be skipped.",
    ]);
  });

  it("pluralises counts above one", () => {
    expect(
      importWarnings(
        batch({
          summary: {
            ...batch().summary!,
            duplicateInFileCount: 2,
            alreadyImportedCount: 3,
            errorCount: 4,
          },
        }),
      ),
    ).toEqual([
      "2 rows repeat a reference earlier in this statement and will be skipped.",
      "3 rows match a purchase already on file and will be skipped.",
      "4 rows could not be read and will be skipped.",
    ]);
  });

  it("is empty when every row is new and every column was read", () => {
    expect(
      importWarnings(
        batch({
          summary: {
            ...batch().summary!,
            duplicateInFileCount: 0,
            alreadyImportedCount: 0,
            errorCount: 0,
          },
        }),
      ),
    ).toEqual([]);
    expect(importWarnings(undefined)).toEqual([]);
  });
});

describe("discard confirmation", () => {
  it("asks only when a parsed batch would be thrown away", () => {
    expect(needsDiscardConfirmation(batch())).toBe(true);
    expect(needsDiscardConfirmation(batch({ status: "Pending" }))).toBe(false);
    expect(needsDiscardConfirmation(batch({ status: "Failed" }))).toBe(false);
    expect(needsDiscardConfirmation(batch({ status: "Committed" }))).toBe(false);
    expect(needsDiscardConfirmation(undefined)).toBe(false);
  });

  it("tells the user nothing has been recorded and the statement can be uploaded again", () => {
    const notice = discardImportNotice(batch());
    expect(notice.title).toBe("Discard this import?");
    expect(notice.description).toBe(
      "The 10 rows read from statement.csv will be thrown away. Nothing has been recorded, and the statement can be uploaded again.",
    );
  });
});

describe("IMPORT_ROW_FILTER_STATUSES", () => {
  it("maps each review tab to the row statuses it shows", () => {
    expect(IMPORT_ROW_FILTER_STATUSES.all).toBeUndefined();
    expect(IMPORT_ROW_FILTER_STATUSES.new).toEqual(["New"]);
    expect(IMPORT_ROW_FILTER_STATUSES.duplicates).toEqual(["DuplicateInFile", "AlreadyImported"]);
    expect(IMPORT_ROW_FILTER_STATUSES.errors).toEqual(["Error"]);
  });
});
