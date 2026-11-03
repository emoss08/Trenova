import { VirtualRows, type VirtualRow, type VirtualRowsHandle } from "@/components/virtual-rows";
import type { PageSize } from "@/lib/document-reading/extraction-payload";
import { pageAspect } from "@/lib/document-reading/page-geometry";
import { PAGE_IMAGE_CACHE, type PageImageSource } from "@/lib/queries/document-page";
import { AlertTriangleIcon, GripVerticalIcon, XIcon } from "@trenova/shared/components/icons";
import { useObjectUrl } from "@trenova/shared/hooks/use-object-url";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { useEffect, useRef, useState, type DragEvent, type KeyboardEvent } from "react";
import {
  nextFlaggedPage,
  partRange,
  type PageFlag,
  type PartEditing,
  type ViewerPart,
} from "./billing-queue-page-viewer-types";

/** A thumbnail's width in the rail, in pixels. */
const THUMB_WIDTH = 64;
/** The room under and around a thumbnail: its number and the gap to the next. */
const THUMB_CHROME = 34;

const pageKey = (page: number) => `page:${page}`;

/**
 * Every page as a thumbnail down the side of the viewer, mounted only near the rail's viewport
 * so a hundred-page file fetches only the pages scrolled to. Given parts, the pages stand under
 * their document's name and page range, and a part that can change is dragged by its handle (or
 * moved with the arrow keys on it) and taken out with its cross. A flagged page carries a mark,
 * and the count at the top goes to the next one. The page being read is outlined and kept in
 * view; clicking a thumbnail goes to its page, and the arrow keys, Home and End move through them
 * with the focus.
 */
export function BillingQueuePageRail({
  source,
  pageCount,
  sizes,
  page,
  onPage,
  parts,
  flags,
  editing,
}: {
  source: PageImageSource;
  pageCount: number;
  sizes: ReadonlyMap<number, PageSize>;
  page: number;
  onPage: (page: number) => void;
  parts: readonly ViewerPart[];
  flags: ReadonlyMap<number, PageFlag>;
  editing: PartEditing | null;
}) {
  const t = useT();
  const railRef = useRef<HTMLDivElement>(null);
  const rowsRef = useRef<VirtualRowsHandle>(null);
  const focusTurn = useRef(false);
  const [dragging, setDragging] = useState<string | null>(null);
  const [dropOn, setDropOn] = useState<{ key: string; after: boolean } | null>(null);
  const grouped = parts.length > 0;

  useEffect(() => {
    rowsRef.current?.scrollToKey(pageKey(page), { align: "auto" });
    if (!focusTurn.current) return;
    focusTurn.current = false;
    const frame = requestAnimationFrame(() => {
      railRef.current?.querySelector<HTMLElement>(`[data-thumb="${page}"]`)?.focus();
    });
    return () => cancelAnimationFrame(frame);
  }, [page]);

  const onKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (!(event.target instanceof HTMLElement) || !event.target.dataset.thumb) return;
    const moves: Record<string, number> = {
      ArrowDown: page + 1,
      ArrowUp: page - 1,
      Home: 1,
      End: pageCount,
    };
    const to = moves[event.key];
    if (to === undefined) return;
    event.preventDefault();
    if (to < 1 || to > pageCount || to === page) return;
    focusTurn.current = true;
    onPage(to);
  };

  const thumbRow = (number: number): VirtualRow => ({
    key: pageKey(number),
    render: () => (
      <div className="flex justify-center px-4 pb-3">
        <PageThumb
          source={source}
          number={number}
          size={sizes.get(number)}
          current={number === page}
          flagged={flags.has(number)}
          onTurn={() => onPage(number)}
        />
      </div>
    ),
  });

  const fromIndex = parts.findIndex((part) => part.key === dragging);
  // A part dropped on another takes its place: the others shift toward where it came from.
  const drop = (target: ViewerPart, index: number) => (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    const moving = dragging;
    setDragging(null);
    setDropOn(null);
    if (moving === null || moving === target.key || editing === null) return;
    editing.onMove(moving, index);
  };

  const rows: VirtualRow[] = grouped
    ? parts.flatMap((part, index) => {
        const reading = page >= part.firstPage && page <= part.lastPage;
        const droppable = editing !== null && !editing.busy && part.movable && dragging !== null;
        const header: VirtualRow = {
          key: `part:${part.key}`,
          sticky: true,
          render: () => (
            // oxlint-disable-next-line jsx-a11y/no-static-element-interactions -- a drop target for the drag handle, whose arrow keys are the keyboard's way to move a part
            <div
              data-part={part.key}
              onDragOver={(event) => {
                if (!droppable || dragging === part.key) return;
                event.preventDefault();
                setDropOn({ key: part.key, after: fromIndex < index });
              }}
              onDragLeave={() =>
                setDropOn((current) => (current?.key === part.key ? null : current))
              }
              onDrop={drop(part, index)}
              className={cn(
                "group bg-card flex items-center gap-1 px-2.5 pt-2 pb-1.5 text-2xs",
                index > 0 && "border-border border-t",
                dropOn?.key === part.key &&
                  (dropOn.after ? "border-brand border-b-2" : "border-brand border-t-2"),
                dragging === part.key && "opacity-40",
              )}
            >
              {editing !== null && part.movable ? (
                <button
                  type="button"
                  draggable={!editing.busy}
                  disabled={editing.busy}
                  aria-label={t("Move {0}. Use the arrow keys to move it.", part.name)}
                  onDragStart={(event) => {
                    event.dataTransfer.effectAllowed = "move";
                    event.dataTransfer.setData("text/plain", part.key);
                    setDragging(part.key);
                  }}
                  onDragEnd={() => {
                    setDragging(null);
                    setDropOn(null);
                  }}
                  onKeyDown={(event) => {
                    const step = { ArrowUp: -1, ArrowDown: 1 }[event.key];
                    if (step === undefined) return;
                    event.preventDefault();
                    event.stopPropagation();
                    const to = index + step;
                    if (to < 0 || to >= parts.length || !parts[to].movable) return;
                    editing.onMove(part.key, to);
                  }}
                  className="ui-focus-ring text-foreground-muted hover:text-foreground hover:bg-muted grid size-4 shrink-0 cursor-grab place-items-center rounded-xs opacity-0 group-hover:opacity-100 focus-visible:opacity-100 disabled:cursor-default"
                >
                  <GripVerticalIcon className="size-3" />
                </button>
              ) : null}
              <span
                className={cn(
                  "min-w-0 flex-1 truncate font-medium",
                  reading ? "text-brand" : "text-foreground",
                )}
                title={part.name}
              >
                {part.name}
              </span>
              <span className="text-foreground-muted shrink-0 font-mono">{partRange(part)}</span>
              {editing !== null && part.removable ? (
                <button
                  type="button"
                  disabled={editing.busy}
                  aria-label={t("Remove {0} from the packet", part.name)}
                  onClick={() => void editing.onRemove(part.key)}
                  className="ui-focus-ring text-foreground-muted hover:text-foreground hover:bg-muted grid size-4 shrink-0 place-items-center rounded-xs opacity-0 group-hover:opacity-100 focus-visible:opacity-100"
                >
                  <XIcon className="size-3" />
                </button>
              ) : null}
            </div>
          ),
        };
        const thumbs = Array.from({ length: part.lastPage - part.firstPage + 1 }, (_, offset) =>
          thumbRow(part.firstPage + offset),
        );
        return [header, ...thumbs];
      })
    : Array.from({ length: pageCount }, (_, index) => thumbRow(index + 1));

  const flagged = flags.size;
  const nextFlag = nextFlaggedPage(flags, page);

  return (
    // oxlint-disable-next-line jsx-a11y/no-static-element-interactions -- the arrow keys move between the thumbnails this wraps, each a button
    <div
      ref={railRef}
      onKeyDown={onKeyDown}
      aria-busy={editing?.busy || undefined}
      className={cn(
        "border-border bg-card hidden h-full shrink-0 flex-col border-r sm:flex",
        grouped ? "w-38" : "w-24",
        editing?.busy && "opacity-60",
      )}
    >
      {flagged > 0 && nextFlag !== null ? (
        <div className="flex shrink-0 justify-center px-2 pt-3 pb-1">
          <button
            type="button"
            onClick={() => onPage(nextFlag)}
            className="ui-focus-ring bg-warning-subtle text-warning-subtle-foreground hover:bg-warning-subtle-hover inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-2xs font-medium"
          >
            <AlertTriangleIcon className="size-3" />
            {t("{0} to check", flagged)}
          </button>
        </div>
      ) : (
        <div className="h-2 shrink-0" />
      )}
      <VirtualRows
        rows={rows}
        role="group"
        aria-label={t("Pages")}
        handleRef={rowsRef}
        estimateSize={Math.round(THUMB_WIDTH / pageAspect(null)) + THUMB_CHROME}
        initialHeight={640}
        overscan={4}
        mask="card"
        className="min-h-0 flex-1"
      />
    </div>
  );
}

