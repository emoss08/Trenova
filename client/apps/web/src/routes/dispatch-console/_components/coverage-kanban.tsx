import type { DispatchBoardMove } from "@/lib/graphql/dispatch-console";
import { useDroppable } from "@dnd-kit/core";
import { Badge } from "@trenova/shared/components/ui/badge";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { cn, formatCompactCurrency } from "@trenova/shared/lib/utils";
import { Building2Icon, FlameIcon, SnowflakeIcon, TriangleAlertIcon } from "lucide-react";
import { useMemo } from "react";
import { CoverageKanbanSkeleton } from "./console-skeletons";
import {
  URGENCY_ORDER,
  formatMiles,
  formatMinutesToPickup,
  groupMovesByUrgency,
  urgencyMeta,
  type UrgencyBucket,
} from "./dispatch-vocabulary";

function MoveCard({
  move,
  isSelected,
  onSelect,
}: {
  move: DispatchBoardMove;
  isSelected: boolean;
  onSelect: (moveId: string) => void;
}) {
  const { setNodeRef, isOver, active } = useDroppable({
    id: `move:${move.moveId}`,
    data: { type: "move-target", move },
  });
  const isDriverDrag = active?.data.current?.type === "driver";

  return (
    <div
      ref={setNodeRef}
      role="button"
      tabIndex={0}
      onClick={() => onSelect(move.moveId)}
      onKeyDown={(event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          onSelect(move.moveId);
        }
      }}
      className={cn(
        "flex cursor-pointer flex-col gap-1.5 rounded-md border bg-card p-2 transition-[border-color,box-shadow]",
        isSelected ? "border-brand shadow-[0_0_0_1px_var(--brand)]" : "hover:border-brand/40",
        isOver && isDriverDrag && "border-brand bg-brand/5 shadow-[0_0_0_2px_var(--brand)]",
      )}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="flex min-w-0 items-center gap-1.5">
          <span className="truncate font-mono text-xs font-semibold">{move.proNumber}</span>
          {move.moveCount > 1 && (
            <Badge variant="outline" className="h-4 shrink-0 rounded px-1 text-[9px]">
              Leg {move.sequence + 1}/{move.moveCount}
            </Badge>
          )}
        </div>
        <span
          className={cn(
            "shrink-0 text-[10px] tabular-nums",
            move.minutesToPickup < 0
              ? "font-medium text-red-600 dark:text-red-400"
              : "text-muted-foreground",
          )}
        >
          {move.originWindowStart > 0 ? formatMinutesToPickup(move.minutesToPickup) : "unscheduled"}
        </span>
      </div>

      <div className="flex items-center gap-1 text-[11px]">
        <span className="truncate">
          {move.originCity || move.originName || "—"}
          {move.originState ? `, ${move.originState}` : ""}
        </span>
        <span className="text-muted-foreground">→</span>
        <span className="truncate">
          {move.destinationCity || move.destinationName || "—"}
          {move.destinationState ? `, ${move.destinationState}` : ""}
        </span>
      </div>

      <div className="flex flex-wrap items-center gap-1">
        <span className="text-[10px] text-muted-foreground">
          {move.originWindowStart > 0
            ? formatUnixDateTime(move.originWindowStart)
            : "No appointment"}
        </span>
        {move.distance != null && (
          <Badge variant="outline" className="h-4 rounded px-1 text-[9px]">
            {formatMiles(move.distance)}
          </Badge>
        )}
        {move.revenue != null && (
          <Badge variant="outline" className="h-4 rounded px-1 text-[9px]">
            {formatCompactCurrency(move.revenue)}
          </Badge>
        )}
        {move.hasHazmat && (
          <Badge variant="inactive" className="h-4 rounded px-1 text-[9px]">
            <FlameIcon className="mr-0.5 size-2.5" aria-hidden />
            Hazmat
          </Badge>
        )}
        {move.temperatureMin != null && (
          <Badge variant="info" className="h-4 rounded px-1 text-[9px]">
            <SnowflakeIcon className="mr-0.5 size-2.5" aria-hidden />
            {move.temperatureMin}–{move.temperatureMax}°F
          </Badge>
        )}
        {move.hasActiveHold && (
          <Badge variant="inactive" className="h-4 rounded px-1 text-[9px]">
            <TriangleAlertIcon className="mr-0.5 size-2.5" aria-hidden />
            On hold
          </Badge>
        )}
      </div>

      <div className="flex items-center justify-between gap-2">
        <span className="truncate text-[10px] text-muted-foreground">{move.customerName}</span>
        {move.isCovered &&
          (move.coverageType === "carrier" ? (
            <Badge variant="active" className="h-4 shrink-0 rounded px-1 text-[9px]">
              <Building2Icon className="mr-0.5 size-2.5" aria-hidden />
              {move.assignedCarrierName}
              {move.carrierTotalCost != null
                ? ` · ${formatCompactCurrency(move.carrierTotalCost)}`
                : ""}
            </Badge>
          ) : (
            <Badge variant="active" className="h-4 shrink-0 rounded px-1 text-[9px]">
              {move.assignedWorkerName}
              {move.assignedTractorCode ? ` · ${move.assignedTractorCode}` : ""}
            </Badge>
          ))}
      </div>
    </div>
  );
}

