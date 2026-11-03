import type { ExtractedField, PageSize } from "@/lib/document-reading/extraction-payload";
import { boxStyle, pageAspect, scrollToBox } from "@/lib/document-reading/page-geometry";
import { PAGE_IMAGE_CACHE, type PageImageSource } from "@/lib/queries/document-page";
import { AlertTriangleIcon, RefreshCw01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useObjectUrl } from "@trenova/shared/hooks/use-object-url";
import { useReducedMotion } from "@trenova/shared/hooks/use-reduced-motion";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { useEffect, useRef, useState, type CSSProperties, type RefObject } from "react";
import type { PageFlag } from "./billing-queue-page-viewer-types";

/** A page's box in the column, sized before its drawing arrives so nothing moves when it does. */
export function sheetBox(width: number, aspect: number, rotation: number) {
  const sideways = rotation % 180 !== 0;
  const height = sideways ? width * aspect : width / aspect;
  const face: CSSProperties = sideways
    ? {
        position: "absolute",
        top: "50%",
        left: "50%",
        width: height,
        height: width,
        transform: `translate(-50%, -50%) rotate(${rotation}deg)`,
      }
    : {
        position: "absolute",
        inset: 0,
        transform: rotation ? `rotate(${rotation}deg)` : undefined,
      };
  return { height, face };
}

/**
 * One page of the column, fetched only while it is near the viewport. It holds its own shape
 * from the page's known size (a letter page until the drawing says otherwise), draws a box over
 * every value read from it, and brings the selected value to the middle of the viewer.
 */