function PageThumb({
  source,
  number,
  size,
  current,
  flagged,
  onTurn,
}: {
  source: PageImageSource;
  number: number;
  size: PageSize | undefined;
  current: boolean;
  flagged: boolean;
  onTurn: () => void;
}) {
  const t = useT();
  const query = useQuery({
    ...(source.thumbnail ?? source.image)(number),
    ...PAGE_IMAGE_CACHE,
    retry: 1,
  });
  const url = useObjectUrl(query.data);

  return (
    <button
      type="button"
      data-thumb={number}
      tabIndex={current ? 0 : -1}
      aria-current={current ? "page" : undefined}
      aria-label={flagged ? t("Page {0}, to check", number) : t("Page {0}", number)}
      onClick={onTurn}
      className="group ui-focus-ring relative flex flex-col items-center gap-1.5 rounded-sm"
      style={{ width: THUMB_WIDTH }}
    >
      <span
        className={cn(
          "border-border bg-card relative block w-full overflow-hidden rounded-xs border transition-colors",
          current ? "border-brand ring-brand ring-2" : "group-hover:border-border-strong",
          url === null && !query.isError && "ui-shimmer",
        )}
        style={{ aspectRatio: pageAspect(size) }}
      >
        {url !== null ? (
          <img
            className="absolute inset-0 size-full object-contain"
            src={url}
            alt=""
            decoding="async"
            draggable={false}
          />
        ) : null}
      </span>
      {flagged ? (
        <span
          aria-hidden
          className="bg-warning text-warning-on-solid border-card absolute -top-1.5 -right-1.5 grid size-3.5 place-items-center rounded-full border-2 text-2xs font-semibold"
        >
          !
        </span>
      ) : null}
      <span
        aria-hidden
        className={cn(
          "font-mono text-2xs",
          current ? "text-foreground font-medium" : "text-foreground-muted",
        )}
      >
        {number}
      </span>
    </button>
  );
}