function UrgencyColumn({
  bucket,
  moves,
  selectedMoveId,
  onSelectMove,
}: {
  bucket: UrgencyBucket;
  moves: DispatchBoardMove[];
  selectedMoveId: string | null;
  onSelectMove: (moveId: string) => void;
}) {
  const meta = urgencyMeta(bucket);

  return (
    <div className="flex min-h-0 w-60 shrink-0 flex-col rounded-md border bg-muted/30 xl:w-auto xl:flex-1">
      <header className="flex items-center gap-1.5 border-b px-2 py-1.5" title={meta.description}>
        <span className={cn("size-1.5 rounded-full", meta.dotClass)} aria-hidden />
        <span className="text-[10.5px] font-semibold tracking-wide uppercase">{meta.label}</span>
        <span className="ml-auto text-[10.5px] text-muted-foreground tabular-nums">
          {moves.length}
        </span>
      </header>
      <ScrollArea className="min-h-0 flex-1" viewportClassName="min-h-0">
        <div className="flex flex-col gap-1.5 p-1.5">
          {moves.map((move) => (
            <MoveCard
              key={move.moveId}
              move={move}
              isSelected={selectedMoveId === move.moveId}
              onSelect={onSelectMove}
            />
          ))}
          {moves.length === 0 && (
            <p className="py-6 text-center text-[11px] text-muted-foreground">Nothing here.</p>
          )}
        </div>
      </ScrollArea>
    </div>
  );
}

/**
 * Grouping by urgency is the triage a dispatcher performs anyway. Doing it for them means
 * the queue that is already broken is the one they see first.
 */
export function CoverageKanban({
  moves,
  isLoading,
  urgencyFocus,
  selectedMoveId,
  onSelectMove,
}: {
  moves: readonly DispatchBoardMove[];
  isLoading: boolean;
  urgencyFocus: UrgencyBucket | null;
  selectedMoveId: string | null;
  onSelectMove: (moveId: string) => void;
}) {
  const grouped = useMemo(() => groupMovesByUrgency(moves), [moves]);

  const visibleBuckets = urgencyFocus ? [urgencyFocus] : URGENCY_ORDER;

  if (isLoading) {
    return <CoverageKanbanSkeleton />;
  }

  return (
    <div className="flex h-full min-h-0 gap-2 overflow-x-auto">
      {visibleBuckets.map((bucket) => (
        <UrgencyColumn
          key={bucket}
          bucket={bucket}
          moves={grouped.get(bucket) ?? []}
          selectedMoveId={selectedMoveId}
          onSelectMove={onSelectMove}
        />
      ))}
    </div>
  );
}