export function BillingQueuePageSheet({
  source,
  fileName,
  number,
  size,
  width,
  rotation,
  marks,
  selected,
  hot,
  onPoint,
  onSelect,
  wellRef,
  onDownload,
  flag,
  onRotate,
}: {
  source: PageImageSource;
  fileName: string;
  number: number;
  size: PageSize | undefined;
  width: number;
  rotation: number;
  marks: readonly ExtractedField[];
  selected: string | null;
  hot: string | null;
  onPoint: (key: string | null) => void;
  onSelect: (key: string) => void;
  wellRef: RefObject<HTMLDivElement | null>;
  onDownload: () => void;
  flag: PageFlag | undefined;
  onRotate: (() => void) | null;
}) {
  const t = useT();
  const reduced = useReducedMotion();
  const sheetRef = useRef<HTMLDivElement>(null);
  const [drawn, setDrawn] = useState<number | null>(null);
  const imageQuery = useQuery({ ...source.image(number), ...PAGE_IMAGE_CACHE, retry: 1 });
  const imageUrl = useObjectUrl(imageQuery.data);
  const ready = imageUrl !== null && drawn !== null;
  const aspect = drawn ?? pageAspect(size);
  const { height, face } = sheetBox(width, aspect, rotation);
  const target = marks.find((mark) => mark.key === selected) ?? null;
  const box = target?.box ?? null;

  useEffect(() => {
    const view = wellRef.current;
    const sheet = sheetRef.current;
    if (!ready || box === null || view === null || sheet === null) return;
    if (typeof view.scrollTo !== "function") return;
    const frame = requestAnimationFrame(() => {
      const sheetRect = sheet.getBoundingClientRect();
      const viewRect = view.getBoundingClientRect();
      const { left, top } = scrollToBox(box, {
        pageLeft: sheetRect.left - viewRect.left + view.scrollLeft,
        pageTop: sheetRect.top - viewRect.top + view.scrollTop,
        pageWidth: sheet.offsetWidth,
        pageHeight: sheet.offsetHeight,
        viewWidth: view.clientWidth,
        viewHeight: view.clientHeight,
      });
      view.scrollTo({ left, top, behavior: reduced ? "auto" : "smooth" });
    });
    return () => cancelAnimationFrame(frame);
  }, [box, width, rotation, ready, reduced, wellRef]);

  return (
    <div
      ref={sheetRef}
      data-sheet={number}
      className="group/sheet border-border bg-card relative shrink-0 border"
      aria-busy={!ready && !imageQuery.isError}
      style={{ width, height }}
    >
      {flag ? (
        <div
          role="note"
          className="bg-warning-subtle border-warning-border text-warning-subtle-foreground absolute inset-x-3 top-3 z-10 flex items-center gap-3 rounded-md border py-2 pr-2 pl-3"
        >
          <AlertTriangleIcon className="size-4 shrink-0" />
          <div className="min-w-0 flex-1 text-xs leading-snug">
            <p className="font-medium">{flag.title}</p>
            <p className="opacity-80">{flag.detail}</p>
          </div>
          {flag.action ? <FlagAction action={flag.action} /> : null}
        </div>
      ) : null}
      {onRotate ? (
        <button
          type="button"
          onClick={onRotate}
          aria-label={t("Rotate page {0}", number)}
          className={cn(
            "ui-focus-ring bg-foreground text-background absolute right-2 z-10 flex h-6.5 items-center gap-1 rounded-md px-2 text-xs opacity-0 transition-opacity group-hover/sheet:opacity-100 focus-visible:opacity-100",
            flag ? "bottom-2" : "top-2",
          )}
        >
          <RefreshCw01Icon className="size-3" />
          {t("Rotate page")}
        </button>
      ) : null}
      {imageQuery.isError ? (
        <div
          role="status"
          className="absolute inset-0 flex flex-col items-center justify-center gap-2 p-4 text-center text-sm"
        >
          <p className="text-foreground font-medium">{t("Page {0} couldn't be drawn.", number)}</p>
          <p className="text-foreground-muted">{t("Download the file to read it.")}</p>
          <div className="flex gap-2">
            <Button size="sm" variant="outline" onClick={() => void imageQuery.refetch()}>
              {t("Try again")}
            </Button>
            <Button size="sm" variant="outline" onClick={onDownload}>
              {t("Download")}
            </Button>
          </div>
        </div>
      ) : (
        <>
          {!ready ? <Skeleton className="absolute inset-0 rounded-none" /> : null}
          <div style={face}>
            {imageUrl !== null ? (
              <img
                className={cn("block size-full", !ready && "opacity-0")}
                src={imageUrl}
                alt={t("Page {0} of {1}", number, fileName)}
                decoding="async"
                draggable={false}
                onLoad={(event) => {
                  const image = event.currentTarget;
                  setDrawn(
                    image.naturalWidth > 0 && image.naturalHeight > 0
                      ? image.naturalWidth / image.naturalHeight
                      : pageAspect(size),
                  );
                }}
              />
            ) : null}
            {ready ? (
              <div className="absolute inset-0">
                {marks.map((mark) =>
                  mark.box ? (
                    // oxlint-disable-next-line jsx-a11y/click-events-have-key-events, jsx-a11y/no-static-element-interactions -- a box is a pointer shortcut to its field, whose row in the panel is the keyboard's way in
                    <span
                      key={mark.key}
                      aria-hidden
                      title={t("{0}: {1}", mark.label, mark.value)}
                      style={boxStyle(mark.box)}
                      onMouseEnter={() => onPoint(mark.key)}
                      onMouseLeave={() => onPoint(null)}
                      onClick={() => onSelect(mark.key)}
                      className={cn(
                        "absolute cursor-pointer rounded-xs border transition-colors",
                        mark.needsLook
                          ? "border-warning bg-warning/10"
                          : "bg-brand/8 border-brand/30",
                        (hot === mark.key || selected === mark.key) && "border-brand bg-brand/15",
                        selected === mark.key && "ring-brand ring-2",
                      )}
                    />
                  ) : null,
                )}
                {target?.box ? (
                  <span
                    aria-hidden
                    className="bg-brand text-foreground-on-solid absolute -translate-y-full rounded-xs px-1.5 py-0.5 text-2xs font-medium whitespace-nowrap"
                    style={{ left: `${target.box.x * 100}%`, top: `${target.box.y * 100}%` }}
                  >
                    {target.label}
                  </span>
                ) : null}
              </div>
            ) : null}
          </div>
        </>
      )}
    </div>
  );
}

function FlagAction({ action }: { action: NonNullable<PageFlag["action"]> }) {
  const [acting, setActing] = useState(false);
  const act = async () => {
    if (acting) return;
    setActing(true);
    try {
      await action.onAction();
    } finally {
      setActing(false);
    }
  };

  return (
    <Button
      type="button"
      size="xs"
      variant="outline"
      aria-label={action.ariaLabel}
      isLoading={acting}
      disabled={action.disabled}
      onClick={() => void act()}
      className="shrink-0"
    >
      {action.label}
    </Button>
  );
}
