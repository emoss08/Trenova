import "@/lib/pdf-worker";
import "react-pdf/dist/Page/TextLayer.css";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Document, Page } from "react-pdf";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { markMatches } from "./highlight";

/** The widest a page is drawn, and a letter page's width over its height. */
const MAX_WIDTH = 560;
const PAGE_RATIO = 8.5 / 11;

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
    const area = frame.current?.parentElement;
    if (area === null || area === undefined) {
      return;
    }
    const observer = new ResizeObserver(([entry]) => {
      const { width: across, height: down } = entry.contentRect;
      setWidth(Math.floor(Math.min(MAX_WIDTH, across, down * PAGE_RATIO)));
    });
    observer.observe(area);
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
          loading={<Skeleton className="aspect-[8.5/11] w-105 max-w-full" />}
          error={null}
        >
          <Page
            pageNumber={page}
            width={width}
            renderAnnotationLayer={false}
            customTextRenderer={renderText}
            className="ring-border rounded-xs ring-1"
            onRenderError={onFailed}
            loading={<Skeleton className="aspect-[8.5/11] w-105 max-w-full" />}
          />
        </Document>
      )}
    </div>
  );
}
