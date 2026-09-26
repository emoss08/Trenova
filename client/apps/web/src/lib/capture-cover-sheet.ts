import { PDFDocument, StandardFonts, rgb, type PDFFont, type PDFPage } from "pdf-lib";

/** One sheet to print: its code, and what it routes to, in words a person reads. */
export type CoverSheetPrint = {
  id: string;
  /** The QR code, one string per row, "1" for a dark module. */
  modules: readonly string[];
  /** Absent on a plain separator, which routes nothing. */
  record: { kind: string; title: string; subtitle: string } | null;
  documentType: string | null;
  expiresOn: string;
};

/** The sheet's fixed wording, translated by the caller. */
export type CoverSheetWording = {
  heading: string;
  separatorTitle: string;
  documentTypeLabel: string;
  instructions: string;
  separatorInstructions: string;
  expires: (date: string) => string;
};

export type QrRun = { x: number; y: number; width: number };

/**
 * The dark modules as horizontal runs, one per stretch of adjacent dark
 * modules in a row. Drawing runs rather than single modules keeps the page's
 * content small and leaves no hairline seams between modules when a printer
 * rounds each square separately.
 */
export function qrRuns(modules: readonly string[]): QrRun[] {
  const runs: QrRun[] = [];
  modules.forEach((row, y) => {
    let start = -1;
    for (let x = 0; x <= row.length; x++) {
      const dark = x < row.length && row[x] === "1";
      if (dark && start < 0) {
        start = x;
      } else if (!dark && start >= 0) {
        runs.push({ x: start, y, width: x - start });
        start = -1;
      }
    }
  });

  return runs;
}

/**
 * What of a string the font can draw. The standard PDF fonts carry only the
 * Windows Latin set, and a record named in another script would otherwise
 * stop the whole sheet from printing; a character the font lacks is replaced
 * rather than lost silently.
 */
export function printable(font: PDFFont, text: string): string {
  const supported = new Set(font.getCharacterSet());
  let out = "";
  for (const char of text) {
    const code = char.codePointAt(0) ?? 0;
    out += supported.has(code) ? char : "?";
  }

  return out;
}

function wrap(font: PDFFont, text: string, size: number, maxWidth: number): string[] {
  const lines: string[] = [];
  let line = "";
  for (const word of printable(font, text).split(/\s+/).filter(Boolean)) {
    const candidate = line === "" ? word : `${line} ${word}`;
    if (font.widthOfTextAtSize(candidate, size) <= maxWidth || line === "") {
      line = candidate;
    } else {
      lines.push(line);
      line = word;
    }
  }
  if (line !== "") {
    lines.push(line);
  }

  return lines;
}

/** Shrinks a single line until it fits, down to a floor, then cuts it. */
function fitted(font: PDFFont, text: string, size: number, floor: number, maxWidth: number) {
  const value = printable(font, text);
  let fontSize = size;
  while (fontSize > floor && font.widthOfTextAtSize(value, fontSize) > maxWidth) {
    fontSize -= 1;
  }
  let shown = value;
  while (shown.length > 1 && font.widthOfTextAtSize(shown, fontSize) > maxWidth) {
    shown = `${shown.slice(0, -2)}…`;
  }

  return { text: shown, size: fontSize };
}

const LETTER = { width: 612, height: 792 } as const;
const MARGIN = 54;
/** The code's printed side, quiet zone included: three and a quarter inches. */
const QR_SIDE = 234;
/** The quiet zone the QR symbology requires on every side, in modules. */
const QUIET_MODULES = 4;
const INK = rgb(0, 0, 0);
const MUTED = rgb(0.35, 0.37, 0.42);

