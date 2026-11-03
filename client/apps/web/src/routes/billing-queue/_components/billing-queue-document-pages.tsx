"use no memo";
import { useDeferredMeasure } from "@/hooks/use-deferred-measure";
import type { ExtractedField, PageSize } from "@/lib/document-reading/extraction-payload";
import { pageAspect } from "@/lib/document-reading/page-geometry";
import type { PageImageSource } from "@/lib/queries/document-page";
import { toast } from "@trenova/shared/components/ui/toast";
import { useElementSize } from "@trenova/shared/hooks/use-element-size";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useT } from "@trenova/shared/i18n/use-t";
import { api } from "@trenova/shared/lib/api";
import { downloadBlob } from "@trenova/shared/lib/utils";
import { useVirtualizer, type VirtualItem } from "@tanstack/react-virtual";
import { useEffect, useRef, useState, type KeyboardEvent } from "react";
import { BillingQueuePageRail } from "./billing-queue-page-rail";
import { BillingQueuePageSheet, sheetBox } from "./billing-queue-page-sheet";
import { BillingQueuePageToolbar } from "./billing-queue-page-toolbar";
import {
  partRange,
  type PageFlag,
  type PartEditing,
  type ViewerPart,
} from "./billing-queue-page-viewer-types";
import {
  FIT_WIDTH,
  sheetWidthFor,
  stepZoom,
  wheelZoom,
  type PageZoom,
} from "./billing-queue-page-zoom";

/** The room under each page: its number and the gap to the next page. */
const PAGE_CHROME = 36;
/** The room a document's name takes above its first page. */
const DIVIDER = 30;
const COLUMN_GUTTER = 48;
const COLUMN_TOP = 24;
/** Room under the last page, so the toolbar never covers it. */
const COLUMN_BOTTOM = 72;
/** The page read is the one crossing this far down the viewer. */
const READING_LINE = 0.35;

const NO_FIELDS: readonly ExtractedField[] = [];
const NO_PAGES: readonly PageSize[] = [];
const NO_PARTS: readonly ViewerPart[] = [];
const NO_FLAGS: ReadonlyMap<number, PageFlag> = new Map();
const ignore = () => {};

/** The page whose row crosses `offset`, from rows laid out top to bottom. */
export function pageAtOffset(rows: readonly Pick<VirtualItem, "start" | "end">[], offset: number) {
  let low = 0;
  let high = rows.length - 1;
  while (low < high) {
    const middle = Math.ceil((low + high) / 2);
    if (rows[middle].start <= offset) low = middle;
    else high = middle - 1;
  }
  return low + 1;
}

/**
 * A file's pages as the server draws them, from a stored document or a packet assembled on the
 * fly, stacked in one column that scrolls like a PDF reader, with every page as a thumbnail in a
 * rail beside it. Only the pages near the viewport are mounted and fetched, in the column and in
 * the rail, so a hundred-page packet costs what its first screen does.
 *
 * The page crossing the reading line is the current one: the rail outlines it and the toolbar
 * counts it. Turning a page from the rail, the toolbar, the page count or Page Up and Page Down
 * scrolls to it. Pages fit the width or the whole page, or take a scale from plus, minus,
 * Ctrl + scroll or a pinch; 0 fits the width again. Each page turns on its own, or all together,
 * in the view; given editing, a page turns in the file itself, as the file's own drawing shows.
 *
 * Given parts, each document's name and pages head its first page and its pages in the rail;
 * given flags, a page says what to check across its top. When fields are given, each page draws a
 * box over every value Nova read from it, and the selected one is scrolled to the middle.
 */
