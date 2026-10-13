import { pdfjs } from "react-pdf";

/**
 * Points PDF.js at the worker that matches the copy of PDF.js react-pdf runs,
 * once, for every page that renders a PDF with it. The version is read from
 * react-pdf rather than resolved from the app's own pdfjs-dist, which is a
 * different major version: a worker from that one refuses to load with
 * "The API version does not match the Worker version".
 */
if (typeof window !== "undefined") {
  pdfjs.GlobalWorkerOptions.workerSrc = `https://unpkg.com/pdfjs-dist@${pdfjs.version}/build/pdf.worker.min.mjs`;
}
