/** What a built-in PDF viewer is asked to show first, through the URL's fragment. */
export type PdfViewerOptions = {
  /** One-based; the first page needs no fragment. */
  page?: number;
  /** 1 is 100%. Wins over `fit`. */
  zoom?: number;
  /** Fits the whole page in the frame. */
  fit?: boolean;
  /** False hides the viewer's own toolbar where the browser allows it. */
  toolbar?: boolean;
};

/**
 * The fragment the browsers' built-in PDF viewers read from a document's URL, in the form
 * Adobe's open parameters define: `#page=3&zoom=150&toolbar=0`. Chrome, Edge and Firefox
 * honour all of them; Safari honours the page. Anything a viewer does not know it ignores,
 * so the document still opens.
 */
export function pdfViewerFragment({ page, zoom, fit, toolbar }: PdfViewerOptions): string {
  const parts: string[] = [];
  if (page !== undefined && Number.isFinite(page) && page > 1) {
    parts.push(`page=${Math.floor(page)}`);
  }
  if (zoom !== undefined && Number.isFinite(zoom) && zoom > 0) {
    parts.push(`zoom=${Math.round(zoom * 100)}`);
  } else if (fit) {
    parts.push("view=Fit");
  }
  if (toolbar === false) {
    parts.push("toolbar=0");
  }

  return parts.length === 0 ? "" : `#${parts.join("&")}`;
}
