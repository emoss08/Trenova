import "@/lib/pdf-worker";
import "react-pdf/dist/Page/TextLayer.css";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Document, Page } from "react-pdf";
import { markMatches } from "./highlight";

/**
 * One page of a PDF, sized to the space it is given, with its text layer so
 * the value Nova read can be marked where it appears. Loaded on its own, so
 * PDF.js only reaches the browser when a document is opened.
 */
export function PdfPage({
  url,
  page,
  highlight,
  onPageCount,
  onFailed,
}: {
  url: string;
  page: number;
  /** The value to mark on the page; empty marks nothing. */
  highlight: string;
  onPageCount: (count: number) => void;
  onFailed: () => void;
}) {
  const frame = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState(0);
  const file = useMemo(() => ({ url, withCredentials: true }), [url]);

  useEffect(() => {
    const element = frame.current;
    if (element === null) {
      return;
    }
    const observer = new ResizeObserver(([entry]) => {
      setWidth(Math.min(560, Math.floor(entry.contentRect.width)));
    });
    observer.observe(element);
    return () => observer.disconnect();
  }, []);

  const renderText = useCallback(
    ({ str }: { str: string }) => markMatches(str, highlight),
    [highlight],
  );

  return (
    <div ref={frame} className="tb-pdf flex w-full justify-center">
      {width > 0 && (
        <Document
          file={file}
          onLoadSuccess={({ numPages }) => onPageCount(numPages)}
          onLoadError={onFailed}
          onSourceError={onFailed}
          loading={null}
          error={null}
        >
          <Page
            pageNumber={page}
            width={width}
            renderAnnotationLayer={false}
            customTextRenderer={renderText}
            className="ring-border rounded-xs ring-1"
            onRenderError={onFailed}
            loading={null}
          />
        </Document>
      )}
    </div>
  );
}