function drawSheet(
  page: PDFPage,
  sheet: CoverSheetPrint,
  wording: CoverSheetWording,
  fonts: { regular: PDFFont; bold: PDFFont },
) {
  const width = LETTER.width - MARGIN * 2;
  let y = LETTER.height - MARGIN;

  const heading = fitted(fonts.bold, wording.heading, 12, 9, width);
  y -= heading.size;
  page.drawText(heading.text, { x: MARGIN, y, size: heading.size, font: fonts.bold, color: MUTED });
  y -= 18;
  page.drawLine({
    start: { x: MARGIN, y },
    end: { x: LETTER.width - MARGIN, y },
    thickness: 0.75,
    color: MUTED,
  });
  y -= 30;

  if (sheet.record === null) {
    const title = fitted(fonts.bold, wording.separatorTitle, 40, 20, width);
    y -= title.size;
    page.drawText(title.text, { x: MARGIN, y, size: title.size, font: fonts.bold, color: INK });
    y -= 20;
  } else {
    const kind = fitted(fonts.regular, sheet.record.kind, 14, 10, width);
    y -= kind.size;
    page.drawText(kind.text, { x: MARGIN, y, size: kind.size, font: fonts.regular, color: MUTED });
    y -= 10;

    const title = fitted(fonts.bold, sheet.record.title, 40, 18, width);
    y -= title.size;
    page.drawText(title.text, { x: MARGIN, y, size: title.size, font: fonts.bold, color: INK });
    y -= 10;

    if (sheet.record.subtitle !== "") {
      const subtitle = fitted(fonts.regular, sheet.record.subtitle, 18, 11, width);
      y -= subtitle.size;
      page.drawText(subtitle.text, {
        x: MARGIN,
        y,
        size: subtitle.size,
        font: fonts.regular,
        color: INK,
      });
      y -= 10;
    }
  }

  if (sheet.documentType !== null && sheet.documentType !== "") {
    const label = `${wording.documentTypeLabel}: ${sheet.documentType}`;
    const line = fitted(fonts.regular, label, 16, 11, width);
    y -= line.size + 6;
    page.drawText(line.text, { x: MARGIN, y, size: line.size, font: fonts.regular, color: INK });
  }

  const size = sheet.modules.length;
  const module = QR_SIDE / (size + QUIET_MODULES * 2);
  const qrLeft = (LETTER.width - QR_SIDE) / 2 + QUIET_MODULES * module;
  const qrTop = y - 36 - QUIET_MODULES * module;
  for (const run of qrRuns(sheet.modules)) {
    page.drawRectangle({
      x: qrLeft + run.x * module,
      y: qrTop - (run.y + 1) * module,
      width: run.width * module,
      height: module,
      color: INK,
    });
  }
  y = qrTop - size * module - QUIET_MODULES * module - 30;

  const instructions = sheet.record === null ? wording.separatorInstructions : wording.instructions;
  for (const line of wrap(fonts.regular, instructions, 12, width)) {
    y -= 12;
    page.drawText(line, { x: MARGIN, y, size: 12, font: fonts.regular, color: INK });
    y -= 5;
  }

  page.drawText(printable(fonts.regular, wording.expires(sheet.expiresOn)), {
    x: MARGIN,
    y: MARGIN,
    size: 9,
    font: fonts.regular,
    color: MUTED,
  });
  const id = printable(fonts.regular, sheet.id);
  page.drawText(id, {
    x: LETTER.width - MARGIN - fonts.regular.widthOfTextAtSize(id, 8),
    y: MARGIN,
    size: 8,
    font: fonts.regular,
    color: MUTED,
  });
}

/** Lays out one US letter page per sheet, ready to print. */
export async function buildCoverSheetPdf(
  sheets: readonly CoverSheetPrint[],
  wording: CoverSheetWording,
): Promise<Uint8Array> {
  const pdf = await PDFDocument.create();
  pdf.setTitle(wording.heading);
  pdf.setCreator("Trenova");
  const fonts = {
    regular: await pdf.embedFont(StandardFonts.Helvetica),
    bold: await pdf.embedFont(StandardFonts.HelveticaBold),
  };

  for (const sheet of sheets) {
    drawSheet(pdf.addPage([LETTER.width, LETTER.height]), sheet, wording, fonts);
  }

  return pdf.save();
}
