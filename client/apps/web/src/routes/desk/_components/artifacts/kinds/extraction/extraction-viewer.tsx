import { documentPage } from "@/lib/queries/document-page";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  LinkExternal01Icon,
  ZoomInIcon,
  ZoomOutIcon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useObjectUrl } from "@trenova/shared/hooks/use-object-url";
import { useReducedMotion } from "@trenova/shared/hooks/use-reduced-motion";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useRef, useState, type KeyboardEvent, type RefObject } from "react";
import { PageSkeleton } from "./extraction-skeleton";
import type { PageSize } from "./extraction-payload";
import {
  ZOOM_STEPS,
  boxStyle,
  pageAspect,
  scrollToBox,
  sheetWidth,
  stepZoom,
  type PageBox,
  type Zoom,
} from "./page-geometry";

/** A value found on the page, as the page draws it. */
export type PageMark = {
  key: string;
  label: string;
  value: string;
  doubtful: boolean;
  box: PageBox;
};

/** At most this many pages get a button each; past it the switcher says which page of how many. */
const PAGE_BUTTONS = 8;

/** How long a page the server drew is kept once no viewer shows it. */
const PAGE_GC_MS = 5 * 60 * 1000;

type ViewerProps = {
  documentId: string;
  fileName: string;
  page: number;
  pageCount: number;
  pages: readonly PageSize[];
  /** Pages holding a field to check, marked in the switcher. */
  pagesToCheck: ReadonlySet<number>;
  /** Pages to read ahead of being turned to, such as the next one. */
  readAhead: readonly number[];
  onPage: (page: number) => void;
  marks: readonly PageMark[];
  hot: string | null;
  selected: string | null;
  /** The field pointed at on the page, or null when the pointer leaves its box. */
  onPoint: (key: string | null) => void;
  /** A box pressed on the page. */
  onSelect: (key: string) => void;
  onOpenFile: (() => void) | null;
};

/**
 * The document as it was read, a page at a time, with a box over each value found on it. The
 * page opens fitted to the viewer's width, never narrower than a readable page. The selected value is scrolled to the middle of the viewer on whatever page
 * it was read from, and Page Up, Page Down, plus, minus and 0 turn and zoom from the keyboard.
 */
