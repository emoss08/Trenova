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

/** At most this many pages are drawn as thumbnails; past it the rail says which page of how many. */
const PAGE_THUMBS = 8;

/** How long a page the server drew is kept once no viewer shows it. */
const PAGE_GC_MS = 5 * 60 * 1000;

type ViewerProps = {
  documentId: string;
  fileName: string;
  page: number;
  pageCount: number;
  pages: readonly PageSize[];
  /** How many fields to check each page holds, for the count on its thumbnail. */
  checksByPage: ReadonlyMap<number, number>;
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
  /** The section, so the card can bring it into view when a value is found from a snippet. */
  sectionRef?: RefObject<HTMLElement | null>;
};

/**
 * The document as it was read, a page at a time, with a mark over each value found on it:
 * a faint tint where a value was read, an amber outline where one needs a person, and the
 * selected value outlined with its name above it. The pages stand beside it as thumbnails,
 * each counting what it holds to check. The page opens fitted to the frame, never narrower
 * than a readable page, and the selected value is scrolled to the middle of the frame on
 * whatever page it was read from; Page Up, Page Down, plus, minus and 0 turn and zoom.
 */
export function ExtractionViewer(props: ViewerProps) {
  const { sectionRef, documentId, page, pageCount, onPage, checksByPage, readAhead, onOpenFile } =
    props;
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
    <section ref={sectionRef} className="dk-rec-sec dk-ex-doc" aria-label={t("Document")}>
      <div className="dk-rec-sh">
        <h3>{t("The document")}</h3>
        <span className="dk-ex-doc-acts">
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
          {onOpenFile && (
            <Button variant="outline" size="xs" className="dk-ex-open" onClick={onOpenFile}>
              {t("Open file")}
              <LinkExternal01Icon className="size-3" />
            </Button>
          )}
        </span>
      </div>
      <span className="sr-only" aria-live="polite">
        {t("Page {0} of {1}", page, pageCount)}
      </span>
      <div className={cn("dk-ex-doc-body", pageCount < 2 && "dk-one")}>
        {pageCount > 1 && pageCount <= PAGE_THUMBS && (
          <div className="dk-ex-thumbs" role="group" aria-label={t("Pages")}>
            {Array.from({ length: pageCount }, (_, index) => (
              <PageThumb
                key={index + 1}
                documentId={documentId}
                number={index + 1}
                size={props.pages.find((candidate) => candidate.number === index + 1)}
                checks={checksByPage.get(index + 1) ?? 0}
                current={index + 1 === page}
                onTurn={() => turn(index + 1)}
              />
            ))}
          </div>
        )}
        {pageCount > PAGE_THUMBS && (
          <div className="dk-ex-pager">
            <Button
              variant="ghost"
              size="icon-xs"
              aria-label={t("Previous page")}
              disabled={page <= 1}
              onClick={() => turn(page - 1)}
            >
              <ChevronLeftIcon className="size-3.5" />
            </Button>
            <span>{t("Page {0} of {1}", page, pageCount)}</span>
            <Button
              variant="ghost"
              size="icon-xs"
              aria-label={t("Next page")}
              disabled={page >= pageCount}
              onClick={() => turn(page + 1)}
            >
              <ChevronRightIcon className="size-3.5" />
            </Button>
          </div>
        )}
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
      </div>
      <p className="dk-ex-legend">
        <span>
          <i aria-hidden className="dk-ex-lg" />
          {t("Read")}
        </span>
        <span>
          <i aria-hidden className="dk-ex-lg dk-warn" />
          {t("To check")}
        </span>
        <span className="dk-ex-legend-hint">{t("Choose a value to find it on the page")}</span>
      </p>
    </section>
  );
}

/** A page as a thumbnail beside the one shown, with how many fields on it are to be checked. */
function PageThumb({
  documentId,
  number,
  size,
  checks,
  current,
  onTurn,
}: {
  documentId: string;
  number: number;
  size: PageSize | undefined;
  checks: number;
  current: boolean;
  onTurn: () => void;
}) {
  const t = useT();
  const query = useQuery({
    ...documentPage.image(documentId, number),
    staleTime: Number.POSITIVE_INFINITY,
    gcTime: PAGE_GC_MS,
    retry: 1,
  });
  const url = useObjectUrl(query.data);
  return (
    <Button
      variant="bare"
      size="bare"
      className="dk-ex-thumb ui-focus-ring"
      aria-current={current ? "page" : undefined}
      aria-label={
        checks > 0
          ? t(
              "Page {0}, {1, plural, one {# field to check} other {# fields to check}}",
              number,
              checks,
            )
          : t("Page {0}", number)
      }
      onClick={onTurn}
    >
      <span
        className={cn("dk-ex-thumb-sheet", url === null && "ui-shimmer")}
        style={{ aspectRatio: pageAspect(size) }}
      >
        {url !== null && <img src={url} alt="" draggable={false} />}
      </span>
      <span className="dk-ex-thumb-n" aria-hidden>
        {number}
        {checks > 0 && <i>{checks}</i>}
      </span>
    </Button>
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
