import {
  closestCenter,
  defaultDropAnimationSideEffects,
  DndContext,
  DragOverlay,
  MouseSensor,
  pointerWithin,
  TouchSensor,
  useDroppable,
  useSensor,
  useSensors,
  type CollisionDetection,
  type DragEndEvent,
  type DragOverEvent,
  type DragStartEvent,
  type DropAnimation,
  type UniqueIdentifier,
} from "@dnd-kit/core";
import { rectSortingStrategy, SortableContext, type SortingStrategy } from "@dnd-kit/sortable";
import { Lock01Icon, PlusIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { Document } from "@trenova/shared/types/document";
import type { ShipmentBillingRequirement } from "@trenova/shared/types/shipment";
import { useEffect, useId, useRef, useState, type KeyboardEvent, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { DocumentCard, DocumentCardLifted, type SelectMode } from "./billing-queue-document-card";
import {
  isLaneDropId,
  laneDropId,
  laneOf,
  lanesAfterCross,
  lanesAfterDrop,
  lanesWithGroup,
  lanesWithoutGroup,
  sameOrder,
  type Lane,
  type Lanes,
} from "./billing-queue-board-lanes";
import { MissingDocumentSlot } from "./billing-queue-missing-document";
import "./billing-queue-document-board.css";

/** The packet as the board edits it: the documents in order, and how to change them. */
export type BoardPacket = {
  ids: readonly string[];
  editable: boolean;
  onPlace: (documentId: string, index: number) => void;
  onRemove: (documentId: string) => void;
  /** Saves the packet a drag left behind, naming the documents that were moved; settles with the save. */
  onArrange: (next: string[], moved: readonly string[]) => Promise<void>;
};

type BoardProps = {
  documents: readonly Document[];
  missing: readonly ShipmentBillingRequirement[];
  /** Absent when the item sends no packet: the board is then one lane. */
  packet?: BoardPacket;
  /** The selected documents: one shows its line under the board, several the dock. */
  selection: ReadonlySet<string>;
  onSelectionChange: (next: ReadonlySet<string>) => void;
  /** Several documents can be selected at once, for a bulk action. */
  multiSelect: boolean;
  onOpen: (doc: Document) => void;
  busy: ReadonlySet<string>;
  attentionOf: (doc: Document) => string | undefined;
  progressFor: (requirement: ShipmentBillingRequirement) => number | undefined;
  /** Present while the item can take documents. */
  onUploadFor?: (requirement: ShipmentBillingRequirement) => void;
  onDropFor?: (requirement: ShipmentBillingRequirement, files: File[]) => void;
  onAddFiles?: () => void;
  onDropFiles?: (files: File[]) => void;
};

// The card in hand flies to its place in the cards' own easing, settling its tilt
// on the way, while its place waits hidden under it.
const dropAnimation: DropAnimation = {
  duration: 220,
  easing: "cubic-bezier(0.2, 0.8, 0.2, 1)",
  sideEffects: defaultDropAnimationSideEffects({
    className: { dragOverlay: "bq-card-dropping" },
    styles: { active: { opacity: "0" } },
  }),
};

// What stays out of the packet keeps the shipment's own order, so its cards hold still
// while one is dragged among them instead of making room for an order nothing keeps.
const keepInPlace: SortingStrategy = () => null;

// The card under the pointer wins; between cards, the lane under it; outside both,
// whatever is nearest. A lane alone would swallow every card inside it.
const collide: CollisionDetection = (args) => {
  const hits = pointerWithin(args);
  const cards = hits.filter((hit) => !isLaneDropId(String(hit.id)));
  if (cards.length > 0) return cards;
  return hits.length > 0 ? hits : closestCenter(args);
};

/**
 * The shipment's paperwork as the invoice will use it, in two lanes: the packet,
 * in the order it follows the invoice, with a slot for each required document
 * still missing; and what stays behind. A document is dragged within or across the
 * lanes, crosses by its arrow or from its menu, and moves along the packet with the
 * arrow keys. While it is dragged the cards make room for it at once, a dashed place
 * shows where it will land, and the lane it is crossing into is outlined; the packet
 * is saved once, when it is dropped. An item that sends no packet shows one lane.
 */
export function BillingQueueDocumentBoard({
  documents,
  missing,
  packet,
  selection,
  onSelectionChange,
  multiSelect,
  onOpen,
  busy,
  attentionOf,
  progressFor,
  onUploadFor,
  onDropFor,
  onAddFiles,
  onDropFiles,
}: BoardProps) {
  const t = useT();
  const [drag, setDrag] = useState<{
    id: string;
    from: Lane;
    lanes: Lanes;
    group: readonly string[];
  } | null>(null);
  const [landed, setLanded] = useState<string[] | null>(null);
  const pages = useRef(new Map<string, HTMLButtonElement>());
  const refocus = useRef<string | null>(null);
  const anchor = useRef<string | null>(null);
  const editable = Boolean(packet?.editable);
  const sensors = useSensors(
    useSensor(MouseSensor, { activationConstraint: { distance: 6 } }),
    useSensor(TouchSensor, { activationConstraint: { delay: 220, tolerance: 6 } }),
  );

  const byId = new Map(documents.map((doc) => [doc.id, doc]));
  const packetIds = (landed ?? packet?.ids ?? []).filter((id) => byId.has(id));
  const packetSet = new Set(packetIds);
  const leftIds = documents.map((doc) => doc.id).filter((id) => !packetSet.has(id));
  const allIds = documents.map((doc) => doc.id);
  const shipmentOrder = new Map(allIds.map((id, index) => [id, index]));
  const inShipmentOrder = (ids: readonly string[]) =>
    [...ids].sort((a, b) => (shipmentOrder.get(a) ?? 0) - (shipmentOrder.get(b) ?? 0));
  const shown: Lanes = drag?.lanes ?? { packet: packetIds, out: leftIds };
  const dragged = drag ? byId.get(drag.id) : undefined;
  const crossingInto = drag ? laneOf(drag.lanes, drag.id) : null;
  const target = drag && crossingInto !== drag.from ? crossingInto : null;

  useEffect(() => {
    if (!refocus.current) return;
    pages.current.get(refocus.current)?.focus();
    refocus.current = null;
  });

  const registerPage = (id: string, node: HTMLButtonElement | null) => {
    if (node) pages.current.set(id, node);
    else pages.current.delete(id);
  };

  const clearOnEscape = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.key === "Escape" && selection.size > 0) onSelectionChange(new Set());
  };

  // A plain click selects the card alone and becomes the anchor; a toggle adds or
  // removes it and moves the anchor; a range adds every card from the anchor to it
  // within their lane, and with no anchor in that lane adds just this card.
  const select = (lane: readonly string[], id: string, mode: SelectMode) => {
    if (mode === "only" || !multiSelect) {
      anchor.current = id;
      onSelectionChange(new Set([id]));
      return;
    }
    const next = new Set(selection);
    const from = anchor.current ? lane.indexOf(anchor.current) : -1;
    const to = lane.indexOf(id);
    if (mode === "range" && from >= 0 && to >= 0) {
      for (const entry of lane.slice(Math.min(from, to), Math.max(from, to) + 1)) next.add(entry);
    } else if (next.has(id)) {
      next.delete(id);
      anchor.current = id;
    } else {
      next.add(id);
      anchor.current = id;
    }
    onSelectionChange(next);
  };

  // Dragging a selected card carries the whole selection, packet first, each lane
  // in its order; dragging any other card carries that card alone.
  const groupFor = (id: string): string[] =>
    selection.size > 1 && selection.has(id)
      ? [...packetIds, ...leftIds].filter((entry) => selection.has(entry))
      : [id];

  const onDragStart = ({ active }: DragStartEvent) => {
    const id = String(active.id);
    const all = { packet: [...packetIds], out: [...leftIds] };
    const from = laneOf(all, id);
    if (!from) return;
    const group = groupFor(id);
    setDrag({ id, from, group, lanes: lanesWithoutGroup(all, id, group) });
  };

  // Crossing into the other lane moves the card there at once, beside the card it
  // is over (after it when the pointer is past that card's middle), so the lane
  // makes room as it would after the drop.
  const onDragOver = ({ active, over }: DragOverEvent) => {
    if (!drag || !over) return;
    const moving = active.rect.current.translated;
    const past = moving ? moving.left > over.rect.left + over.rect.width / 2 : false;
    const lanes = lanesAfterCross(drag.lanes, String(active.id), String(over.id), past);
    if (lanes === drag.lanes) return;
    setDrag({ ...drag, lanes: { ...lanes, out: inShipmentOrder(lanes.out) } });
  };

  // The dropped order is drawn in the same frame the drag ends and held until its
  // save settles, so the cards never fall back to the old order on the way.
  const onDragEnd = ({ active, over }: DragEndEvent) => {
    const current = drag;
    setDrag(null);
    if (!current || !packet || !over) return;
    const id = String(active.id);
    const dropped = lanesAfterDrop(current.lanes, id, String(over.id));
    const lanes = lanesWithGroup(dropped, id, current.group);
    if (sameOrder(lanes.packet, packetIds)) return;
    setLanded(lanes.packet);
    void packet
      .onArrange(lanes.packet, current.group)
      .catch(() => undefined)
      .finally(() => setLanded(null));
  };

  const card = (id: string, lane: readonly string[], order?: number) => {
    const doc = byId.get(id);
    if (!doc) return null;
    const movable = editable && packet;
    return (
      <DocumentCard
        key={doc.id}
        doc={doc}
        order={order}
        selected={selection.has(doc.id)}
        attention={attentionOf(doc)}
        busy={busy.has(doc.id)}
        sortable={Boolean(movable)}
        onPageNode={registerPage}
        multiSelect={multiSelect}
        multi={selection.size > 1}
        onSelect={(mode) => select(lane, doc.id, mode)}
        onOpen={() => onOpen(doc)}
        onMove={
          movable
            ? () => {
                refocus.current = doc.id;
                if (order === undefined) movable.onPlace(doc.id, packetIds.length);
                else movable.onRemove(doc.id);
              }
            : undefined
        }
        onStep={
          movable && order !== undefined
            ? (step) => {
                const to = order - 1 + step;
                if (to < 0 || to >= packetIds.length) return;
                refocus.current = doc.id;
                movable.onPlace(doc.id, to);
              }
            : undefined
        }
      />
    );
  };

  const slots = missing.map((requirement) => (
    <MissingDocumentSlot
      key={`missing-${requirement.documentTypeId}`}
      requirement={requirement}
      uploadProgress={progressFor(requirement)}
      onUpload={onUploadFor ? () => onUploadFor(requirement) : undefined}
      onDropFiles={onDropFor ? (files) => onDropFor(requirement, files) : undefined}
    />
  ));
  const add =
    onAddFiles && onDropFiles ? <AddFilesTile onAdd={onAddFiles} onDrop={onDropFiles} /> : null;

  if (!packet) {
    return (
      <div onKeyDown={clearOnEscape}>
        <BoardLane label={t("Shipment documents")} hideLabel>
          {allIds.map((id) => card(id, allIds))}
          {slots}
          {add}
        </BoardLane>
      </div>
    );
  }

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={collide}
      onDragStart={onDragStart}
      onDragOver={onDragOver}
      onDragEnd={onDragEnd}
      onDragCancel={() => setDrag(null)}
      accessibility={{
        announcements: {
          onDragStart: ({ active }) => {
            const count = groupFor(String(active.id)).length;
            return count > 1
              ? t("Picked up {0, plural, one {# document} other {# documents}}", count)
              : t("Picked up {0}", nameOf(byId, active.id));
          },
          onDragOver: () => undefined,
          onDragEnd: ({ active, over }) =>
            over
              ? t("Dropped {0}", nameOf(byId, active.id))
              : t("{0} was put back", nameOf(byId, active.id)),
          onDragCancel: ({ active }) => t("{0} was put back", nameOf(byId, active.id)),
        },
        screenReaderInstructions: {
          draggable: t("Use the left and right arrow keys to move a document along the packet."),
        },
      }}
    >
      <div className="flex flex-col gap-4.5" onKeyDown={clearOnEscape}>
        <BoardLane
          label={t("Packet")}
          hint={
            editable ? (
              t("Sent after the invoice, in order")
            ) : (
              <span className="inline-flex items-center gap-1">
                <Lock01Icon aria-hidden className="size-3" />
                {t("Sent after the invoice")}
              </span>
            )
          }
          dropId={editable ? laneDropId("packet") : undefined}
          dropTarget={target === "packet"}
        >
          <SortableContext items={shown.packet} strategy={rectSortingStrategy}>
            {shown.packet.map((id, index) => card(id, shown.packet, index + 1))}
          </SortableContext>
          {slots}
        </BoardLane>
        {shown.out.length > 0 || add || editable ? (
          <BoardLane
            label={t("Not sent")}
            dropId={editable ? laneDropId("out") : undefined}
            dropTarget={target === "out"}
          >
            <SortableContext items={shown.out} strategy={keepInPlace}>
              {shown.out.map((id) => card(id, shown.out))}
            </SortableContext>
            {add}
            {shown.out.length === 0 && !add ? (
              <p
                role="listitem"
                className="text-foreground-muted col-span-full py-6 text-center text-xs"
              >
                {t("Drag a document here to keep it out of the packet.")}
              </p>
            ) : null}
          </BoardLane>
        ) : null}
      </div>
      {/* On the body: an ancestor with a transform (the pane's reveal) would otherwise
          become the overlay's containing block, offsetting it and clipping it. */}
      {createPortal(
        <DragOverlay dropAnimation={dropAnimation}>
          {dragged && drag ? (
            <DocumentCardLifted
              doc={dragged}
              attention={attentionOf(dragged)}
              count={drag.group.length}
            />
          ) : null}
        </DragOverlay>,
        document.body,
      )}
    </DndContext>
  );
}

