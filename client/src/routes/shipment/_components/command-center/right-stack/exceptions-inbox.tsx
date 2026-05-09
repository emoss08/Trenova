import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { formatToUserTimezone } from "@/lib/date";
import { cn } from "@/lib/utils";
import type { Shipment } from "@/types/shipment";
import {
  AlertTriangleIcon,
  ClockIcon,
  FileWarningIcon,
  type LucideIcon,
} from "lucide-react";
import { useMemo, useState } from "react";
import { useCommandCenterUrl } from "../url-state";
import { ModuleCard } from "./module-card";
import { useExceptionShipments } from "./use-right-stack-data";

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

export function ExceptionsInbox() {
  const { data, isLoading } = useExceptionShipments("all");
  const [, setUrl] = useCommandCenterUrl();
  const [dismissed, setDismissed] = useState<Set<string>>(new Set());

  const items = useMemo<DerivedException[]>(() => {
    const list = (data?.results ?? []) as Shipment[];
    return list
      .map(deriveException)
      .filter((x): x is DerivedException => !!x && !dismissed.has(x.id));
  }, [data?.results, dismissed]);

  const dismiss = (id: string) => {
    setDismissed((prev) => {
      const next = new Set(prev);
      next.add(id);
      return next;
    });
  };

  return (
    <ModuleCard
      id="exceptions"
      title="Exceptions"
      count={items.length}
      countTone="danger"
      rightSlot={
        <Button variant="ghost" size="xxs" className="text-muted-foreground">
          Mute · 1h
        </Button>
      }
    >
      {isLoading && (
        <div className="flex items-center gap-2 px-2 py-2 text-[10.5px] text-muted-foreground">
          <Spinner className="size-3" /> Loading…
        </div>
      )}
      {!isLoading && items.length === 0 && (
        <p className="px-2 py-4 text-center text-[10.5px] text-muted-foreground">All clear ✓</p>
      )}
      {items.map((it, i) => {
        const Icon = KIND_ICON[it.kind];
        const sevTextClass = it.severity === "danger" ? "text-destructive" : "text-warning";
        const sevBgClass = it.severity === "danger" ? "bg-destructive/12" : "bg-warning/15";
        return (
          <div
            key={it.id}
            className={cn(
              "flex items-start gap-2 px-2.5 py-2",
              i > 0 && "border-t border-border",
            )}
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
                  onClick={() => setUrl({ expanded: it.shipmentId })}
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
    </ModuleCard>
  );
}
