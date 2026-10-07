import { useT } from "@trenova/shared/i18n/use-t";
import { CheckIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import { formatAltShortcut, formatShortcut } from "@trenova/shared/lib/shortcuts";
import { cn } from "@trenova/shared/lib/utils";
import type { ShipmentSuggestionTone } from "@trenova/graphql/generated/graphql";
import { useUserTimezone } from "@/hooks/use-user-timezone";
import type { ShipmentSuggestion } from "@/lib/graphql/shipment-board";
import { isTypingTarget, isWithinDialog } from "@/lib/dom";
import { queries } from "@/lib/queries";
import { useShipmentCapabilities } from "@/lib/shipment-board/capabilities";
import { useQuery } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { useBoardActions } from "../use-board-actions";
import { useShipmentBoardUrl } from "../url-state";

const TONE_DOT: Record<ShipmentSuggestionTone, string> = {
  Danger: "bg-danger",
  Warning: "bg-warning",
  Accent: "bg-accent-teal",
  Brand: "bg-brand",
};

const KIND_LABEL: Record<ShipmentSuggestion["kind"], string> = {
  Coverage: "Coverage",
  Tender: "Tender",
  DelayNotice: "Delay notice",
  HoursOfService: "Hours of service",
  Detention: "Detention",
  Retender: "Tender",
};

type Handled = { key: string; summary: string };

const due = (unix: number | null | undefined) =>
  unix ? formatToUserTimezone(unix, { showTimeZone: false, showSeconds: false, showDate: false }) : null;

/**
 * The board's suggested actions, one at a time. The current item states what
 * to do and why; approving runs it, Later moves it to the back, and the last
 * thing done can be undone. Without an AI provider the same queue lists
 * rule-based exceptions with manual actions.
 */
export function ActionQueue() {
  const t = useT();
  const { ai } = useShipmentCapabilities();
  const timezone = useUserTimezone();
  const [, setUrl] = useShipmentBoardUrl();
  const actions = useBoardActions();
  const { data, isLoading } = useQuery({ ...queries.shipmentBoard.suggestions(timezone), staleTime: 15_000 });
  const [handled, setHandled] = useState<Handled[]>([]);
  const [lastDone, setLastDone] = useState<Handled | null>(null);

  const items = data?.items ?? [];
  const current = items[0];
  const upNext = items.slice(1, 4);
  const doneCount = handled.length;
  const total = doneCount + items.length;
  const pending =
    actions.assign.isPending ||
    actions.tender.isPending ||
    actions.notifyDelay.isPending ||
    actions.approveDetention.isPending ||
    actions.decide.isPending;

  const markDone = (item: ShipmentSuggestion, summary: string) => {
    actions.decide.mutate({ key: item.key, decision: "Done" });
    const record = { key: item.key, summary };
    setHandled((list) => [...list, record]);
    setLastDone(record);
  };

  const review = (item: ShipmentSuggestion) => {
    if (item.shipmentId) {
      void setUrl({ view: "table", expanded: item.shipmentId, qf: [] });
    }
  };

  const approve = (item: ShipmentSuggestion) => {
    const { primary } = item;
    const summary = item.title;
    const finish = { onSuccess: () => markDone(item, summary) };
    switch (primary.type) {
      case "AssignDriver":
        if (primary.moveId && primary.workerId) {
          actions.assign.mutate(
            [{ moveId: primary.moveId, workerId: primary.workerId, tractorId: primary.tractorId }],
            finish,
          );
        }
        return;
      case "TenderCarrier":
        if (item.shipmentId) {
          actions.tender.mutate(
            [{ shipmentId: item.shipmentId, carrierId: primary.carrierId ?? undefined }],
            finish,
          );
        }
        return;
      case "NotifyCustomer":
        if (item.shipmentId && primary.message) {
          actions.notifyDelay.mutate({ shipmentId: item.shipmentId, message: primary.message }, finish);
        }
        return;
      case "ApproveDetention":
        if (primary.detentionOccurrenceId) {
          actions.approveDetention.mutate(primary.detentionOccurrenceId, finish);
        }
        return;
      case "Review":
        review(item);
        markDone(item, summary);
        return;
    }
  };

  const later = (item: ShipmentSuggestion) => actions.decide.mutate({ key: item.key, decision: "Later" });

  const undo = (record: Handled) => {
    actions.undo.mutate(record.key);
    setHandled((list) => list.filter((entry) => entry.key !== record.key));
    setLastDone(null);
  };

  const latest = useRef({ current, approve, later, pending });
  latest.current = { current, approve, later, pending };

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (isTypingTarget(event.target) || isWithinDialog(event.target)) return;
      const { current: item, pending: busy } = latest.current;
      if (!item || busy) return;
      if ((event.metaKey || event.ctrlKey) && event.key === "Enter") {
        event.preventDefault();
        latest.current.approve(item);
      } else if (event.altKey && event.code === "KeyL" && !event.metaKey && !event.ctrlKey) {
        event.preventDefault();
        latest.current.later(item);
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, []);

  if (isLoading) {
    return (
      <div className="flex flex-col gap-2">
        <Skeleton className="h-3 w-40" />
        <Skeleton className="h-20 w-full" />
      </div>
    );
  }

  return (
    <section aria-label={ai ? t("Suggested actions") : t("Exceptions")} className="flex flex-col gap-3">
      <header className="flex items-center justify-between">
        <span className="text-muted-foreground text-xs font-medium">
          {ai ? t("Suggested actions") : t("Exceptions")}
        </span>
        {current ? (
          <span className="text-muted-foreground font-mono text-xs tabular-nums">
            {t("{0} of {1}", doneCount + 1, total)}
          </span>
        ) : null}
      </header>
      {total > 0 ? (
        <div className="flex gap-1" aria-hidden>
          {Array.from({ length: total }, (_, index) => (
            <span
              key={index}
              className={cn(
                "h-[3px] flex-1 rounded-full",
                index < doneCount ? "bg-success" : index === doneCount ? "bg-foreground" : "bg-foreground/10",
              )}
            />
          ))}
        </div>
      ) : null}

      {lastDone ? (
        <div className="text-success flex items-center gap-1.5 text-sm">
          <CheckIcon className="size-3.5 shrink-0" />
          <span className="min-w-0 flex-1 truncate">{lastDone.summary}</span>
          <button type="button" className="ui-focus-ring text-foreground rounded-sm underline underline-offset-2" onClick={() => undo(lastDone)}>
            {t("Undo")}
          </button>
        </div>
      ) : null}

      {current ? (
        <article key={current.key} className="animate-slide-in flex flex-col gap-2">
          <div className="text-muted-foreground flex items-center gap-1.5 text-xs">
            <span aria-hidden className={cn("size-1.5 rounded-full", TONE_DOT[current.tone])} />
            <span className="font-medium">{t(KIND_LABEL[current.kind])}</span>
            {due(current.dueAt) ? <span>· {t("due {0}", due(current.dueAt) ?? "")}</span> : null}
          </div>
          <h3 className="text-lg leading-snug font-semibold text-pretty">{current.title}</h3>
          <p className="text-muted-foreground text-sm text-pretty">{current.reason}</p>
          {current.impact.length > 0 ? (
            <p className="font-mono text-xs tabular-nums">{current.impact.join(" · ")}</p>
          ) : null}
          <div className="flex flex-wrap items-center gap-1.5 pt-1">
            <Button
              size="sm"
              onClick={() => approve(current)}
              disabled={pending}
            >
              {ai ? current.primary.label : current.manualLabel}
              <Kbd className="border-current/30 bg-transparent text-current">{formatShortcut("↵")}</Kbd>
            </Button>
            {current.shipmentId ? (
              <Button size="sm" variant="outline" onClick={() => review(current)}>
                {t("Review")}
              </Button>
            ) : null}
            <Button size="sm" variant="ghost" onClick={() => later(current)} disabled={pending}>
              {t("Later")}
              <Kbd>{formatAltShortcut("L")}</Kbd>
            </Button>
          </div>
        </article>
      ) : (
        <div className="flex flex-col gap-1 py-2">
          <span className="text-success inline-flex items-center gap-1.5 text-sm font-medium">
            <CheckIcon className="size-3.5" />
            {t("All caught up")}
          </span>
          <span className="text-muted-foreground text-sm">
            {t("{0} handled this shift", (data?.handledThisShift ?? 0).toLocaleString())}
          </span>
        </div>
      )}

      {upNext.length > 0 ? (
        <div className="flex flex-col">
          <span className="text-muted-foreground pb-1 text-xs font-medium">{t("Up next")}</span>
          {upNext.map((item) => (
            <button
              key={item.key}
              type="button"
              onClick={() => review(item)}
              className="ui-focus-ring hover:bg-surface-hover -mx-2 flex h-7 items-center gap-2 rounded-md px-2 text-left text-sm"
            >
              <span aria-hidden className={cn("size-1.5 shrink-0 rounded-full", TONE_DOT[item.tone])} />
              <span className={cn("min-w-0 flex-1 truncate", item.deferred && "text-muted-foreground")}>{item.title}</span>
              {due(item.dueAt) ? (
                <span className="text-muted-foreground font-mono text-xs tabular-nums">{due(item.dueAt)}</span>
              ) : null}
            </button>
          ))}
        </div>
      ) : null}
    </section>
  );
}
