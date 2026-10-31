"use no memo";
import { useT } from "@trenova/shared/i18n/use-t";
import { documentPage } from "@/lib/queries/document-page";
import type { ExtractedField, PageSize } from "@/lib/document-reading/extraction-payload";
import { boxStyle, pageAspect, scrollToBox } from "@/lib/document-reading/page-geometry";
import { documentContentEndpoint } from "@/services/document";
import { api } from "@trenova/shared/lib/api";
import { ChevronLeftIcon, ChevronRightIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { toast } from "@trenova/shared/components/ui/toast";
import { useObjectUrl } from "@trenova/shared/hooks/use-object-url";
import { useReducedMotion } from "@trenova/shared/hooks/use-reduced-motion";
import { cn } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { useEffect, useRef, useState, type KeyboardEvent, type ReactNode } from "react";

/** How long a page the server drew is kept once no viewer shows it. */
const PAGE_GC_MS = 5 * 60 * 1000;
/** The page's width at 100%, as the review draws it. */
const PAGE_WIDTH_PX = 520;
const ZOOMS = [0.75, 1, 1.25, 1.5, 2, 2.5] as const;

/**
 * A document's pages as the server draws them, with a box over every value Nova
 * read from each: a faint tint where a value was read, an amber outline where one
 * needs a person, and the selected value outlined with its name above it and
 * scrolled into view. The toolbar under the page zooms, rotates, downloads and
 * turns pages; Page Up, Page Down, plus, minus and 0 do the same from the keyboard.
 */
export function BillingQueueDocumentPages({
  documentId,
  fileName,
  pageCount,
  pages,
  fields,
  page,
  onPage,
  selected,
  hot,
  onPoint,
  onSelect,
}: {
  documentId: string;
  fileName: string;
  pageCount: number;
  pages: readonly PageSize[];
  fields: readonly ExtractedField[];
  page: number;
  onPage: (page: number) => void;
  selected: string | null;
  hot: string | null;
  onPoint: (key: string | null) => void;
  onSelect: (key: string) => void;
}) {
  const t = useT();
  const wellRef = useRef<HTMLDivElement>(null);
  const [zoom, setZoom] = useState<number>(1);
  const [rotation, setRotation] = useState(0);
  const count = Math.max(1, pageCount);
  const marks = fields.filter((field) => field.page === page && field.box !== null);

  const stepZoom = (direction: 1 | -1) => {
    const at = ZOOMS.indexOf(zoom as (typeof ZOOMS)[number]);
    setZoom(ZOOMS[Math.min(ZOOMS.length - 1, Math.max(0, at + direction))]);
  };
  const turn = (to: number) => {
    if (to >= 1 && to <= count && to !== page) onPage(to);
  };
  const download = async () => {
    try {
      await api.download(documentContentEndpoint(documentId, "download"), fileName);
    } catch {
      toast.error(t("The file couldn't be downloaded"));
    }
  };
  const onKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.altKey || event.ctrlKey || event.metaKey) return;
    const actions: Record<string, () => void> = {
      PageDown: () => turn(page + 1),
      PageUp: () => turn(page - 1),
      "+": () => stepZoom(1),
      "=": () => stepZoom(1),
      "-": () => stepZoom(-1),
      "0": () => setZoom(1),
    };
    const action = actions[event.key];
    if (action) {
      event.preventDefault();
      action();
    }
  };

  return (
    <section aria-label={t("Document viewer")} className="bg-muted flex h-full min-h-0 flex-col">
      <div
        ref={wellRef}
        // oxlint-disable-next-line jsx-a11y/no-noninteractive-tabindex -- the page well scrolls and takes the page and zoom keys
        tabIndex={0}
        onKeyDown={onKeyDown}
        className="ui-inset-focus-ring flex min-h-0 flex-1 flex-col items-center gap-3 overflow-auto px-4 py-6"
      >
        <DocumentSheet
          documentId={documentId}
          fileName={fileName}
          page={page}
          size={pages.find((candidate) => candidate.number === page)}
          zoom={zoom}
          rotation={rotation}
          marks={marks}
          selected={selected}
          hot={hot}
          onPoint={onPoint}
          onSelect={onSelect}
          wellRef={wellRef}
          onDownload={() => void download()}
        />
        <div
          role="toolbar"
          aria-label={t("Viewer controls")}
          className="bg-foreground text-background sticky bottom-0 flex shrink-0 items-center gap-0.5 rounded-lg p-0.75 text-sm"
        >
          <ToolbarButton label={t("Zoom out")} onClick={() => stepZoom(-1)}>
            −
          </ToolbarButton>
          <span className="px-1.5 font-mono text-xs">{t("{0}%", Math.round(zoom * 100))}</span>
          <ToolbarButton label={t("Zoom in")} onClick={() => stepZoom(1)}>
            +
          </ToolbarButton>
          <span aria-hidden className="bg-background/25 mx-1 h-4 w-px" />
          <ToolbarButton
            wide
            label={t("Rotate")}
            onClick={() => setRotation((r) => (r + 90) % 360)}
          >
            {t("Rotate")}
          </ToolbarButton>
          <ToolbarButton wide label={t("Download")} onClick={() => void download()}>
            {t("Download")}
          </ToolbarButton>
          {count > 1 ? (
            <>
              <ToolbarButton
                label={t("Previous page")}
                onClick={() => turn(page - 1)}
                disabled={page <= 1}
              >
                <ChevronLeftIcon className="size-3.5" />
              </ToolbarButton>
              <span className="text-background/70 px-1 font-mono text-xs">
                {t("{0} of {1}", page, count)}
              </span>
              <ToolbarButton
                label={t("Next page")}
                onClick={() => turn(page + 1)}
                disabled={page >= count}
              >
                <ChevronRightIcon className="size-3.5" />
              </ToolbarButton>
            </>
          ) : (
            <span className="text-background/70 px-2 font-mono text-xs">
              {t("{0, plural, one {# page} other {# pages}}", count)}
            </span>
          )}
        </div>
      </div>
    </section>
  );
}

