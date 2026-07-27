import { Button } from "@trenova/shared/components/ui/button";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { Shipment } from "@trenova/shared/types/shipment";
import { AlertTriangleIcon, ClockIcon, FileWarningIcon, type LucideIcon } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useExceptionShipments } from "./use-work-queues";

type ExceptionKind = "eta-slip" | "detention" | "doc-issue";
type Severity = "danger" | "warning";

type DerivedException = {
  id: string;
  shipmentId: string;
  proNumber: string;
  customer: string;
  kind: ExceptionKind;
  severity: Severity;
  title: string;
  body: string;
  time: string;
  actionLabel: string;
};

const KIND_ICON: Record<ExceptionKind, LucideIcon> = {
  "eta-slip": AlertTriangleIcon,
  detention: ClockIcon,
  "doc-issue": FileWarningIcon,
};

function severityFor(s: Shipment): Severity {
  return s.status === "Delayed" ? "danger" : "warning";
}

function kindFor(s: Shipment): ExceptionKind {
  if (s.billingTransferStatus === "SentBackToOps") return "doc-issue";
  // Detention if any stop has actualArrival without actualDeparture (dwell).
  const dwellingStop = s.moves
    ?.flatMap((m) => m.stops ?? [])
    .find((stop) => stop.actualArrival && !stop.actualDeparture);
  if (dwellingStop) return "detention";
  return "eta-slip";
}

function titleFor(kind: ExceptionKind): string {
  switch (kind) {
    case "eta-slip":
      return "ETA slip";
    case "detention":
      return "Detention dwell";
    case "doc-issue":
      return "Docs sent back";
  }
}

function actionFor(kind: ExceptionKind): string {
  switch (kind) {
    case "eta-slip":
      return "Re-quote ETA";
    case "detention":
      return "Bill detention";
    case "doc-issue":
      return "Resolve docs";
  }
}

function relativeAgo(ts: number | null | undefined): string {
  if (!ts) return "—";
  return formatToUserTimezone(ts, { showTimeZone: false, showSeconds: false });
}

function deriveException(s: Shipment): DerivedException | null {
  if (!s.id) return null;
  const kind = kindFor(s);
  return {
    id: s.id,
    shipmentId: s.id,
    proNumber: s.proNumber || s.id,
    customer: s.customer?.name ?? "—",
    kind,
    severity: severityFor(s),
    title: titleFor(kind),
    body: `${s.proNumber || s.id} · ${s.customer?.name ?? "—"}`,
    time: relativeAgo(s.updatedAt),
    actionLabel: actionFor(kind),
  };
}

export type ExceptionsListProps = {
  enabled?: boolean;
  limit?: number;
  onSelect: (shipmentId: string) => void;
  onCount?: (count: number) => void;
};

/**
 * The at-risk shipment inbox, without chrome. Dismissals are local to the
 * surface showing them: dismissing on the home screen is a way to clear your
 * own view, not a change to the shipment.
 */
export function ExceptionsList({ enabled = true, limit, onSelect, onCount }: ExceptionsListProps) {
  const { data, isLoading } = useExceptionShipments("all", undefined, enabled);
  const [dismissed, setDismissed] = useState<Set<string>>(new Set());

  const items = useMemo<DerivedException[]>(() => {
    const list = (data?.results ?? []) as Shipment[];
    return list
      .map(deriveException)
      .filter((x): x is DerivedException => !!x && !dismissed.has(x.id));
  }, [data?.results, dismissed]);

  useEffect(() => {
    onCount?.(items.length);
  }, [items.length, onCount]);

  const dismiss = (id: string) => {
    setDismissed((prev) => {
      const next = new Set(prev);
      next.add(id);
      return next;
    });
  };

  const visible = limit ? items.slice(0, limit) : items;

  return (
    <>
      {isLoading && (
        <div className="flex items-center gap-2 px-2 py-2 text-[10.5px] text-muted-foreground">
          <Spinner className="size-3" /> Loading…
        </div>
      )}
      {!isLoading && visible.length === 0 && (
        <p className="px-2 py-4 text-center text-[10.5px] text-muted-foreground">All clear ✓</p>
      )}
      {visible.map((it, i) => {
        const Icon = KIND_ICON[it.kind];
        const sevTextClass = it.severity === "danger" ? "text-destructive" : "text-warning";
        const sevBgClass = it.severity === "danger" ? "bg-destructive/12" : "bg-warning/15";
        return (
          <div
            key={it.id}
            className={cn("flex items-start gap-2 px-2.5 py-2", i > 0 && "border-t border-border")}
          >
            <span
              className={cn(
                "flex size-5 shrink-0 items-center justify-center rounded",
                sevBgClass,
                sevTextClass,
              )}
              aria-hidden
            >
              <Icon className="size-3" />
            </span>
            <div className="flex min-w-0 flex-1 flex-col gap-0.5">
              <div className="flex items-baseline justify-between gap-2">
                <span className="truncate text-[11px] font-semibold">{it.title}</span>
                <span className="shrink-0 font-table text-[9.5px] text-muted-foreground tabular-nums">
                  {it.time}
                </span>
              </div>
              <p className="truncate font-table text-[10px] text-muted-foreground tabular-nums">
                {it.body}
              </p>
              <div className="mt-0.5 flex items-center gap-1">
                <Button
                  variant="outline"
                  size="xxs"
                  className={cn("border-current", sevTextClass)}
                  onClick={() => onSelect(it.shipmentId)}
                >
                  {it.actionLabel}
                </Button>
                <Button
                  variant="ghost"
                  size="xxs"
                  className="text-muted-foreground"
                  onClick={() => dismiss(it.id)}
                >
                  Dismiss
                </Button>
              </div>
            </div>
          </div>
        );
      })}
    </>
  );
}