function nameOf(byId: ReadonlyMap<string, Document>, id: UniqueIdentifier): string {
  const doc = byId.get(String(id));
  return doc ? (doc.documentType?.name ?? doc.originalName) : String(id);
}

function BoardLane({
  label,
  hint,
  hideLabel = false,
  dropId,
  dropTarget = false,
  children,
}: {
  label: string;
  hint?: ReactNode;
  hideLabel?: boolean;
  /** The lane takes a dragged card anywhere in it, not only over another card. */
  dropId?: string;
  dropTarget?: boolean;
  children: ReactNode;
}) {
  const id = useId();

  return (
    <section
      aria-labelledby={id}
      className="bq-board-lane flex flex-col gap-2.5"
      data-drop-target={dropTarget || undefined}
    >
      <h4
        id={id}
        className={cn("flex items-baseline gap-2 text-sm font-medium", hideLabel && "sr-only")}
      >
        {label}
        {hint ? <span className="text-foreground-muted text-xs font-normal">{hint}</span> : null}
      </h4>
      {dropId ? (
        <DroppableCards dropId={dropId}>{children}</DroppableCards>
      ) : (
        <div role="list" className={CARDS_CLASS}>
          {children}
        </div>
      )}
    </section>
  );
}

const CARDS_CLASS = "bq-board-cards grid grid-cols-[repeat(auto-fill,minmax(6.5rem,1fr))] gap-2.5";

