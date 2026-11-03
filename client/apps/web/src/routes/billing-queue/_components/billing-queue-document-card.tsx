import { LazyImage } from "@/components/image";
import { documentContentUrl } from "@/services/document";
import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { ArrowDownIcon, ArrowUpIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { Document } from "@trenova/shared/types/document";
import { useRef, useState } from "react";

/**
 * A document's first page as the server drew it, so the biller sees which paper it
 * is before opening it. Until the preview exists, or for a file it cannot draw, the
 * card shows a blank page.
 */
function DocumentThumb({ doc }: { doc: Document }) {
  const previewUrl = doc.previewStatus === "Ready" ? documentContentUrl(doc.id, "preview") : null;
  const [failed, setFailed] = useState<string | null>(null);

  if (previewUrl && failed !== previewUrl) {
    return (
      <LazyImage
        src={previewUrl}
        alt=""
        onError={() => setFailed(previewUrl)}
        wrapperClassName="block size-full"
        className="size-full object-cover object-top"
      />
    );
  }

  return (
    <span aria-hidden className="flex size-full flex-col gap-1.5 px-[12%] py-[14%]">
      <span className="bg-border h-1 w-1/2 rounded-full" />
      <span className="bg-border h-0.5 rounded-full" />
      <span className="bg-border h-0.5 w-4/5 rounded-full" />
      <span className="bg-border h-0.5 rounded-full" />
      <span className="bg-border h-0.5 w-3/5 rounded-full" />
    </span>
  );
}

const documentName = (doc: Document) => doc.documentType?.name ?? doc.originalName;

function CardName({ doc, attention }: { doc: Document; attention?: string }) {
  return (
    <span className="bq-card-name flex min-w-0 items-center gap-1.5 text-xs font-medium">
      {attention ? (
        <span aria-hidden className="bg-warning size-1.5 shrink-0 rounded-full" />
      ) : null}
      <span className="truncate">{documentName(doc)}</span>
    </span>
  );
}

const PAGE_CLASS =
  "bq-card-page ui-field ui-focus-ring bg-card relative block aspect-[17/22] w-full overflow-hidden";

/**
 * The card lifted under the pointer while it is dragged: the same page and name,
 * tilted a little and ringed, so it reads as the document in hand. Carrying several,
 * it shows the others as sheets fanned behind it and how many there are.
 */
export function DocumentCardLifted({
  doc,
  attention,
  count,
}: {
  doc: Document;
  attention?: string;
  count: number;
}) {
  return (
    <div
      className="bq-card-lifted flex min-w-0 flex-col gap-1.5"
      data-stack={count > 1 ? Math.min(count, 3) : undefined}
      aria-hidden
    >
      <span className="bq-card-stack relative block">
        <span className={PAGE_CLASS}>
          <DocumentThumb doc={doc} />
        </span>
        {count > 1 ? (
          <span className="bg-brand text-brand-foreground absolute -top-2 -right-2 z-10 grid h-5 min-w-5 place-items-center rounded-full px-1.5 text-2xs font-medium tabular-nums">
            {count}
          </span>
        ) : null}
      </span>
      <CardName doc={doc} attention={attention} />
    </div>
  );
}

/** How a click changes the selection: to this card alone, this card in or out, or a run of cards. */
export type SelectMode = "only" | "toggle" | "range";

export type DocumentCardProps = {
  doc: Document;
  /** Its place after the invoice, counting from one; absent outside the packet. */
  order?: number;
  /** In the board's selection, which is what the ring, the box and the dock all show. */
  selected: boolean;
  /** Something about it needs the biller: a signature it lacks, a rejection. */
  attention?: string;
  /** A save or delete for it is still running. */
  busy: boolean;
  /** It can be dragged to another place or lane. */
  sortable: boolean;
  /** Sends it with the invoice or leaves it out; absent once the packet is fixed. */
  onMove?: () => void;
  /** Moves it one place along the packet, from the arrow keys. */
  onStep?: (step: -1 | 1) => void;
  onSelect: (mode: SelectMode) => void;
  /** Opens it in the document viewer, from a double click. */
  onOpen: () => void;
  /** Told the card's page button, so the board can put focus back on it after a move. */
  onPageNode?: (id: string, node: HTMLButtonElement | null) => void;
  /** Several cards can be selected at once; the box is offered only then. */
  multiSelect: boolean;
  /** More than one card is selected, so every card shows its box. */
  multi: boolean;
};

/**
 * One document on the packet board: its first page and its type. Only what
 * differs is said: a dot when it needs the biller. A selected card is ringed like
 * a focused field. A click selects it alone; ⌘ or Ctrl adds it to the selection or
 * takes it out, as its box does, and shift selects the run up to it. Dragging it
 * leaves a dashed place where it will land while the cards around it make room.
 */
export function DocumentCard({
  doc,
  order,
  selected,
  attention,
  busy,
  sortable,
  onMove,
  onStep,
  onSelect,
  onOpen,
  onPageNode,
  multiSelect,
  multi,
}: DocumentCardProps) {
  const t = useT();
  const shift = useRef(false);
  const name = documentName(doc);
  const inPacket = order !== undefined;
  const { setNodeRef, setActivatorNodeRef, listeners, transform, transition, isDragging } =
    useSortable({ id: doc.id, disabled: !sortable || busy });
  const moveLabel = inPacket
    ? t("Leave {0} out of the packet", name)
    : t("Send {0} with the invoice", name);

  return (
    <div
      ref={setNodeRef}
      role="listitem"
      data-dragging={isDragging || undefined}
      style={{ transform: CSS.Translate.toString(transform), transition }}
      className="bq-card group relative flex min-w-0 flex-col gap-1.5"
    >
      <button
        ref={(node) => {
          setActivatorNodeRef(node);
          onPageNode?.(doc.id, node);
        }}
        type="button"
        aria-pressed={selected}
        aria-keyshortcuts={onStep && inPacket ? "ArrowLeft ArrowRight" : undefined}
        {...listeners}
        onClick={(event) =>
          onSelect(event.shiftKey ? "range" : event.metaKey || event.ctrlKey ? "toggle" : "only")
        }
        onDoubleClick={onOpen}
        onKeyDown={(event) => {
          if (!onStep || !inPacket) return;
          const step = event.key === "ArrowLeft" ? -1 : event.key === "ArrowRight" ? 1 : 0;
          if (step === 0) return;
          event.preventDefault();
          onStep(step);
        }}
        className={cn(PAGE_CLASS, sortable && !busy && "cursor-grab")}
      >
        <span className="sr-only">
          {name}
          {attention ? <>, {attention}</> : null}
          {inPacket ? <>, {t("number {0} in the packet", order)}</> : null}
        </span>
        <DocumentThumb doc={doc} />
        {busy ? (
          <span
            role="status"
            aria-label={t("Saving")}
            className="bg-card/70 absolute inset-0 grid place-items-center"
          >
            <Spinner className="size-4" />
          </span>
        ) : null}
      </button>
      {multiSelect && !isDragging ? (
        <span
          className={cn(
            "bq-card-control bg-card absolute top-1.5 left-1.5 z-10 grid size-5 place-items-center rounded-sm transition-opacity",
            multi
              ? "opacity-100"
              : "opacity-0 group-focus-within:opacity-100 group-hover:opacity-100",
          )}
        >
          <Checkbox
            checked={selected}
            aria-label={t("Select {0}", name)}
            onPointerDown={(event) => {
              shift.current = event.shiftKey;
            }}
            onCheckedChange={() => {
              onSelect(shift.current ? "range" : "toggle");
              shift.current = false;
            }}
          />
        </span>
      ) : null}
      {onMove && !busy && !isDragging ? (
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                size="icon-xs"
                variant="outline"
                aria-label={moveLabel}
                onClick={onMove}
                className="bq-card-control bg-card absolute top-1.5 right-1.5 opacity-0 transition-opacity group-focus-within:opacity-100 group-hover:opacity-100"
              >
                {inPacket ? (
                  <ArrowDownIcon className="size-3" />
                ) : (
                  <ArrowUpIcon className="size-3" />
                )}
              </Button>
            }
          />
          <TooltipContent side="top" sideOffset={8}>
            {inPacket ? t("Leave out") : t("Send with invoice")}
          </TooltipContent>
        </Tooltip>
      ) : null}
      <CardName doc={doc} attention={attention} />
    </div>
  );
}
