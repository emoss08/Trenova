import type { DispatchBoardMove } from "@/lib/graphql/dispatch-console";
import { VirtualRows, type VirtualRow, type VirtualRowsHandle } from "@/components/virtual-rows";
import {
  AlertTriangleIcon,
  Building07Icon,
  CheckCircleIcon,
  FlameIcon,
  Lock01Icon,
  Send01Icon,
  Snowflake01Icon,
} from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptyTable, type EmptyTableColumn } from "@trenova/shared/components/ui/empty-table";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixTime } from "@trenova/shared/lib/date";
import { cn, formatCompactCurrency } from "@trenova/shared/lib/utils";
import { useEffect, useRef, type ReactNode } from "react";
import { MoveQueueRowsSkeleton, QUEUE_GRID_CLASS } from "./console-skeletons";
import {
  URGENCY_ORDER,
  formatMiles,
  groupMovesByUrgency,
  pickupState,
  urgencyMeta,
  type PickupTone,
} from "./dispatch-vocabulary";
import { formatWhen } from "./format-when";
import { queueRowId } from "./queue-ids";
import { tenderChipMeta } from "./tender-vocabulary";
import type { DispatchOutcome } from "./use-dispatch-actions";

// Two lines of text and the row's padding; the group heading is measured once mounted.
const QUEUE_ROW_ESTIMATE_PX = 52;

export const QUEUE_COLUMNS: readonly EmptyTableColumn[] = [
  { label: "Pickup" },
  { label: "Move" },
  { label: "Next step" },
];

const PICKUP_TONE_CLASS: Record<PickupTone, string> = {
  danger: "text-danger-foreground",
  warning: "text-warning-foreground",
  neutral: "text-foreground",
};

function lane(move: DispatchBoardMove): string {
  const origin = [move.originCity || move.originName, move.originState].filter(Boolean).join(", ");
  const destination = [move.destinationCity || move.destinationName, move.destinationState]
    .filter(Boolean)
    .join(", ");
  return `${origin || "—"} → ${destination || "—"}`;
}

/** The customer and the money on one line, each part only when the move has it. */
function moveFacts(move: DispatchBoardMove): string {
  return [
    move.customerName,
    move.distance != null ? formatMiles(move.distance) : null,
    move.revenue != null ? formatCompactCurrency(move.revenue) : null,
  ]
    .filter(Boolean)
    .join(" · ");
}

/**
 * The one thing that happens next for a move, in the order a dispatcher would act on it:
 * a hold stops everything, a tender in flight is someone else's answer to wait on, a
 * covered move names who holds it, Nova's proposal waits on a decision, an approval that
 * could not be carried out says why, and anything else waits on a ranking.
 */
