import { PDFDocument, StandardFonts } from "pdf-lib";
import { describe, expect, it } from "vitest";
import {
  buildCoverSheetPdf,
  printable,
  qrRuns,
  type CoverSheetPrint,
  type CoverSheetWording,
} from "../capture-cover-sheet";

const wording: CoverSheetWording = {
  heading: "Trenova cover sheet",
  separatorTitle: "Separator",
  documentTypeLabel: "Document type",
  instructions: "Put this sheet on top of the pages for this record and scan the stack.",
  separatorInstructions: "Put this sheet between documents to split the stack.",
  expires: (date) => `Use by ${date}`,
};

function sheet(overrides: Partial<CoverSheetPrint>): CoverSheetPrint {
  return {
    id: "ccs_01",
    modules: ["1101", "0110", "1111", "0000"],
    record: { kind: "Shipment", title: "PRO-4471", subtitle: "BOL 88213" },
    documentType: "Proof of delivery",
    expiresOn: "Dec 31, 2026",
    ...overrides,
  };
}

describe("qrRuns", () => {
  it("draws each stretch of dark modules in a row as one run", () => {
    expect(qrRuns(["1101", "0110", "1111", "0000"])).toEqual([
      { x: 0, y: 0, width: 2 },
      { x: 3, y: 0, width: 1 },
      { x: 1, y: 1, width: 2 },
      { x: 0, y: 2, width: 4 },
    ]);
  });

  it("draws nothing for an empty code", () => {
    expect(qrRuns([])).toEqual([]);
  });
});

describe("buildCoverSheetPdf", () => {
  it("prints one letter page per sheet, separators included", async () => {
    const bytes = await buildCoverSheetPdf(
      [sheet({}), sheet({ id: "ccs_02", record: null, documentType: null })],
      wording,
    );
    const pdf = await PDFDocument.load(bytes);

    expect(pdf.getPageCount()).toBe(2);
    for (const page of pdf.getPages()) {
      expect(page.getSize()).toEqual({ width: 612, height: 792 });
    }
    expect(pdf.getTitle()).toBe("Trenova cover sheet");
  });

  it("prints a record named in a script the standard fonts cannot draw", async () => {
    const bytes = await buildCoverSheetPdf(
      [sheet({ record: { kind: "運單", title: "PRO-4471 上海", subtitle: "" } })],
      wording,
    );

    expect((await PDFDocument.load(bytes)).getPageCount()).toBe(1);
  });
});

describe("printable", () => {
  it("keeps what the font draws and marks what it cannot", async () => {
    const font = await (await PDFDocument.create()).embedFont(StandardFonts.Helvetica);
    expect(printable(font, "PRO-4471 · Müller")).toBe("PRO-4471 · Müller");
    expect(printable(font, "運單 7")).toBe("?? 7");
  });
});