function ToolbarButton({
  label,
  onClick,
  disabled = false,
  wide = false,
  children,
}: {
  label: string;
  onClick: () => void;
  disabled?: boolean;
  wide?: boolean;
  children: ReactNode;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      disabled={disabled}
      onClick={onClick}
      className={cn(
        "ui-focus-ring hover:bg-background/15 flex h-6.5 items-center justify-center rounded-md disabled:opacity-40",
        wide ? "px-2" : "w-7",
      )}
    >
      {children}
    </button>
  );
}

function DocumentSheet({
  documentId,
  fileName,
  page,
  size,
  zoom,
  rotation,
  marks,
  selected,
  hot,
  onPoint,
  onSelect,
  wellRef,
  onDownload,
}: {
  documentId: string;
  fileName: string;
  page: number;
  size: PageSize | undefined;
  zoom: number;
  rotation: number;
  marks: ExtractedField[];
  selected: string | null;
  hot: string | null;
  onPoint: (key: string | null) => void;
  onSelect: (key: string) => void;
  wellRef: React.RefObject<HTMLDivElement | null>;
  onDownload: () => void;
}) {
  const t = useT();
  const reduced = useReducedMotion();
  const sheetRef = useRef<HTMLDivElement>(null);
  const [drawn, setDrawn] = useState(false);
  const imageQuery = useQuery({
    ...documentPage.image(documentId, page),
    staleTime: Number.POSITIVE_INFINITY,
    gcTime: PAGE_GC_MS,
    retry: 1,
  });
  const imageUrl = useObjectUrl(imageQuery.data);
  const ready = imageUrl !== null && drawn;
  const target = marks.find((mark) => mark.key === selected) ?? null;

  const box = target?.box ?? null;
  useEffect(() => {
    const view = wellRef.current;
    const sheet = sheetRef.current;
    if (box === null || view === null || sheet === null || typeof view.scrollTo !== "function") {
      return;
    }
    const { left, top } = scrollToBox(box, {
      pageLeft: sheet.offsetLeft,
      pageTop: sheet.offsetTop,
      pageWidth: sheet.offsetWidth,
      pageHeight: sheet.offsetHeight,
      viewWidth: view.clientWidth,
      viewHeight: view.clientHeight,
    });
    view.scrollTo({ left, top, behavior: reduced ? "auto" : "smooth" });
  }, [box, zoom, ready, reduced, wellRef]);

  if (imageQuery.isError) {
    return (
      <div
        role="status"
        className="border-border bg-card flex w-full max-w-130 flex-col items-center gap-2 rounded-md border px-6 py-12 text-center text-sm"
      >
        <p className="text-foreground font-medium">{t("Page {0} couldn't be drawn.", page)}</p>
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
    );
  }

  return (
    <div
      ref={sheetRef}
      className="border-border bg-card relative shrink-0 border transition-transform"
      aria-busy={!ready}
      style={{
        width:
          zoom === 1 ? `min(100%, ${PAGE_WIDTH_PX}px)` : `${Math.round(PAGE_WIDTH_PX * zoom)}px`,
        aspectRatio: ready ? undefined : pageAspect(size),
        transform: rotation ? `rotate(${rotation}deg)` : undefined,
      }}
    >
      {!ready ? <Skeleton className="absolute inset-0 rounded-none" /> : null}
      {imageUrl !== null ? (
        <img
          className="block w-full"
          src={imageUrl}
          alt={t("Page {0} of {1}", page, fileName)}
          decoding="async"
          draggable={false}
          onLoad={() => setDrawn(true)}
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
                  mark.needsLook ? "border-warning bg-warning/10" : "bg-brand/8 border-brand/30",
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
  );
}