function NextStep({ move }: { move: DispatchBoardMove }) {
  const t = useT();

  if (move.hasActiveHold) {
    return (
      <>
        <div className="flex min-w-0 items-center gap-1.5">
          <span className="text-muted-foreground truncate">
            {t("Can't dispatch until the hold is released")}
          </span>
        </div>
        <div className="text-muted-foreground truncate text-2xs">{t("Shipment on hold")}</div>
      </>
    );
  }

  if (move.liveTender) {
    const meta = tenderChipMeta(move.liveTender);
    return (
      <>
        <div className="flex min-w-0 items-center gap-1.5">
          <Badge
            variant={
              meta.tone === "attention" ? "warning" : meta.tone === "active" ? "success" : "info"
            }
            className="h-4 shrink-0 rounded-md px-1 text-3xs"
          >
            <Send01Icon className="mr-0.5 size-2.5" aria-hidden />
            {t("Tender")}
          </Badge>
          <span className="truncate">{meta.label}</span>
        </div>
        <div className="text-muted-foreground truncate text-2xs">
          {t(
            "{0, plural, one {# carrier offered} other {# carriers offered}}",
            move.liveTender.offerCount,
          )}
        </div>
      </>
    );
  }

  if (move.isCovered) {
    return move.coverageType === "carrier" ? (
      <>
        <div className="flex min-w-0 items-center gap-1.5">
          <Building07Icon className="text-muted-foreground size-3 shrink-0" aria-hidden />
          <span className="truncate font-medium">{move.assignedCarrierName}</span>
        </div>
        <div className="text-muted-foreground truncate text-2xs">
          {move.carrierTotalCost != null
            ? t("Covered by carrier · {0}", formatCompactCurrency(move.carrierTotalCost))
            : t("Covered by carrier")}
        </div>
      </>
    ) : (
      <>
        <div className="flex min-w-0 items-center gap-1.5">
          <CheckCircleIcon className="text-success-foreground size-3 shrink-0" aria-hidden />
          <span className="truncate font-medium">
            {move.assignedTractorCode
              ? t("{0} · {1}", move.assignedWorkerName, move.assignedTractorCode)
              : move.assignedWorkerName}
          </span>
        </div>
        <div className="text-muted-foreground truncate text-2xs">{t("Covered")}</div>
      </>
    );
  }

  const proposal = move.pendingProposal;
  if (proposal) {
    return (
      <>
        <div className="flex min-w-0 items-center gap-1.5">
          <NovaBadge size="sm" label={t("Nova proposes")} />
          <span className={cn("truncate font-medium", proposal.stale && "text-muted-foreground")}>
            {proposal.workerName || t("A driver")}
          </span>
          <span className="text-muted-foreground ml-auto shrink-0 font-mono text-xs tabular-nums">
            {t("{0}%", Math.round(proposal.confidence * 100))}
          </span>
        </div>
        <div className="text-muted-foreground truncate text-2xs">
          {proposal.stale
            ? t("The move changed since this was proposed; the next plan replaces it")
            : proposal.rationale}
        </div>
      </>
    );
  }

  const failed = move.failedProposal;
  if (failed) {
    return (
      <>
        <div className="flex min-w-0 items-center gap-1.5">
          <AlertTriangleIcon className="text-warning-foreground size-3 shrink-0" aria-hidden />
          <span className="truncate font-medium">
            {failed.workerName
              ? t("Approved {0}; not assigned", failed.workerName)
              : t("Approved; not assigned")}
          </span>
        </div>
        <div className="text-muted-foreground truncate text-2xs" title={failed.reason}>
          {failed.reason}
        </div>
      </>
    );
  }

  return (
    <>
      <div className="flex min-w-0 items-center gap-1.5">
        <span className="text-muted-foreground truncate">{t("No proposal yet")}</span>
      </div>
      <div className="text-muted-foreground truncate text-2xs">{t("Open it to rank drivers")}</div>
    </>
  );
}

function MoveFlags({ move }: { move: DispatchBoardMove }) {
  const t = useT();

  return (
    <>
      {move.temperatureMin != null && (
        <Badge variant="info" className="h-4 shrink-0 rounded-md px-1 text-3xs">
          <Snowflake01Icon className="mr-0.5 size-2.5" aria-hidden />
          {t("{0}–{1}°F", move.temperatureMin, move.temperatureMax)}
        </Badge>
      )}
      {move.hasHazmat && (
        <Badge variant="danger" className="h-4 shrink-0 rounded-md px-1 text-3xs">
          <FlameIcon className="mr-0.5 size-2.5" aria-hidden />
          {t("Hazmat")}
        </Badge>
      )}
      {move.hasActiveHold && (
        <Badge variant="warning" className="h-4 shrink-0 rounded-md px-1 text-3xs">
          <Lock01Icon className="mr-0.5 size-2.5" aria-hidden />
          {t("On hold")}
        </Badge>
      )}
      {move.moveCount > 1 && (
        <Badge
          variant="neutral"
          appearance="outline"
          className="h-4 shrink-0 rounded-md px-1 text-3xs"
        >
          {t("Leg {0}/{1}", move.sequence + 1, move.moveCount)}
        </Badge>
      )}
    </>
  );
}