export function BillingQueueDocumentPages({
  source,
  fileName,
  pageCount,
  pages = NO_PAGES,
  fields = NO_FIELDS,
  page,
  onPage,
  selected = null,
  hot = null,
  onPoint = ignore,
  onSelect = ignore,
  parts = NO_PARTS,
  flags = NO_FLAGS,
  editing = null,
}: {
  source: PageImageSource;
  fileName: string;
  pageCount: number;
  pages?: readonly PageSize[];
  fields?: readonly ExtractedField[];
  page: number;
  onPage: (page: number) => void;
  selected?: string | null;
  hot?: string | null;
  onPoint?: (key: string | null) => void;
  onSelect?: (key: string) => void;
  parts?: readonly ViewerPart[];
  flags?: ReadonlyMap<number, PageFlag>;
  editing?: PartEditing | null;
}) {
  const t = useT();
  const wellRef = useRef<HTMLDivElement | null>(null);
  const [measureWell, wellSize] = useElementSize<HTMLDivElement>();
  const [zoom, setZoom] = useState<PageZoom>(FIT_WIDTH);
  const [turnAll, setTurnAll] = useState(0);
  const [turned, setTurned] = useState<ReadonlyMap<number, number>>(new Map());
  const count = Math.max(1, pageCount);
  const sizes = new Map(pages.map((size) => [size.number, size]));
  const starts = new Map(parts.map((part) => [part.firstPage, part]));
  // A file whose pages can be changed turns them in the file; its view turns nothing itself.
  const rotationOf = (number: number) =>
    editing === null ? (turnAll + (turned.get(number) ?? 0)) % 360 : 0;
  const letter = pageAspect(null);
  const sheetWidth = sheetWidthFor(
    zoom,
    {
      width: wellSize.width || 600,
      height: wellSize.height || 800,
      gutter: COLUMN_GUTTER,
      chrome: COLUMN_TOP + PAGE_CHROME + DIVIDER,
    },
    turnAll % 180 === 0 ? letter : 1 / letter,
  );
  const shown = useRef(page);
  const held = useRef(false);

  // oxlint-disable-next-line react/incompatible-library -- TanStack Virtual is incompatible with the React Compiler; file is already "use no memo", which oxlint 1.89 ignores
  const virtualizer = useVirtualizer({
    count,
    getScrollElement: () => wellRef.current,
    estimateSize: (index) => {
      const number = index + 1;
      const { height } = sheetBox(sheetWidth, pageAspect(sizes.get(number)), rotationOf(number));
      return Math.round(height) + PAGE_CHROME + (starts.has(number) ? DIVIDER : 0);
    },
    overscan: 2,
    paddingStart: COLUMN_TOP,
    paddingEnd: COLUMN_BOTTOM,
    initialRect: { width: 600, height: 800 },
  });
  const measureRow = useDeferredMeasure(virtualizer);

  useEffect(() => {
    virtualizer.measure();
    held.current = true;
    virtualizer.scrollToIndex(shown.current - 1, { align: "start" });
  }, [sheetWidth, turnAll, parts.length, virtualizer]);

  useEffect(() => {
    if (page === shown.current) return;
    shown.current = page;
    held.current = true;
    virtualizer.scrollToIndex(page - 1, { align: "start" });
  }, [page, virtualizer]);

  const onWheel = useLatestCallback((event: WheelEvent) => {
    held.current = false;
    if (!event.ctrlKey && !event.metaKey) return;
    event.preventDefault();
    setZoom(wheelZoom(sheetWidth, event.deltaY));
  });
  useEffect(() => {
    const well = wellRef.current;
    if (well === null) return;
    well.addEventListener("wheel", onWheel, { passive: false });
    return () => well.removeEventListener("wheel", onWheel);
  }, [onWheel]);

  // A page turned to stays current while the column scrolls to it, even where the column
  // cannot bring it to the top (the last pages); the person's own scrolling lets go of it.
  const release = () => {
    held.current = false;
  };
  const onScroll = () => {
    const well = wellRef.current;
    if (well === null || held.current) return;
    const atEnd = well.scrollTop > 0 && well.scrollTop + well.clientHeight >= well.scrollHeight - 2;
    const reading = atEnd
      ? count
      : pageAtOffset(
          virtualizer.measurementsCache,
          well.scrollTop + well.clientHeight * READING_LINE,
        );
    if (reading === shown.current) return;
    shown.current = reading;
    onPage(reading);
  };

  const turn = (to: number) => {
    if (to >= 1 && to <= count && to !== page) onPage(to);
  };
  const rotate = (number: number) => {
    setTurned((current) => new Map(current).set(number, ((current.get(number) ?? 0) + 90) % 360));
  };
  const [downloading, setDownloading] = useState(false);
  const download = async () => {
    if (downloading) return;
    setDownloading(true);
    try {
      const file = await api.download(source.downloadEndpoint, fileName);
      downloadBlob(file.blob, file.fileName || fileName);
    } catch {
      toast.error(t("The file couldn't be downloaded"));
    } finally {
      setDownloading(false);
    }
  };
  const onKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    release();
    if (event.altKey || event.ctrlKey || event.metaKey) return;
    const actions: Record<string, () => void> = {
      PageDown: () => turn(page + 1),
      PageUp: () => turn(page - 1),
      "+": () => setZoom(stepZoom(sheetWidth, 1)),
      "=": () => setZoom(stepZoom(sheetWidth, 1)),
      "-": () => setZoom(stepZoom(sheetWidth, -1)),
      "0": () => setZoom(FIT_WIDTH),
    };
    const action = actions[event.key];
    if (action) {
      event.preventDefault();
      action();
    }
  };

  return (
    <section aria-label={t("Document viewer")} className="bg-muted flex h-full min-h-0">
      <BillingQueuePageRail
        source={source}
        pageCount={count}
        sizes={sizes}
        page={page}
        onPage={turn}
        parts={parts}
        flags={flags}
        editing={editing}
      />
      <div ref={measureWell} className="relative flex min-h-0 min-w-0 flex-1 flex-col">
        <div
          ref={wellRef}
          role="region"
          aria-label={t("Page {0} of {1}. Page Up and Page Down turn pages.", page, count)}
          // oxlint-disable-next-line jsx-a11y/no-noninteractive-tabindex -- the page column scrolls and takes the page and zoom keys
          tabIndex={0}
          onKeyDown={onKeyDown}
          onScroll={onScroll}
          onTouchMove={release}
          onPointerDown={release}
          className="ui-inset-focus-ring min-h-0 flex-1 overflow-auto"
        >
          <div
            className="relative"
            style={{
              height: virtualizer.getTotalSize(),
              width: `max(100%, ${sheetWidth + COLUMN_GUTTER}px)`,
            }}
          >
            {virtualizer.getVirtualItems().map((item) => {
              const number = item.index + 1;
              const starting = starts.get(number);
              return (
                <div
                  key={item.key}
                  data-index={item.index}
                  ref={measureRow}
                  className="absolute inset-x-0 top-0 flex flex-col items-center"
                  style={{ transform: `translateY(${item.start}px)` }}
                >
                  {starting ? (
                    <div
                      className="text-foreground-muted flex h-7.5 items-center gap-2.5 pb-2 text-xs"
                      style={{ width: sheetWidth }}
                    >
                      <span className="text-foreground truncate font-medium">{starting.name}</span>
                      <span className="shrink-0 font-mono text-2xs">
                        {starting.firstPage === starting.lastPage
                          ? t("Page {0}", starting.firstPage)
                          : t("Pages {0}", partRange(starting))}
                      </span>
                      <span aria-hidden className="bg-border-strong h-px flex-1" />
                    </div>
                  ) : null}
                  <BillingQueuePageSheet
                    source={source}
                    fileName={fileName}
                    number={number}
                    size={sizes.get(number)}
                    width={sheetWidth}
                    rotation={rotationOf(number)}
                    marks={fields.filter((field) => field.page === number && field.box !== null)}
                    selected={selected}
                    hot={hot}
                    onPoint={onPoint}
                    onSelect={onSelect}
                    wellRef={wellRef}
                    onDownload={() => void download()}
                    flag={flags.get(number)}
                    onRotate={
                      editing === null
                        ? () => rotate(number)
                        : editing.rotatable(number) && !editing.busy
                          ? () => editing.onRotate(number)
                          : null
                    }
                  />
                  <span
                    aria-hidden
                    className="text-foreground-muted pt-1.5 pb-4 font-mono text-2xs"
                  >
                    {number}
                  </span>
                </div>
              );
            })}
          </div>
        </div>
        <BillingQueuePageToolbar
          zoom={zoom}
          sheetWidth={sheetWidth}
          page={page}
          pageCount={count}
          onZoom={setZoom}
          onStepZoom={(direction) => setZoom(stepZoom(sheetWidth, direction))}
          onRotateAll={
            editing === null ? () => setTurnAll((current) => (current + 90) % 360) : null
          }
          onDownload={() => void download()}
          downloading={downloading}
          onTurn={turn}
        />
      </div>
    </section>
  );
}
