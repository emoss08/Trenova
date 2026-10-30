import { VirtualRows, type VirtualRowsHandle } from "@/components/virtual-rows";
import type { DispatchBoardMove } from "@/lib/graphql/dispatch-console";
import { useDroppable } from "@dnd-kit/core";
import {
  AlertTriangleIcon,
  Building07Icon,
  FlameIcon,
  Send01Icon,
  Snowflake01Icon,
} from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { ScrollArea, ScrollBar } from "@trenova/shared/components/ui/scroll-area";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { cn, formatCompactCurrency } from "@trenova/shared/lib/utils";
import { useEffect, useMemo, useRef } from "react";
import { CoverageKanbanSkeleton } from "./console-skeletons";
import {
  URGENCY_ORDER,
  formatMiles,
  formatMinutesToPickup,
  groupMovesByUrgency,
  urgencyMeta,
  type UrgencyBucket,
} from "./dispatch-vocabulary";
import { formatOfferCountdown, tenderChipMeta, type TenderChipTone } from "./tender-vocabulary";

// A card with one row of badges, plus the gap above it.
const MOVE_CARD_ESTIMATE_PX = 114;

const TENDER_CHIP_VARIANT: Record<TenderChipTone, "info" | "success" | "warning"> = {
  info: "info",
  active: "success",
  attention: "warning",
};

function TenderChip({ move }: { move: DispatchBoardMove }) {
  const t = useT();
  const nowSeconds = useNow("minute");

  const summary = move.liveTender;
  if (!summary) return null;

  const meta = tenderChipMeta(summary);
  const countdown =
    summary.status === "Active"
      ? formatOfferCountdown(summary.currentOfferExpiresAt, nowSeconds)
      : "";

  return (
    <Badge variant={TENDER_CHIP_VARIANT[meta.tone]} className="h-4 rounded-md px-1 text-3xs">
      <Send01Icon className="mr-0.5 size-2.5" aria-hidden />
      {t(meta.label)}
      {countdown ? ` · ${countdown}` : ""}
    </Badge>
  );
}