function QueueRow({
  move,
  isSelected,
  isDeparting,
  nowSeconds,
  onSelect,
}: {
  move: DispatchBoardMove;
  isSelected: boolean;
  isDeparting: boolean;
  nowSeconds: number;
  onSelect: (moveId: string) => void;
}) {
  const pickup = pickupState(
    move,
    nowSeconds,
    (seconds) => formatWhen(seconds, nowSeconds),
    formatUnixTime,
  );

  return (
    // A covered move collapses out rather than vanishing, so the eye sees where it went.
    <div
      className={cn(
        "grid pt-px transition-[grid-template-rows,opacity] duration-200 ease-settle motion-reduce:transition-none",
        isDeparting ? "grid-rows-[0fr] opacity-0" : "grid-rows-[1fr]",
      )}
    >
      <div className="overflow-hidden">
        <div
          id={queueRowId(move.moveId)}
          role="option"
          aria-selected={isSelected}
          tabIndex={isSelected ? 0 : -1}
          data-glide-target
          onClick={() => onSelect(move.moveId)}
          onKeyDown={(event) => {
            if (event.key === "Enter" || event.key === " ") {
              event.preventDefault();
              onSelect(move.moveId);
            }
          }}
          className={cn(
            QUEUE_GRID_CLASS,
            "ui-focus-ring relative mx-1.5 cursor-pointer items-center rounded-md px-2 py-1.75 text-xs",
            isSelected && "bg-surface-hover",
          )}
        >
          <div className="min-w-0">
            <span className={cn("block truncate font-medium", PICKUP_TONE_CLASS[pickup.tone])}>
              {pickup.headline}
            </span>
            <span className="text-muted-foreground block truncate text-2xs">{pickup.window}</span>
          </div>
          <div className="min-w-0">
            <div className="flex min-w-0 items-center gap-2">
              <span className="shrink-0 font-mono font-medium">{move.proNumber}</span>
              <span className="truncate">{lane(move)}</span>
              <MoveFlags move={move} />
            </div>
            <div className="text-muted-foreground truncate text-2xs">{moveFacts(move)}</div>
          </div>
          <div className="min-w-0">
            <NextStep move={move} />
          </div>
        </div>
      </div>
    </div>
  );
}

export function GroupHeading({
  label,
  description,
  dotClass,
  count,
}: {
  label: string;
  description: string;
  dotClass: string;
  count: number;
}) {
  return (
    // The heading is the page's background, not a row's, so it reads as a divider between
    // groups; once rows scroll under it, its rule eases in with the scroll area's top fade.
    <div
      role="presentation"
      className="bg-background flex h-7 items-center gap-2 border-y border-transparent px-3.5 text-2xs font-medium transition-[border-color] duration-300 ease-settle group-data-stuck/vrow:border-b-border"
    >
      <span className={cn("size-1.5 rounded-full", dotClass)} aria-hidden />
      <span>{label}</span>
      <span className="text-muted-foreground font-normal">{description}</span>
      <span className="text-muted-foreground ml-auto font-normal tabular-nums">{count}</span>
    </div>
  );
}

/** Each outcome's tone: a tinted strip, its rule and its text, so a refusal reads apart from a
 * success before its words do. */
const OUTCOME_TONES = {
  ok: {
    surface: "bg-success-subtle border-success-border text-success-subtle-foreground",
    Icon: CheckCircleIcon,
  },
  warn: {
    surface: "bg-warning-subtle border-warning-border text-warning-subtle-foreground",
    Icon: AlertTriangleIcon,
  },
} satisfies Record<DispatchOutcome["tone"], { surface: string; Icon: typeof CheckCircleIcon }>;

/** What the last write here came to, in place, with the way to take it back. */
export function QueueOutcome({
  outcome,
  canUndo,
  onUndo,
}: {
  outcome: DispatchOutcome | null;
  canUndo: boolean;
  onUndo: () => void;
}) {
  return (
    <div role="status" aria-live="polite" aria-atomic="true">
      {outcome ? (
        <QueueOutcomeRow key={outcome.id} outcome={outcome} canUndo={canUndo} onUndo={onUndo} />
      ) : null}
    </div>
  );
}

function QueueOutcomeRow({
  outcome,
  canUndo,
  onUndo,
}: {
  outcome: DispatchOutcome;
  canUndo: boolean;
  onUndo: () => void;
}) {
  const t = useT();
  const { surface, Icon } = OUTCOME_TONES[outcome.tone];

  return (
    <div
      data-tone={outcome.tone}
      className={cn("flex animate-rise items-center gap-2 border-b px-3.5 py-2 text-xs", surface)}
    >
      <Icon className="size-3.5 shrink-0" aria-hidden />
      <span className="min-w-0 flex-1">{outcome.text}</span>
      {outcome.undoable && canUndo ? (
        <Button
          variant="link"
          size="xxs"
          className="shrink-0 font-medium text-current"
          onClick={onUndo}
          shortcut="U"
        >
          {t("Undo")}
        </Button>
      ) : null}
    </div>
  );
}

