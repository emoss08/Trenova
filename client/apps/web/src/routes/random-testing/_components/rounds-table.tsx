import { useT } from "@trenova/shared/i18n/use-t";
import type { RandomDrawListRow } from "@/lib/graphql/worker-drug-alcohol";
import { drawShortOfTarget, roundStatusLabel, roundStatusTone } from "@/lib/random-testing";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { m, useReducedMotion } from "motion/react";
import { useEffect, useState } from "react";

const STAGGER_LIMIT = 8;

type RoundsTableProps = {
  rows: readonly RandomDrawListRow[];
  canDraw: boolean;
  /** The round a finalise or void is in flight for, so only its buttons spin. */
  busyId: string | null;
  onOpen: (drawId: string) => void;
  onFinalise: (draw: RandomDrawListRow) => void;
  onVoid: (draw: RandomDrawListRow) => void;
};

/**
 * Every round drawn, newest first. A draft round can still be finalised or
 * voided from here; a final one is the record and only opens.
 */
export function RoundsTable({
  rows,
  canDraw,
  busyId,
  onOpen,
  onFinalise,
  onVoid,
}: RoundsTableProps) {
  const t = useT();

  const reduceMotion = useReducedMotion();
  const [settled, setSettled] = useState(false);

  useEffect(() => {
    const timer = window.setTimeout(() => setSettled(true), 600);
    return () => window.clearTimeout(timer);
  }, []);

  return (
    <Table aria-label={t("Rounds")} className="text-xs">
      <TableHeader>
        <TableRow className="hover:bg-transparent">
          <TableHead className="h-8 px-3">{t("Round")}</TableHead>
          <TableHead className="h-8">{t("Pool")}</TableHead>
          <TableHead className="h-8">{t("Status")}</TableHead>
          <TableHead className="h-8">{t("Drug")}</TableHead>
          <TableHead className="h-8">{t("Alcohol")}</TableHead>
          <TableHead className="h-8 text-right">{t("In the hat")}</TableHead>
          <TableHead className="h-8">{t("Drawn")}</TableHead>
          <TableHead className="h-8 px-3 text-right">
            <span className="sr-only">{t("Actions")}</span>
          </TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {rows.map((draw, index) => {
          const short = draw.status !== "Cancelled" && drawShortOfTarget(draw);
          const busy = busyId === draw.id;
          return (
            <m.tr
              key={draw.id}
              className="hover:bg-muted/50 border-b transition-colors"
              initial={settled || reduceMotion ? false : { opacity: 0, y: 4 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.25, delay: Math.min(index, STAGGER_LIMIT) * 0.03 }}
            >
              <TableCell className="px-3 py-2">
                <button
                  type="button"
                  className="font-mono font-medium tabular-nums hover:underline"
                  onClick={() => onOpen(draw.id)}
                >
                  {draw.periodKey}
                </button>
                <span className="text-muted-foreground block tabular-nums">
                  {formatUnixDate(draw.periodStart)} – {formatUnixDate(draw.periodEnd)}
                </span>
              </TableCell>
              <TableCell className="py-2">
                {draw.pool ? (
                  <span className="flex min-w-0 items-center gap-1.5">
                    <Badge variant="secondary">{draw.pool.code}</Badge>
                    <span className="text-muted-foreground truncate">{draw.pool.name}</span>
                  </span>
                ) : (
                  <span className="text-muted-foreground">—</span>
                )}
              </TableCell>
              <TableCell className="py-2">
                <span className="flex items-center gap-1.5">
                  <Badge variant={roundStatusTone(draw.status)}>
                    {roundStatusLabel(draw.status)}
                  </Badge>
                  {short ? <Badge variant="warning">{t("Short")}</Badge> : null}
                </span>
              </TableCell>
              <TableCell className="py-2">
                <Fill
                  label={t("Drug")}
                  selected={draw.drugSelected}
                  target={draw.drugTarget}
                  muted={draw.status === "Cancelled"}
                />
              </TableCell>
              <TableCell className="py-2">
                <Fill
                  label={t("Alcohol")}
                  selected={draw.alcoholSelected}
                  target={draw.alcoholTarget}
                  muted={draw.status === "Cancelled"}
                />
              </TableCell>
              <TableCell className="py-2 text-right tabular-nums">{draw.poolSize}</TableCell>
              <TableCell className="text-muted-foreground py-2 tabular-nums">
                {formatUnixDate(draw.drawnAt)}
              </TableCell>
              <TableCell className="px-3 py-2">
                <span className="flex items-center justify-end gap-1">
                  <Button
                    size="xs"
                    variant="ghost"
                    aria-label={`Open ${draw.periodKey}`}
                    onClick={() => onOpen(draw.id)}
                  >
                    {t("Open")}
                  </Button>
                  {canDraw && draw.status === "Draft" ? (
                    <>
                      <Button
                        size="xs"
                        variant="outline"
                        isLoading={busy}
                        aria-label={`Finalise ${draw.periodKey}`}
                        onClick={() => onFinalise(draw)}
                      >
                        {t("Finalise")}
                      </Button>
                      <Button
                        size="xs"
                        variant="ghost"
                        disabled={busy}
                        aria-label={`Void ${draw.periodKey}`}
                        onClick={() => onVoid(draw)}
                      >
                        {t("Void")}
                      </Button>
                    </>
                  ) : null}
                </span>
              </TableCell>
            </m.tr>
          );
        })}
      </TableBody>
    </Table>
  );
}

type FillProps = {
  label: string;
  selected: number;
  target: number;
  muted: boolean;
};

function Fill({ label, selected, target, muted }: FillProps) {
  const share = target > 0 ? Math.min(100, Math.round((selected / target) * 100)) : 0;
  return (
    <span
      role="img"
      aria-label={`${label}: ${selected} of ${target}`}
      className="flex min-w-24 items-center gap-2"
    >
      <span className="font-mono tabular-nums">
        {selected}
        <span className="text-muted-foreground">/{target}</span>
      </span>
      <span className="bg-muted flex h-1 w-12 overflow-hidden rounded-full">
        <span
          aria-hidden
          className={cn(
            "h-full rounded-full",
            muted ? "bg-muted-foreground/40" : selected < target ? "bg-brand/50" : "bg-brand",
          )}
          style={{ width: `${share}%` }}
        />
      </span>
    </span>
  );
}
