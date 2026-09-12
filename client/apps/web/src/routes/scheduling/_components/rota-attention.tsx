import { useT } from "@trenova/shared/i18n/use-t";
import type { RotaBoardRow } from "@/lib/graphql/scheduling";
import {
  rotaConflicts,
  summarizeSwaps,
  unrosteredWorkers,
  type RotaConflict,
  type UnrosteredWorker,
} from "@/lib/scheduling-board";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { formatShiftDate, rotaStateTone } from "@trenova/shared/lib/scheduling";
import { AlertTriangleIcon, ChevronRightIcon, CircleCheckIcon, RepeatIcon } from "lucide-react";
import { m, useReducedMotion } from "motion/react";
import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router";

const SHOWN_LIMIT = 6;
const STAGGER_LIMIT = 8;

function workerHref(workerId: string): string {
  return `/hr/workers?entityId=${workerId}&modType=edit`;
}

type RotaAttentionProps = {
  rows: readonly RotaBoardRow[];
  swaps: readonly { status: string }[] | undefined;
  onOpenSwaps?: () => void;
};

/**
 * What on the board needs a hand: a day somebody is rostered onto that they
 * cannot work, a person with nothing to work, and a swap two drivers have
 * agreed that the office has not answered. The board shows all three as
 * cells and badges; this is the same three as a list somebody can work down.
 */
export function RotaAttention({ rows, swaps, onOpenSwaps }: RotaAttentionProps) {
  const t = useT();

  const conflicts = useMemo(() => rotaConflicts(rows), [rows]);
  const unrostered = useMemo(() => unrosteredWorkers(rows), [rows]);
  const swapSummary = useMemo(() => summarizeSwaps(swaps ?? []), [swaps]);
  const total = conflicts.length + unrostered.length + swapSummary.awaitingOffice;

  return (
    <section
      aria-labelledby="rota-attention-heading"
      className="bg-card overflow-hidden rounded-lg border"
    >
      <header className="flex items-center justify-between gap-2 border-b px-3 py-2">
        <div className="flex items-center gap-2">
          {total > 0 ? (
            <AlertTriangleIcon className="text-destructive size-3.5" aria-hidden />
          ) : (
            <CircleCheckIcon className="text-muted-foreground size-3.5" aria-hidden />
          )}
          <h3 id="rota-attention-heading" className="text-sm font-medium">
            {t("Needs a look")}
          </h3>
          {total > 0 ? (
            <Badge variant="inactive" className="text-2xs h-4 px-1 tabular-nums">
              {total}
            </Badge>
          ) : null}
        </div>
        {swapSummary.awaitingOffice > 0 && onOpenSwaps ? (
          <Button size="xs" variant="outline" onClick={onOpenSwaps}>
            <RepeatIcon className="size-3" />
            {t("{0, plural, one {# swap} other {# swaps}} to decide", swapSummary.awaitingOffice)}
          </Button>
        ) : null}
      </header>

      {total === 0 ? (
        <p className="text-muted-foreground px-3 py-3 text-sm">
          {t("Everyone on the board has a shift they can work, and nothing is waiting on you.")}
        </p>
      ) : (
        <AttentionList conflicts={conflicts} unrostered={unrostered} />
      )}
    </section>
  );
}

function AttentionList({
  conflicts,
  unrostered,
}: {
  conflicts: RotaConflict[];
  unrostered: UnrosteredWorker[];
}) {
  const t = useT();

  const reduceMotion = useReducedMotion();
  const [settled, setSettled] = useState(false);
  const [expanded, setExpanded] = useState(false);

  useEffect(() => {
    const timer = window.setTimeout(() => setSettled(true), 600);
    return () => window.clearTimeout(timer);
  }, []);

  const items = [
    ...conflicts.map((conflict) => ({
      key: `conflict-${conflict.workerId}-${conflict.date}`,
      workerId: conflict.workerId,
      name: conflict.name,
      detail: `${formatShiftDate(conflict.date)} · rostered while ${rotaStateTone(conflict.state).label.toLowerCase()}`,
      badge: "Conflict",
      tone: "inactive" as const,
    })),
    ...unrostered.map((worker) => ({
      key: `unrostered-${worker.workerId}`,
      workerId: worker.workerId,
      name: worker.name,
      detail: worker.shiftName
        ? `On ${worker.shiftName}, with no working day this week`
        : `On no pattern${worker.fleetCode ? ` · ${worker.fleetCode}` : ""}`,
      badge: "No shift",
      tone: "warning" as const,
    })),
  ];
  const shown = expanded ? items : items.slice(0, SHOWN_LIMIT);
  const hidden = items.length - shown.length;

  return (
    <>
      <ul className="divide-y">
        {shown.map((item, index) => (
          <m.li
            key={item.key}
            initial={settled || reduceMotion ? false : { opacity: 0, y: 4 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.25, delay: Math.min(index, STAGGER_LIMIT) * 0.03 }}
          >
            <Link
              to={workerHref(item.workerId)}
              className="group/row hover:bg-accent grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 px-3 py-2 transition-colors"
            >
              <span className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1">
                <span className="text-sm font-medium">{item.name}</span>
                <span className="text-muted-foreground text-xs">{item.detail}</span>
              </span>
              <span className="flex items-center gap-2">
                <Badge variant={item.tone}>{item.badge}</Badge>
                <ChevronRightIcon
                  aria-hidden
                  className="text-muted-foreground size-4 transition-transform group-hover/row:translate-x-0.5"
                />
              </span>
            </Link>
          </m.li>
        ))}
      </ul>
      {hidden > 0 || expanded ? (
        <div className="border-t px-3 py-1.5">
          <Button
            size="xs"
            variant="ghost"
            className="text-muted-foreground -ml-2"
            onClick={() => setExpanded((value) => !value)}
          >
            {expanded ? t("Show fewer") : t("Show {0} more", hidden)}
          </Button>
        </div>
      ) : null}
    </>
  );
}
