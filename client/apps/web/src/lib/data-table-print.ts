import type { RowData } from "@tanstack/react-table";
import { escapeHtml } from "@trenova/shared/lib/html";
import { formatCsvValue, type ExportColumn } from "./data-table-export";

/** The most rows one print holds; more than this belongs in a PDF report. */
export const PRINT_MAX_ROWS = 2_000;

export type PrintDocumentParams<TData extends RowData> = {
  title: string;
  /** What the rows are: their filters, how many, when they were printed. */
  subtitle: string;
  columns: readonly ExportColumn<TData>[];
  rows: readonly TData[];
  /** Columns laid out landscape once there are more than this. */
  landscapeAfter?: number;
};

// i18n-ignore: a print stylesheet, not text a person reads
const PRINT_STYLES = `
  @page { margin: 12mm; }
  * { box-sizing: border-box; }
  body { font: 10px/1.35 system-ui, -apple-system, "Segoe UI", sans-serif; color: #171717; margin: 0; }
  header { margin-bottom: 8px; }
  h1 { font-size: 14px; font-weight: 600; margin: 0 0 2px; }
  p { margin: 0; color: #525252; }
  table { width: 100%; border-collapse: collapse; table-layout: auto; }
  thead { display: table-header-group; }
  tr { break-inside: avoid; }
  th { text-align: left; font-weight: 500; color: #525252; border-bottom: 1px solid #a3a3a3; padding: 4px 6px; }
  td { border-bottom: 1px solid #e5e5e5; padding: 3px 6px; vertical-align: top; overflow-wrap: anywhere; }
  td.num { text-align: right; font-variant-numeric: tabular-nums; }
`;

function isNumeric(value: unknown): boolean {
  return typeof value === "number" || typeof value === "bigint";
}

/**
 * A table as a standalone page built for paper: the header repeats on every sheet,
 * rows never split across one, numbers line up on the right, and a wide table turns
 * landscape. Every value is escaped; nothing from a row is read as markup.
 */
export function buildPrintDocument<TData extends RowData>({
  title,
  subtitle,
  columns,
  rows,
  landscapeAfter = 7,
}: PrintDocumentParams<TData>): string {
  const head = columns.map((column) => `<th>${escapeHtml(column.header)}</th>`).join("");
  const body = rows
    .map((row) => {
      const cells = columns
        .map((column) => {
          const value = column.getValue(row);
          const text = escapeHtml(formatCsvValue(value));
          // i18n-ignore: table markup around an already escaped value
          return isNumeric(value) ? `<td class="num">${text}</td>` : `<td>${text}</td>`;
        })
        .join("");
      return `<tr>${cells}</tr>`;
    })
    .join("");
  const orientation = columns.length > landscapeAfter ? "@page { size: landscape; }" : "";

  return [
    "<!doctype html>",
    '<html><head><meta charset="utf-8">',
    `<title>${escapeHtml(title)}</title>`,
    `<style>${PRINT_STYLES}${orientation}</style>`,
    "</head><body>",
    `<header><h1>${escapeHtml(title)}</h1><p>${escapeHtml(subtitle)}</p></header>`,
    `<table><thead><tr>${head}</tr></thead><tbody>${body}</tbody></table>`,
    "</body></html>",
  ].join("");
}

/**
 * Opens the browser's print dialog on a document without leaving the page. The
 * document is printed from a hidden frame that is removed once the dialog closes;
 * "Save as PDF" in that dialog is how a person gets a PDF of it.
 */
export function printDocument(html: string): Promise<void> {
  return new Promise((resolve, reject) => {
    const frame = document.createElement("iframe");
    frame.setAttribute("aria-hidden", "true");
    frame.setAttribute("sandbox", "allow-modals allow-same-origin");
    frame.style.position = "fixed";
    frame.style.width = "0";
    frame.style.height = "0";
    frame.style.border = "0";
    frame.style.visibility = "hidden";

    const cleanUp = () => {
      frame.remove();
      resolve();
    };

    frame.onload = () => {
      const view = frame.contentWindow;
      if (!view) {
        frame.remove();
        reject(new Error("The print preview could not be opened."));
        return;
      }
      view.addEventListener("afterprint", cleanUp, { once: true });
      view.focus();
      view.print();
    };
    frame.srcdoc = html;
    document.body.append(frame);
  });
}
