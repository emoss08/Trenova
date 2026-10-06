import { PdfPage } from "@/components/elements/pdf-viewer";
import type { CapturePage } from "@/lib/graphql/capture";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { useT } from "@trenova/shared/i18n/use-t";
import { apiUrl } from "@trenova/shared/lib/api-url";
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  RefreshCcw01Icon,
  RefreshCw01Icon,
  ZoomInIcon,
  ZoomOutIcon,
} from "@trenova/shared/components/icons";
import { useState, type KeyboardEvent } from "react";
import type { PagePlace } from "./page-layout";

const ZOOM_STEP = 0.25;
const ZOOM_MIN = 0.5;
const ZOOM_MAX = 3;

/**
 * One page at full size, turned as the person turned it, saved or not, with
 * the pages before and after it in the stack's order a key away. Turning it
 * here turns it in the stack.
 */
export function PagePreviewDialog({
  page,
  number,
  rotation,
  place,
  position,
  total,
  onPrevious,
  onNext,
  onRotate,
  onOpenChange,
}: {
  page: CapturePage | null;
  number: number;
  rotation: number;
  place: PagePlace | null;
  /** Its place among every page in the stack, from 1. */
  position: number;
  total: number;
  onPrevious: () => void;
  onNext: () => void;
  /** Absent when the stack cannot be changed. */
  onRotate?: (quarterTurns: number) => void;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const [zoom, setZoom] = useState(1);
  const open = page !== null;
  const hasPrevious = position > 1;
  const hasNext = position < total;

  const onKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.defaultPrevented || event.altKey || event.ctrlKey || event.metaKey) {
      return;
    }
    if (event.key === "ArrowLeft" && hasPrevious) {
      event.preventDefault();
      onPrevious();
    } else if (event.key === "ArrowRight" && hasNext) {
      event.preventDefault();
      onNext();
    }
  };

  const where =
    place === null
      ? null
      : place.document === null
        ? t("Set aside")
        : t("Document {0}, page {1} of {2}", place.document, place.index, place.of);
  const turned =
    rotation === 0
      ? t("As it was captured")
      : t("Turned {0}° clockwise, as it will be filed", rotation);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        size="xl"
        className="max-h-[90vh] grid-rows-[auto_auto_minmax(0,1fr)]"
        onKeyDown={onKeyDown}
      >
        <DialogHeader>
          <DialogTitle>{t("Page {0}", number)}</DialogTitle>
          <DialogDescription>{[where, turned].filter(Boolean).join(" · ")}</DialogDescription>
        </DialogHeader>

        <div className="flex flex-wrap items-center justify-between gap-2">
          <div className="flex items-center gap-1">
            <Button
              type="button"
              size="icon-sm"
              variant="outline"
              onClick={onPrevious}
              disabled={!hasPrevious}
              aria-label={t("Previous page")}
              aria-keyshortcuts="ArrowLeft"
            >
              <ChevronLeftIcon className="size-4" />
            </Button>
            <span className="text-foreground-muted min-w-16 text-center text-xs tabular-nums">
              {t("{0} of {1}", position, total)}
            </span>
            <Button
              type="button"
              size="icon-sm"
              variant="outline"
              onClick={onNext}
              disabled={!hasNext}
              aria-label={t("Next page")}
              aria-keyshortcuts="ArrowRight"
            >
              <ChevronRightIcon className="size-4" />
            </Button>
          </div>

          <div className="flex items-center gap-1">
            <Button
              type="button"
              size="icon-sm"
              variant="ghost"
              onClick={() => setZoom((z) => Math.max(ZOOM_MIN, z - ZOOM_STEP))}
              disabled={zoom <= ZOOM_MIN}
              aria-label={t("Zoom out")}
            >
              <ZoomOutIcon className="size-4" />
            </Button>
            <Button
              type="button"
              size="sm"
              variant="ghost"
              className="min-w-14 tabular-nums"
              onClick={() => setZoom(1)}
              title={t("Fit to width")}
            >
              {Math.round(zoom * 100)}%
            </Button>
            <Button
              type="button"
              size="icon-sm"
              variant="ghost"
              onClick={() => setZoom((z) => Math.min(ZOOM_MAX, z + ZOOM_STEP))}
              disabled={zoom >= ZOOM_MAX}
              aria-label={t("Zoom in")}
            >
              <ZoomInIcon className="size-4" />
            </Button>
            {onRotate !== undefined && (
              <>
                <Button
                  type="button"
                  size="icon-sm"
                  variant="ghost"
                  onClick={() => onRotate(-1)}
                  aria-label={t("Rotate left")}
                >
                  <RefreshCcw01Icon className="size-4" />
                </Button>
                <Button
                  type="button"
                  size="icon-sm"
                  variant="ghost"
                  onClick={() => onRotate(1)}
                  aria-label={t("Rotate right")}
                >
                  <RefreshCw01Icon className="size-4" />
                </Button>
              </>
            )}
          </div>
        </div>

        {page !== null && (
          <PdfPage
            key={page.id}
            file={apiUrl(page.contentPath)}
            rotate={rotation}
            zoom={zoom}
            className="border-border bg-muted min-h-[60vh] rounded-md border"
          />
        )}
      </DialogContent>
    </Dialog>
  );
}