function DroppableCards({ dropId, children }: { dropId: string; children: ReactNode }) {
  const { setNodeRef } = useDroppable({ id: dropId });

  return (
    <div ref={setNodeRef} role="list" className={CARDS_CLASS}>
      {children}
    </div>
  );
}

/** The last place in a lane: pick files to add, or drop them on it. */
function AddFilesTile({ onAdd, onDrop }: { onAdd: () => void; onDrop: (files: File[]) => void }) {
  const t = useT();
  const [over, setOver] = useState(false);

  return (
    <div role="listitem" className="flex min-w-0 flex-col">
      <button
        type="button"
        onClick={onAdd}
        onDragEnter={(event) => {
          if (event.dataTransfer.types.includes("Files")) setOver(true);
        }}
        onDragOver={(event) => {
          if (!event.dataTransfer.types.includes("Files")) return;
          event.preventDefault();
          event.dataTransfer.dropEffect = "copy";
        }}
        onDragLeave={(event) => {
          if (!event.currentTarget.contains(event.relatedTarget as Node | null)) setOver(false);
        }}
        onDrop={(event) => {
          if (!event.dataTransfer.types.includes("Files")) return;
          event.preventDefault();
          setOver(false);
          onDrop([...event.dataTransfer.files]);
        }}
        className={cn(
          "ui-focus-ring border-border-strong text-foreground-muted hover:text-foreground hover:border-foreground-faint flex aspect-[17/22] w-full flex-col items-center justify-center gap-1.5 rounded-md border border-dashed text-xs transition-colors",
          over && "field-focus-state text-foreground",
        )}
      >
        <PlusIcon className="size-4" />
        {t("Add files")}
      </button>
    </div>
  );
}
