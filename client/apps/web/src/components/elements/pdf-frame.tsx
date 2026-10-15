import { usePdfDocument } from "@/hooks/use-pdf-document";
import { pdfViewerFragment } from "@/lib/pdf-viewer-fragment";
import { File06Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useElementSize } from "@trenova/shared/hooks/use-element-size";
import { useObjectUrl } from "@trenova/shared/hooks/use-object-url";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { CSSProperties, ReactNode } from "react";

export type PdfFrameProps = {
  /** The API endpoint that serves the PDF, as `api.download` takes it. */
  endpoint: string;
  /** The frame's accessible name: what the document is. */
  title: string;
  page?: number;
  /** 1 is 100%. */
  zoom?: number;
  /** Fits the whole page in the frame when no zoom is given. */
  fit?: boolean;
  /** False hides the viewer's toolbar, for a single page shown as a picture of itself. */
  toolbar?: boolean;
  /** Degrees clockwise, a quarter turn at a time: the frame is turned, not the file. */
  rotate?: number;
  /** Drawn while the file arrives; a plain shimmer when not given. */
  loading?: ReactNode;
  className?: string;
};

/** A rotation in degrees as quarter turns clockwise, 0 to 3. */
function quarterTurns(rotate: number): number {
  return ((Math.round(rotate / 90) % 4) + 4) % 4;
}

/**
 * A PDF shown by the browser's own viewer, which brings its paging, zoom, search and
 * printing. The file is fetched with the session's credentials and opened from a blob URL,
 * since the API refuses to be framed. A browser with no built-in viewer, usually on a phone,
 * is offered the file to open instead.
 */
export function PdfFrame({
  endpoint,
  title,
  page,
  zoom,
  fit,
  toolbar,
  rotate = 0,
  loading,
  className,
}: PdfFrameProps) {
  const t = useT();
  const pdf = usePdfDocument(endpoint);
  const url = useObjectUrl(pdf.data);
  const [measure, size] = useElementSize<HTMLDivElement>();
  const turns = quarterTurns(rotate);

  if (pdf.isError) {
    return (
      <div className={cn("flex size-full items-center justify-center p-6", className)}>
        <div className="text-foreground-subtle flex flex-col items-center gap-2 text-center text-sm">
          <File06Icon className="size-6" />
          <p>{t("This document could not be shown.")}</p>
          <Button size="sm" variant="outline" onClick={() => void pdf.refetch()}>
            {t("Try again")}
          </Button>
        </div>
      </div>
    );
  }

  if (url === null) {
    return (
      loading ?? (
        <div
          aria-busy="true"
          aria-label={t("Loading the document")}
          className={cn("ui-shimmer size-full rounded-md", className)}
        />
      )
    );
  }

  if (typeof navigator !== "undefined" && navigator.pdfViewerEnabled === false) {
    return (
      <div className={cn("flex size-full items-center justify-center p-6", className)}>
        <div className="text-foreground-subtle flex flex-col items-center gap-2 text-center text-sm">
          <File06Icon className="size-6" />
          <p>{t("This browser can't show PDFs inside the page.")}</p>
          <Button
            size="sm"
            variant="outline"
            onClick={() => {
              const tab = window.open(url, "_blank");
              if (tab !== null) {
                tab.opener = null;
              }
            }}
          >
            {t("Open the PDF")}
          </Button>
        </div>
      </div>
    );
  }

  const src = `${url}${pdfViewerFragment({ page, zoom, fit, toolbar })}`;
  const sideways = turns % 2 === 1;
  const turned: CSSProperties | undefined =
    turns === 0
      ? undefined
      : {
          position: "absolute",
          top: "50%",
          left: "50%",
          width: sideways ? size.height : size.width,
          height: sideways ? size.width : size.height,
          transform: `translate(-50%, -50%) rotate(${turns * 90}deg)`,
        };

  return (
    <div ref={measure} className={cn("relative size-full overflow-hidden", className)}>
      {(turns === 0 || size.width > 0) && (
        <iframe
          key={src}
          src={src}
          title={title}
          style={turned}
          className={cn("border-0", turns === 0 && "size-full")}
        />
      )}
    </div>
  );
}
