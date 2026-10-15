import { describe, expect, it } from "vitest";
import { pdfViewerFragment } from "../pdf-viewer-fragment";

describe("pdfViewerFragment", () => {
  it("asks for nothing when the viewer's own start is wanted", () => {
    expect(pdfViewerFragment({})).toBe("");
    expect(pdfViewerFragment({ page: 1 })).toBe("");
  });

  it("writes the page, zoom and toolbar in Adobe's open-parameter form", () => {
    expect(pdfViewerFragment({ page: 3, zoom: 1.5, toolbar: false })).toBe(
      "#page=3&zoom=150&toolbar=0",
    );
  });

  it("fits the page only when no zoom is given", () => {
    expect(pdfViewerFragment({ fit: true })).toBe("#view=Fit");
    expect(pdfViewerFragment({ fit: true, zoom: 2 })).toBe("#zoom=200");
  });

  it("drops a page or zoom that is not a usable number rather than passing it on", () => {
    expect(pdfViewerFragment({ page: Number.NaN, zoom: 0 })).toBe("");
    expect(pdfViewerFragment({ page: 2.7, zoom: Number.POSITIVE_INFINITY })).toBe("#page=2");
  });
});
