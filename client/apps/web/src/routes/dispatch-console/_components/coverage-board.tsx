import type { DispatchBoardMove } from "@/lib/graphql/dispatch-console";
import { useDroppable } from "@dnd-kit/core";
import { Badge } from "@trenova/shared/components/ui/badge";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { FlameIcon, SnowflakeIcon, TriangleAlertIcon } from "lucide-react";
import { useMemo } from "react";
import {
  URGENCY_ORDER,
  formatMiles,
  formatMinutesToPickup,
  formatMoney,
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
  const { setNodeRef, isOver } = useDroppable({
    id: `move:${move.moveId}`,
    data: { type: "move", move },
  });

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
        "flex cursor-pointer flex-col gap-1.5 rounded-md border p-2 transition-colors",
        isSelected ? "border-brand bg-brand/5" : "border-border bg-card hover:border-brand/40",
        isOver && "border-brand ring-2 ring-brand/40",
      )}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="flex min-w-0 items-center gap-1.5">
          <span className="truncate font-mono text-xs font-medium">{move.proNumber}</span>
          {move.moveCount > 1 ? (
            <Badge variant="outline" className="h-4 shrink-0 rounded px-1 text-[9px]">
              Leg {move.sequence + 1}/{move.moveCount}
            </Badge>
          ) : null}
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
          {move.originWindowStart > 0 ? formatUnixDateTime(move.originWindowStart) : "No appointment"}
        </span>
        {move.distance ? (
          <Badge variant="outline" className="h-4 rounded px-1 text-[9px]">
            {formatMiles(move.distance)}
          </Badge>
        ) : null}
        {move.revenue ? (
          <Badge variant="outline" className="h-4 rounded px-1 text-[9px]">
            {formatMoney(move.revenue)}
          </Badge>
        ) : null}
        {move.hasHazmat ? (
          <Badge variant="inactive" className="h-4 rounded px-1 text-[9px]">
            <FlameIcon className="mr-0.5 size-2.5" aria-hidden />
            Hazmat
          </Badge>
        ) : null}
        {move.temperatureMin !== null && move.temperatureMin !== undefined ? (
          <Badge variant="info" className="h-4 rounded px-1 text-[9px]">
            <SnowflakeIcon className="mr-0.5 size-2.5" aria-hidden />
            {move.temperatureMin}–{move.temperatureMax}°F
          </Badge>
        ) : null}
        {move.hasActiveHold ? (
          <Badge variant="inactive" className="h-4 rounded px-1 text-[9px]">
            <TriangleAlertIcon className="mr-0.5 size-2.5" aria-hidden />
            On hold
          </Badge>
        ) : null}
      </div>

      <div className="flex items-center justify-between gap-2">
        <span className="truncate text-[10px] text-muted-foreground">{move.customerName}</span>
        {move.isCovered ? (
          <Badge variant="active" className="h-4 shrink-0 rounded px-1 text-[9px]">
            {move.assignedWorkerName}
            {move.assignedTractorCode ? ` · ${move.assignedTractorCode}` : ""}
          </Badge>
        ) : null}
      </div>
    </div>
  );
}

export function CoverageBoard({
  moves,
  isLoading,
  selectedMoveId,
  onSelectMove,
}: {
  moves: readonly DispatchBoardMove[];
  isLoading: boolean;
  selectedMoveId: string | null;
  onSelectMove: (moveId: string) => void;
}) {
  // Grouping by urgency is the triage a dispatcher performs anyway. Doing it for them
  // means the queue that is already broken is the one they see first.
  const grouped = useMemo(() => {
    const buckets = new Map<UrgencyBucket, DispatchBoardMove[]>();
    for (const bucket of URGENCY_ORDER) {
      buckets.set(bucket, []);
    }
    for (const move of moves) {
      const bucket = (move.urgency as UrgencyBucket) ?? "Planned";
      buckets.get(bucket)?.push(move);
    }
    return buckets;
  }, [moves]);

  return (
    <section className="flex h-full min-h-0 flex-col gap-2 rounded-md border border-border bg-background p-2">
      <header className="flex items-baseline justify-between gap-2">
        <h2 className="text-xs font-semibold uppercase tracking-wide">Coverage</h2>
        <span className="text-[10px] text-muted-foreground">{moves.length} moves</span>
      </header>

      <ScrollArea className="min-h-0 flex-1">
        <div className="flex flex-col gap-3 pr-2">
          {isLoading
            ? Array.from({ length: 6 }, (_, index) => (
                <Skeleton key={index} className="h-28 rounded-md" />
              ))
            : URGENCY_ORDER.map((bucket) => {
                const items = grouped.get(bucket) ?? [];
                if (items.length === 0) return null;
                const meta = urgencyMeta(bucket);

                return (
                  <div key={bucket} className="flex flex-col gap-1.5">
                    <div className="flex items-center gap-1.5">
                      <Badge variant={meta.variant} className="h-4 rounded px-1 text-[9px]">
                        {meta.label}
                      </Badge>
                      <span className="text-[10px] text-muted-foreground">
                        {items.length} · {meta.description}
                      </span>
                    </div>
                    {items.map((move) => (
                      <MoveCard
                        key={move.moveId}
                        move={move}
                        isSelected={selectedMoveId === move.moveId}
                        onSelect={onSelectMove}
                      />
                    ))}
                  </div>
                );
              })}
          {!isLoading && moves.length === 0 ? (
            <p className="px-1 py-8 text-center text-xs text-muted-foreground">
              Nothing needs coverage in this window.
            </p>
          ) : null}
        </div>
      </ScrollArea>
    </section>
  );
}
