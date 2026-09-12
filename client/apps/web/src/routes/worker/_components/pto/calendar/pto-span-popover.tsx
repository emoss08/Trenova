import { useT } from "@trenova/shared/i18n/use-t";
import { PTOStatusBadge, PTOTypeBadge } from "@trenova/shared/components/status-badge";
import { Avatar, AvatarFallback, AvatarImage } from "@trenova/shared/components/ui/avatar";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { formatRange, formatUnixDateMedium } from "@trenova/shared/lib/date";
import { formatPtoDays, ptoTypeMeta } from "@trenova/shared/lib/pto";
import { cn } from "@trenova/shared/lib/utils";
import type { WorkerPTO } from "@trenova/shared/types/worker";
import { CalendarRangeIcon, ClockIcon, WalletIcon } from "lucide-react";
import { useMemo } from "react";
import { PTOActionsMenu } from "../pto-actions-menu";
import { ptoDaysOf, ptoDecision } from "../pto-columns";
import { ptoWorkerInitials, ptoWorkerName } from "../pto-worker";
import { ptoTiming, spansOnDay, type PTOTimingTone } from "./calendar-layout";

const TIMING_TONE_CLASS: Record<PTOTimingTone, string> = {
  upcoming: "text-foreground",
  active: "text-green-700 dark:text-green-400",
  past: "text-muted-foreground",
};

const TIMING_DOT_CLASS: Record<PTOTimingTone, string> = {
  upcoming: "bg-blue-600",
  active: "bg-green-600",
  past: "bg-muted-foreground/50",
};

export type PTOSpanDetailsProps = {
  pto: WorkerPTO;
  todayUnix: number;
};

/** The body of the popover that opens from a bar on the calendar. */
export function PTOSpanDetails({ pto, todayUnix }: PTOSpanDetailsProps) {
  const t = useT();

  const name = ptoWorkerName(pto);
  const days = ptoDaysOf(pto);
  const timing = ptoTiming(pto, todayUnix);
  const decision = ptoDecision(pto);

  return (
    <div className="flex flex-col gap-2.5" data-testid={`pto-span-details-${pto.id}`}>
      <div className="flex items-start gap-2.5">
        <Avatar className="ring-border size-9 ring-1">
          <AvatarImage src={pto.worker?.profilePicUrl ?? undefined} alt={name} />
          <AvatarFallback className="text-xs font-medium">{ptoWorkerInitials(pto)}</AvatarFallback>
        </Avatar>
        <div className="min-w-0 flex-1 leading-tight">
          <p className="truncate text-sm font-medium">{name}</p>
          <div className="mt-1 flex flex-wrap items-center gap-1">
            <PTOTypeBadge type={pto.type} />
            <PTOStatusBadge status={pto.status} />
          </div>
        </div>
        <PTOActionsMenu pto={pto} />
      </div>

      <dl className="bg-accent/50 divide-border/60 divide-y rounded-lg text-xs">
        <Fact icon={CalendarRangeIcon} label={t("Dates")}>
          <span className="tabular-nums">{formatRange(pto.startDate, pto.endDate)}</span>
          <span className="text-muted-foreground ml-auto tabular-nums">
            {t("{0, plural, one {# day} other {# days}}", days)}
          </span>
        </Fact>
        <Fact icon={ClockIcon} label={t("Timing")}>
          <span className={cn("flex items-center gap-1.5", TIMING_TONE_CLASS[timing.tone])}>
            <span
              className={cn("size-1.5 rounded-full", TIMING_DOT_CLASS[timing.tone])}
              aria-hidden
            />
            {t(timing.label)}
          </span>
        </Fact>
        {pto.balanceAfterDays != null ? (
          <Fact icon={WalletIcon} label={t("Balance after")}>
            <span>{t("Balance after")}</span>
            <span className="ml-auto tabular-nums">{t("{0} days", formatPtoDays(pto.balanceAfterDays))}</span>
          </Fact>
        ) : null}
      </dl>

      {pto.reason ? (
        <p className="border-border text-muted-foreground border-l-2 pl-2 text-xs leading-snug">
          {pto.reason}
        </p>
      ) : null}

      {decision ? (
        <p className="text-muted-foreground text-[11px] leading-tight">
          {decision.verb} <span className="text-foreground">{decision.actor}</span>
          {decision.note ? <> · {decision.note}</> : null}
        </p>
      ) : pto.autoApproved ? (
        <p className="text-muted-foreground text-[11px] leading-tight">{t("Auto-approved by policy")}</p>
      ) : null}
    </div>
  );
}

function Fact({
  icon: Icon,
  label,
  children,
}: {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-center gap-2 px-2 py-1.5">
      <dt className="sr-only">{label}</dt>
      <Icon className="text-muted-foreground size-3.5 shrink-0" aria-hidden />
      <dd className="flex min-w-0 flex-1 items-center gap-1.5">{children}</dd>
    </div>
  );
}

export type PTODayListProps = {
  items: readonly WorkerPTO[];
  dayUnix: number;
};

/** Everyone whose time off covers one day, for the "+N" chip a crowded week folds into. */
export function PTODayList({ items, dayUnix }: PTODayListProps) {
  const t = useT();

  const out = useMemo(() => spansOnDay(items, dayUnix), [items, dayUnix]);

  return (
    <div className="flex flex-col gap-1.5" data-testid="pto-day-list">
      <div className="flex items-baseline justify-between gap-2 px-0.5">
        <p className="text-sm font-medium">{formatUnixDateMedium(dayUnix)}</p>
        <span className="text-muted-foreground text-xs tabular-nums">{t("{0} out", out.length)}</span>
      </div>
      <ScrollArea className="-mx-1 px-1" viewportClassName="max-h-64" maskHeight={16}>
        <ul className="divide-border/60 divide-y">
          {out.map((pto) => {
            const meta = ptoTypeMeta(pto.type);
            return (
              <li key={pto.id} className="flex items-center gap-2 py-1.5">
                <span className={cn("size-2 shrink-0 rounded-full", meta.dotClass)} aria-hidden />
                <div className="min-w-0 flex-1 leading-tight">
                  <p className="truncate text-xs font-medium">{ptoWorkerName(pto)}</p>
                  <p className="text-muted-foreground truncate text-[11px] tabular-nums">
                    {t(meta.label)} · {formatRange(pto.startDate, pto.endDate)}
                    {pto.status === "Requested" ? ` ${t("· awaiting decision")}` : ""}
                  </p>
                </div>
                <PTOActionsMenu pto={pto} className="size-5.5" />
              </li>
            );
          })}
        </ul>
      </ScrollArea>
    </div>
  );
}
