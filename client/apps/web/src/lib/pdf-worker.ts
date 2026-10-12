import { pdfjs } from "react-pdf";

/**
 * Points PDF.js at the worker bundled with the app, once, for every page
 * that renders a PDF with react-pdf. Bundled rather than fetched from a CDN,
 * so a document opens wherever the app does.
 */
if (typeof window !== "undefined") {
  pdfjs.GlobalWorkerOptions.workerSrc = new URL(
    "pdfjs-dist/build/pdf.worker.min.mjs",
    import.meta.url,
  ).toString();
}
