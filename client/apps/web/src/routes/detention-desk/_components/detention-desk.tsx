import { queries } from "@/lib/queries";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { sortDeskEntries, summarizeDesk } from "@trenova/shared/lib/detention";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { TimerIcon } from "lucide-react";
import { AnimatePresence, m } from "motion/react";
import { useEffect, useMemo, useState } from "react";
import { DetentionDeskCard } from "./detention-desk-card";
import { OccurrenceDetailSheet } from "./occurrence-detail-sheet";

/** Refresh cadence for the board and for the local ticking clocks. */
const DESK_REFETCH_MS = 30_000;
const TICK_MS = 1_000;

function useNowSeconds() {
  const [now, setNow] = useState(() => Math.floor(Date.now() / 1000));

  useEffect(() => {
    const id = setInterval(() => setNow(Math.floor(Date.now() / 1000)), TICK_MS);
    return () => clearInterval(id);
  }, []);

  return now;
}

type StatProps = {
  label: string;
  value: string;
  hint?: string;
  emphasis?: "danger" | "warning" | "normal";
};

function Stat({ label, value, hint, emphasis = "normal" }: StatProps) {
  return (
    <div className="bg-card rounded-lg border p-3">
      <p className="text-muted-foreground text-[11px] font-medium tracking-wide uppercase">
        {label}
      </p>
      <p
        className={cn(
          "mt-1 text-2xl font-semibold tabular-nums",
          emphasis === "danger" && "text-red-600 dark:text-red-400",
          emphasis === "warning" && "text-amber-600 dark:text-amber-400",
        )}
      >
        {value}
      </p>
      {hint && <p className="text-muted-foreground mt-0.5 text-xs">{hint}</p>}
    </div>
  );
}

export function DetentionDesk() {
  const nowSeconds = useNowSeconds();
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const { data, isLoading, isError } = useQuery({
    ...queries.detention.desk(),
    refetchInterval: DESK_REFETCH_MS,
  });

  const entries = useMemo(() => sortDeskEntries(data ?? []), [data]);
  const summary = useMemo(() => summarizeDesk(entries), [entries]);

  if (isLoading) {
    return (
      <div className="flex flex-col gap-4">
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-[92px] w-full" />
          ))}
        </div>
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-44 w-full" />
          ))}
        </div>
      </div>
    );
  }

  if (isError) {
    return (
      <div className="flex flex-col items-center gap-2 rounded-lg border border-dashed py-16 text-center">
        <TimerIcon className="text-muted-foreground size-5" />
        <p className="text-sm font-medium">Unable to load the detention desk</p>
        <p className="text-muted-foreground max-w-sm text-xs">
          Retry in a moment. Accruing detention is still being tracked.
        </p>
      </div>
    );
  }

  if (entries.length === 0) {
    return (
      <div className="flex flex-col items-center gap-2 rounded-lg border border-dashed py-16 text-center">
        <TimerIcon className="text-muted-foreground size-5" />
        <p className="text-sm font-medium">No drivers are on a dock right now</p>
        <p className="text-muted-foreground max-w-sm text-xs">
          Stops appear here the moment an arrival is recorded, with the free-time clock and
          notice deadline already running.
        </p>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <Stat
          label="On a dock"
          value={String(summary.total)}
          hint={`${summary.accruing} past free time`}
        />
        <Stat
          label="Notices due"
          value={String(summary.noticesDue)}
          emphasis={summary.noticesDue > 0 ? "warning" : "normal"}
          hint="Send before the deadline or the claim is lost"
        />
        <Stat
          label="Collectable now"
          value={formatCurrency(summary.amountAtRisk)}
          hint="Accrued and still defensible"
        />
        <Stat
          label="Already lost"
          value={formatCurrency(summary.amountLost)}
          emphasis={summary.amountLost > 0 ? "danger" : "normal"}
          hint={`${summary.lost} stop(s) past the notice deadline`}
        />
      </div>

      <AnimatePresence>
        {summary.noticesDue > 0 && (
          <m.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: "auto" }}
            exit={{ opacity: 0, height: 0 }}
            className="overflow-hidden"
          >
            <div className="flex items-center gap-2 rounded-md border border-amber-500/40 bg-amber-500/10 px-3 py-2">
              <Badge className="border-none bg-amber-500/20 text-amber-700 dark:text-amber-400">
                Act now
              </Badge>
              <p className="text-sm">
                {summary.noticesDue} detention{" "}
                {summary.noticesDue === 1 ? "notice" : "notices"} must go out before the
                contractual deadline.
              </p>
            </div>
          </m.div>
        )}
      </AnimatePresence>

      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        <AnimatePresence initial={false}>
          {entries.map((entry) => (
            <m.div
              key={entry.occurrence.id}
              initial={{ opacity: 0, scale: 0.97 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.97 }}
              transition={{ duration: 0.18 }}
            >
              <DetentionDeskCard
                entry={entry}
                nowSeconds={nowSeconds}
                onOpen={setSelectedId}
              />
            </m.div>
          ))}
        </AnimatePresence>
      </div>

      <OccurrenceDetailSheet
        occurrenceId={selectedId}
        onOpenChange={(open) => {
          if (!open) {
            setSelectedId(null);
          }
        }}
      />
    </div>
  );
}
