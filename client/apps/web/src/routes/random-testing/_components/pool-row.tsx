import { useT } from "@trenova/shared/i18n/use-t";
import type { RandomPoolRow } from "@/lib/graphql/worker-drug-alcohol";
import {
  SLOT_STATE_LABELS,
  type CalendarSlot,
  type PoolProgress,
  type SlotState,
} from "@/lib/random-testing";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { randomPeriodLabel } from "@trenova/shared/lib/drug-alcohol";
import { cn } from "@trenova/shared/lib/utils";
import { DicesIcon, ListFilterIcon, PencilIcon } from "lucide-react";

const SLOT_CLASS: Record<SlotState, string> = {
  final: "border-brand bg-brand/15 font-medium",
  draft: "border-brand/60 border-dashed bg-brand/5 font-medium",
  due: "border-foreground/40 border-dashed",
  missed: "border-destructive/50 bg-destructive/10 text-destructive",
  upcoming: "text-muted-foreground",
};

type PoolRowProps = {
  pool: RandomPoolRow;
  calendar: readonly CalendarSlot[];
  progress: PoolProgress;
  year: number;
  selected: boolean;
  canDraw: boolean;
  canEdit: boolean;
  drawing: boolean;
  onSelect: (poolId: string | null) => void;
  onDraw: (poolId: string) => void;
  onEdit: (pool: RandomPoolRow) => void;
};

/**
 * One pool as a full-width row: what it draws from and how often, the
 * year's rounds as a strip of slots, how the collections stand, and the
 * actions. A missed slot is a round that was never drawn, which is the
 * finding a DOT audit turns up first, so it is the one thing on the row
 * allowed a warning colour.
 */
export function PoolRow({
  pool,
  calendar,
  progress,
  year,
  selected,
  canDraw,
  canEdit,
  drawing,
  onSelect,
  onDraw,
  onEdit,
}: PoolRowProps) {
  const t = useT();

  const owed = calendar.find((slot) => slot.state === "due");
  const drivers =
    pool.includedDriverTypes.length > 0 ? pool.includedDriverTypes.join(", ") : "every driver";

  return (
    <li
      aria-label={pool.name}
      className={cn(
        "grid gap-x-6 gap-y-2 px-3 py-2.5 transition-colors lg:grid-cols-[minmax(0,1.2fr)_auto_minmax(0,1fr)_auto] lg:items-center",
        selected && "bg-accent/60",
      )}
    >
      <div className="flex min-w-0 flex-col gap-0.5">
        <div className="flex min-w-0 flex-wrap items-center gap-1.5">
          <span className="truncate text-sm font-medium">{pool.name}</span>
          <Badge variant="secondary">{pool.code}</Badge>
          {pool.isDefault ? <Badge variant="outline">{t("Default")}</Badge> : null}
          {pool.status !== "Active" ? <Badge variant="inactive">{t("Inactive")}</Badge> : null}
          {pool.meetsDotMinimums ? null : <Badge variant="warning">{t("Below the DOT minimum")}</Badge>}
        </div>
        <p className="text-muted-foreground truncate text-xs">
          {t("{0} · {1}% drug · {2}% alcohol · {3}", randomPeriodLabel(pool.period), pool.drugRatePercent, pool.alcoholRatePercent, drivers)}
        </p>
        {pool.description ? (
          <p className="text-muted-foreground truncate text-xs">{t(pool.description)}</p>
        ) : null}
      </div>

      <ol aria-label={`Rounds in ${year}`} className="flex flex-wrap gap-1">
        {calendar.map((slot) => (
          <li key={slot.key}>
            <Tooltip>
              <TooltipTrigger
                render={
                  <span
                    role="img"
                    aria-label={`${slot.label}: ${SLOT_STATE_LABELS[slot.state]}`}
                    className={cn(
                      "flex h-7 w-9 items-center justify-center rounded-md border text-[11px] tabular-nums",
                      SLOT_CLASS[slot.state],
                    )}
                  />
                }
              >
                {t(slot.label)}
              </TooltipTrigger>
              <TooltipContent className="text-xs">
                {slot.key} · {SLOT_STATE_LABELS[slot.state]}
                {slot.draw
                  ? t("· {0}/{1} drug, {2}/{3} alcohol", slot.draw.drugSelected, slot.draw.drugTarget, slot.draw.alcoholSelected, slot.draw.alcoholTarget)
                  : ""}
                {slot.voided > 0 ? t("· {0} voided", slot.voided) : ""}
              </TooltipContent>
            </Tooltip>
          </li>
        ))}
      </ol>

      <p className="text-muted-foreground min-w-0 text-xs tabular-nums">
        {progress.rounds === 0 ? (
          t("Nothing drawn this year")
        ) : (
          <>
            {t("Drug {0} of {1} · Alcohol {2} of {3} {4}", progress.drugSelected, progress.drugTarget, progress.alcoholSelected, progress.alcoholTarget, progress.onPace ? "" : t("· a round fell short"))}
          </>
        )}
      </p>

      <div className="flex shrink-0 items-center justify-end gap-1.5">
        <Button
          size="xs"
          variant={selected ? "default" : "ghost"}
          aria-pressed={selected}
          aria-label={selected ? "Show every pool's rounds" : `Show rounds for ${pool.code}`}
          onClick={() => onSelect(selected ? null : pool.id)}
        >
          <ListFilterIcon className="size-3" />
          {t("Rounds")}
        </Button>
        {canEdit ? (
          <Button
            size="xs"
            variant="outline"
            aria-label={`Edit ${pool.code}`}
            onClick={() => onEdit(pool)}
          >
            <PencilIcon className="size-3" />
            {t("Edit")}
          </Button>
        ) : null}
        {canDraw && pool.status === "Active" ? (
          <Button
            size="xs"
            variant={owed ? "default" : "outline"}
            isLoading={drawing}
            onClick={() => onDraw(pool.id)}
          >
            <DicesIcon className="size-3" />
            {owed ? t("Draw {0}", owed.label) : t("Run draw")}
          </Button>
        ) : null}
      </div>
    </li>
  );
}