function MoveCard({
  move,
  isSelected,
  onSelect,
}: {
  move: DispatchBoardMove;
  isSelected: boolean;
  onSelect: (moveId: string) => void;
}) {
  const t = useT();

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
        "bg-card flex cursor-pointer flex-col gap-1.5 rounded-md border p-2 transition-[border-color,box-shadow]",
        isSelected ? "border-brand shadow-[0_0_0_1px_var(--brand)]" : "hover:border-brand/40",
        isOver && isDriverDrag && "border-brand bg-brand/5 shadow-[0_0_0_2px_var(--brand)]",
      )}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="flex min-w-0 items-center gap-1.5">
          <span className="truncate font-mono text-xs font-semibold">{move.proNumber}</span>
          {/* A reefer reads from the mark alone; its range sits behind it so the badge row
              never wraps to a second line and makes the card taller. */}
          {move.temperatureMin != null && (
            <Tooltip>
              <TooltipTrigger
                render={
                  <span
                    role="img"
                    aria-label={t("Reefer {0}–{1}°F", move.temperatureMin, move.temperatureMax)}
                    className="text-info-foreground flex shrink-0 cursor-help"
                  >
                    <Snowflake01Icon className="size-3" aria-hidden />
                  </span>
                }
              />
              <TooltipContent>
                {t("Reefer {0}–{1}°F", move.temperatureMin, move.temperatureMax)}
              </TooltipContent>
            </Tooltip>
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
        </div>
        <span
          className={cn(
            "shrink-0 text-2xs tabular-nums",
            move.minutesToPickup < 0
              ? "font-medium text-danger-foreground"
              : "text-muted-foreground",
          )}
        >
          {move.originWindowStart > 0 ? formatMinutesToPickup(move.minutesToPickup) : "unscheduled"}
        </span>
      </div>

      <div className="flex items-center gap-1 text-xs">
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
        <span className="text-muted-foreground text-2xs">
          {move.originWindowStart > 0
            ? formatUnixDateTime(move.originWindowStart)
            : t("No appointment")}
        </span>
        {move.distance != null && (
          <Badge variant="neutral" appearance="outline" className="h-4 rounded-md px-1 text-3xs">
            {formatMiles(move.distance)}
          </Badge>
        )}
        {move.revenue != null && (
          <Badge variant="neutral" appearance="outline" className="h-4 rounded-md px-1 text-3xs">
            {formatCompactCurrency(move.revenue)}
          </Badge>
        )}
        {move.hasHazmat && (
          <Badge variant="danger" className="h-4 rounded-md px-1 text-3xs">
            <FlameIcon className="mr-0.5 size-2.5" aria-hidden />
            {t("Hazmat")}
          </Badge>
        )}
        {move.hasActiveHold && (
          <Badge variant="danger" className="h-4 rounded-md px-1 text-3xs">
            <AlertTriangleIcon className="mr-0.5 size-2.5" aria-hidden />
            {t("On hold")}
          </Badge>
        )}
        <TenderChip move={move} />
      </div>

      <div className="flex items-center justify-between gap-2">
        <span className="text-muted-foreground truncate text-2xs">{move.customerName}</span>
        {move.isCovered &&
          (move.coverageType === "carrier" ? (
            <Badge variant="success" className="h-4 shrink-0 rounded-md px-1 text-3xs">
              <Building07Icon className="mr-0.5 size-2.5" aria-hidden />
              {move.assignedCarrierName}
              {move.carrierTotalCost != null
                ? ` · ${formatCompactCurrency(move.carrierTotalCost)}`
                : ""}
            </Badge>
          ) : (
            <Badge variant="success" className="h-4 shrink-0 rounded-md px-1 text-3xs">
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
  const t = useT();

  const meta = urgencyMeta(bucket);

  // Only the cards near the viewport are mounted, so a selection made elsewhere (the
  // keyboard, a drop) is scrolled to rather than left on a card that may not exist yet.
  const listRef = useRef<VirtualRowsHandle>(null);
  useEffect(() => {
    if (selectedMoveId) listRef.current?.scrollToKey(selectedMoveId);
  }, [selectedMoveId]);

  return (
    <div className="bg-muted/30 flex min-h-60 w-64 max-w-md shrink-0 grow flex-col rounded-md border">
      <header
        className="flex items-center gap-1.5 border-b px-2 py-1.5"
        title={t(meta.description)}
      >
        <span className={cn("size-1.5 rounded-full", meta.dotClass)} aria-hidden />
        <span className="text-xs font-semibold">{t(meta.label)}</span>
        <span className="text-muted-foreground ml-auto text-2xs tabular-nums">{moves.length}</span>
      </header>
      <VirtualRows
        rows={moves.map((move) => ({
          key: move.moveId,
          render: () => (
            <MoveCard
              move={move}
              isSelected={selectedMoveId === move.moveId}
              onSelect={onSelectMove}
            />
          ),
        }))}
        handleRef={listRef}
        mask="card"
        className="min-h-0 flex-1"
        rowClassName="px-1.5 pt-1.5"
        estimateSize={MOVE_CARD_ESTIMATE_PX}
        footer={<div className="h-1.5" aria-hidden />}
        empty={
          <p className="text-muted-foreground py-6 text-center text-xs">{t("Nothing here.")}</p>
        }
      />
    </div>
  );
}

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
    <ScrollArea className="h-full" maskVariant="card">
      {/* Columns share the width on a wide screen and never drop below a usable width on a
          narrow one, where the row overflows into the horizontal scroll instead. */}
      <div className="flex h-full gap-2 pb-3">
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
      <ScrollBar orientation="horizontal" />
    </ScrollArea>
  );
}