export type QueueRead = {
  isPending: boolean;
  isError: boolean;
  error: unknown;
  refetch: () => void;
};

/**
 * The moves that need a decision, grouped by the server's urgency and ordered within each
 * group as the board sends them. One hover fill glides between rows and the open row
 * wears the same fill, so where the pointer is and what is open read as one thing.
 */
export function MoveQueue({
  moves,
  read,
  errorState,
  emptyState,
  selectedMoveId,
  departingMoveIds,
  onSelectMove,
}: {
  moves: readonly DispatchBoardMove[];
  read: QueueRead;
  /** Drawn in place of the rows when the board could not be read. */
  errorState: ReactNode;
  /** Drawn in place of the rows after a read that found nothing. */
  emptyState: ReactNode;
  selectedMoveId: string | null;
  departingMoveIds: ReadonlySet<string>;
  onSelectMove: (moveId: string) => void;
}) {
  const t = useT();
  const nowSeconds = useNow("minute");
  const listRef = useRef<VirtualRowsHandle>(null);

  // Only the rows near the viewport are mounted, so a selection made elsewhere (J and K,
  // a link, the timeline) is scrolled to rather than left on a row that may not exist.
  useEffect(() => {
    if (selectedMoveId) listRef.current?.scrollToKey(selectedMoveId, { align: "auto" });
  }, [selectedMoveId]);

  if (read.isPending) {
    return <MoveQueueRowsSkeleton />;
  }
  if (read.isError) {
    return <>{errorState}</>;
  }
  if (moves.length === 0) {
    return <>{emptyState}</>;
  }

  const grouped = groupMovesByUrgency(moves);
  const rows: VirtualRow[] = [];
  for (const bucket of URGENCY_ORDER) {
    const bucketMoves = grouped.get(bucket) ?? [];
    if (bucketMoves.length === 0) continue;
    const meta = urgencyMeta(bucket);
    rows.push({
      key: `group-${bucket}`,
      sticky: true,
      render: () => (
        <GroupHeading
          label={t(meta.label)}
          description={t(meta.description)}
          dotClass={meta.dotClass}
          count={bucketMoves.filter((move) => !departingMoveIds.has(move.moveId)).length}
        />
      ),
    });
    for (const move of bucketMoves) {
      rows.push({
        key: move.moveId,
        render: () => (
          <QueueRow
            move={move}
            isSelected={move.moveId === selectedMoveId}
            isDeparting={departingMoveIds.has(move.moveId)}
            nowSeconds={nowSeconds}
            onSelect={onSelectMove}
          />
        ),
      });
    }
  }

  return (
    <VirtualRows
      rows={rows}
      handleRef={listRef}
      estimateSize={QUEUE_ROW_ESTIMATE_PX}
      initialHeight={640}
      role="listbox"
      aria-label={t("Moves that need a decision")}
      hoverGlide
      mask="card"
      maskClassName="top-7"
      maskHeight={18}
      className="min-h-0 flex-1"
      footer={<div className="h-2" aria-hidden />}
    />
  );
}

/** The queue's empty form, said only after a read that worked. */
export function QueueEmpty({
  narrowed,
  includeCovered,
  onClearSearch,
}: {
  narrowed: boolean;
  includeCovered: boolean;
  onClearSearch: () => void;
}) {
  const t = useT();

  return (
    <EmptyTable
      columns={QUEUE_COLUMNS}
      title={narrowed ? t("No moves match the search") : t("Every move in this window is covered")}
      description={
        narrowed
          ? t("Nothing in this window matches. Clear the search to see every move.")
          : includeCovered
            ? t("There are no moves in this window. Try the next window or a longer one.")
            : t("Nothing here needs a driver. Show covered moves to see who holds each one.")
      }
      onClearFilters={narrowed ? onClearSearch : undefined}
    />
  );
}