export function ExtractionViewer(props: ViewerProps) {
  const { documentId, page, pageCount, onPage, pagesToCheck, readAhead } = props;
  const t = useT();
  const queryClient = useQueryClient();
  const wellRef = useRef<HTMLDivElement>(null);
  const [zoom, setZoom] = useState<Zoom>(ZOOM_STEPS[0]);
  const most = ZOOM_STEPS[ZOOM_STEPS.length - 1];

  useEffect(() => {
    for (const ahead of readAhead) {
      if (ahead < 1 || ahead > pageCount) continue;
      void queryClient.prefetchQuery({
        ...documentPage.image(documentId, ahead),
        staleTime: Number.POSITIVE_INFINITY,
        gcTime: PAGE_GC_MS,
      });
    }
  }, [documentId, pageCount, queryClient, readAhead]);

  const turn = (to: number) => {
    if (to >= 1 && to <= pageCount && to !== page) onPage(to);
  };
  const onKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.altKey || event.ctrlKey || event.metaKey) return;
    const actions: Record<string, () => void> = {
      PageDown: () => turn(page + 1),
      PageUp: () => turn(page - 1),
      "+": () => setZoom(stepZoom(zoom, 1)),
      "=": () => setZoom(stepZoom(zoom, 1)),
      "-": () => setZoom(stepZoom(zoom, -1)),
      "0": () => setZoom(ZOOM_STEPS[0]),
    };
    const action = actions[event.key];
    if (action) {
      event.preventDefault();
      action();
    }
  };

  return (
    <section className="dk-ex-viewer" aria-label={t("Document")}>
      <div className="dk-ex-bar">
        <Button
          variant="ghost"
          size="icon-xs"
          aria-label={t("Previous page")}
          disabled={page <= 1}
          onClick={() => turn(page - 1)}
        >
          <ChevronLeftIcon className="size-3.5" />
        </Button>
        {pageCount <= PAGE_BUTTONS ? (
          <span className="dk-ex-pages">
            {Array.from({ length: pageCount }, (_, index) => {
              const number = index + 1;
              const check = pagesToCheck.has(number);
              return (
                <Button
                  key={number}
                  variant="ghost"
                  size="icon-xs"
                  className="dk-ex-pg"
                  aria-current={number === page ? "page" : undefined}
                  aria-label={
                    check ? t("Page {0}, with fields to check", number) : t("Page {0}", number)
                  }
                  onClick={() => turn(number)}
                >
                  {number}
                  {check && <span className="dk-ex-pdot" aria-hidden />}
                </Button>
              );
            })}
          </span>
        ) : (
          <span className="dk-ex-pno">{t("Page {0} of {1}", page, pageCount)}</span>
        )}
        <Button
          variant="ghost"
          size="icon-xs"
          aria-label={t("Next page")}
          disabled={page >= pageCount}
          onClick={() => turn(page + 1)}
        >
          <ChevronRightIcon className="size-3.5" />
        </Button>
        <span className="flex-1" />
        <Button
          variant="ghost"
          size="icon-xs"
          aria-label={t("Zoom out")}
          disabled={zoom === ZOOM_STEPS[0]}
          onClick={() => setZoom(stepZoom(zoom, -1))}
        >
          <ZoomOutIcon className="size-3.5" />
        </Button>
        <Button
          variant="ghost"
          size="xxs"
          className="dk-ex-zoom"
          aria-label={t("Fit the page to the width")}
          disabled={zoom === ZOOM_STEPS[0]}
          onClick={() => setZoom(ZOOM_STEPS[0])}
        >
          {zoom === ZOOM_STEPS[0] ? t("Fit") : t("{0}%", Math.round(zoom * 100))}
        </Button>
        <Button
          variant="ghost"
          size="icon-xs"
          aria-label={t("Zoom in")}
          disabled={zoom === most}
          onClick={() => setZoom(stepZoom(zoom, 1))}
        >
          <ZoomInIcon className="size-3.5" />
        </Button>
      </div>
      <span className="sr-only" aria-live="polite">
        {t("Page {0} of {1}", page, pageCount)}
      </span>
      <div
        ref={wellRef}
        className="dk-ex-well ui-focus-ring"
        role="region"
        tabIndex={0}
        aria-label={t("Page {0} of {1}. Page Up and Page Down turn pages.", page, pageCount)}
        onKeyDown={onKeyDown}
      >
        <ExtractionPage key={page} {...props} zoom={zoom} wellRef={wellRef} />
      </div>
    </section>
  );
}

type PageProps = ViewerProps & {
  zoom: Zoom;
  wellRef: RefObject<HTMLDivElement | null>;
};

/**
 * One page in its scrolling frame. Until the server's drawing arrives the page stands in its
 * own shape: page 1 as the upload's stored preview, blurred, with the boxes already on it,
 * and any other page as its skeleton. Once drawn, the sheet is the image at its own size, so
 * the boxes over it are exactly where its text is.
 */
function ExtractionPage({
  documentId,
  fileName,
  page,
  pages,
  marks,
  hot,
  selected,
  onPoint,
  onSelect,
  onOpenFile,
  zoom,
  wellRef,
}: PageProps) {
  const t = useT();
  const reduced = useReducedMotion();
  const sheetRef = useRef<HTMLDivElement>(null);
  const [drawn, setDrawn] = useState(false);
  const [broken, setBroken] = useState(false);
  const imageQuery = useQuery({
    ...documentPage.image(documentId, page),
    staleTime: Number.POSITIVE_INFINITY,
    gcTime: PAGE_GC_MS,
    retry: 1,
  });
  const previewQuery = useQuery({
    ...documentPage.preview(documentId),
    enabled: page === 1 && imageQuery.data === undefined && !imageQuery.isError,
    staleTime: Number.POSITIVE_INFINITY,
    gcTime: PAGE_GC_MS,
    retry: false,
  });
  const imageUrl = useObjectUrl(imageQuery.data);
  const previewUrl = useObjectUrl(page === 1 ? previewQuery.data : undefined);
  const ready = imageUrl !== null && drawn;
  const target = marks.find((mark) => mark.key === selected) ?? null;
  const size = pages.find((candidate) => candidate.number === page);

  const boxX = target?.box.x ?? -1;
  const boxY = target?.box.y ?? -1;
  const boxW = target?.box.w ?? 0;
  const boxH = target?.box.h ?? 0;

  useEffect(() => {
    const view = wellRef.current;
    const sheet = sheetRef.current;
    if (boxX < 0 || view === null || sheet === null) return;
    const box = { x: boxX, y: boxY, w: boxW, h: boxH };
    const centre = (behavior: ScrollBehavior) => {
      if (typeof view.scrollTo !== "function") return;
      const { left, top } = scrollToBox(box, {
        pageLeft: sheet.offsetLeft,
        pageTop: sheet.offsetTop,
        pageWidth: sheet.offsetWidth,
        pageHeight: sheet.offsetHeight,
        viewWidth: view.clientWidth,
        viewHeight: view.clientHeight,
      });
      view.scrollTo({ left, top, behavior });
    };
    centre(reduced ? "auto" : "smooth");
    if (typeof ResizeObserver === "undefined") return;
    let first = true;
    const observer = new ResizeObserver(() => {
      if (first) {
        first = false;
        return;
      }
      centre("auto");
    });
    observer.observe(view);
    return () => observer.disconnect();
  }, [selected, boxX, boxY, boxW, boxH, zoom, ready, reduced, wellRef]);

  if (imageQuery.isError || broken) {
    return (
      <div className="dk-ex-pfail" role="status">
        <p>{t("Page {0} could not be drawn.", page)}</p>
        <p className="dk-ex-pfail-sub">
          {t("The fields still read. Open the file to see this page.")}
        </p>
        <div className="dk-ex-pfail-acts">
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              setBroken(false);
              void imageQuery.refetch();
            }}
          >
            {t("Try again")}
          </Button>
          {onOpenFile && (
            <Button variant="outline" size="sm" onClick={onOpenFile}>
              {t("Open file")}
              <LinkExternal01Icon className="size-3.5" />
            </Button>
          )}
        </div>
      </div>
    );
  }

  const placeholder = !ready && previewUrl !== null;
  return (
    <div
      ref={sheetRef}
      className="dk-ex-sheet"
      data-state={ready ? "ready" : placeholder ? "preview" : "loading"}
      data-extraction-page={page}
      aria-busy={!ready}
      style={{
        width: sheetWidth(zoom),
        aspectRatio: ready ? undefined : pageAspect(size),
      }}
    >
      {!ready && !placeholder && <PageSkeleton />}
      {placeholder && <img className="dk-ex-preview" src={previewUrl} alt="" draggable={false} />}
      {imageUrl !== null && (
        <img
          className="dk-ex-img"
          src={imageUrl}
          alt={t("Page {0} of {1}", page, fileName)}
          decoding="async"
          draggable={false}
          onLoad={() => setDrawn(true)}
          onError={() => setBroken(true)}
        />
      )}
      {(ready || placeholder) && (
        <div className="dk-ex-marks">
          {marks.map((mark) => (
            // oxlint-disable-next-line jsx-a11y/click-events-have-key-events, jsx-a11y/no-static-element-interactions -- a box is a pointer shortcut to its field, whose row in the list is the keyboard's way in
            <span
              key={mark.key}
              aria-hidden
              className={cn(
                "dk-ex-box",
                mark.doubtful && "dk-warn",
                hot === mark.key && "dk-hot",
                selected === mark.key && "dk-on",
              )}
              title={t("{0}: {1}", mark.label, mark.value)}
              style={boxStyle(mark.box)}
              onMouseEnter={() => onPoint(mark.key)}
              onMouseLeave={() => onPoint(null)}
              onClick={() => onSelect(mark.key)}
            />
          ))}
          {target && (
            <span
              aria-hidden
              className={cn("dk-ex-flag", target.doubtful && "dk-warn")}
              style={{ left: boxStyle(target.box).left, top: boxStyle(target.box).top }}
            >
              {target.label}
            </span>
          )}
        </div>
      )}
    </div>
  );
}
