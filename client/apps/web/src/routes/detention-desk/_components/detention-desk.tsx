import { queries } from "@/lib/queries";
import { Badge } from "@trenova/shared/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@trenova/shared/components/ui/card";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { sortDeskEntries, summarizeDesk } from "@trenova/shared/lib/detention";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { useEffect, useMemo, useState } from "react";
import { DetentionDeskCard } from "./detention-desk-card";

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
    <Card>
      <CardHeader className="pb-2">
        <CardDescription className="text-xs">{label}</CardDescription>
        <CardTitle
          className={cn(
            "text-2xl tabular-nums",
            emphasis === "danger" && "text-red-600 dark:text-red-400",
            emphasis === "warning" && "text-amber-600 dark:text-amber-400",
          )}
        >
          {value}
        </CardTitle>
      </CardHeader>
      {hint && (
        <CardContent className="pt-0">
          <p className="text-muted-foreground text-xs">{hint}</p>
        </CardContent>
      )}
    </Card>
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
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <Skeleton key={i} className="h-40 w-full" />
        ))}
      </div>
    );
  }

  if (isError) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Unable to load the detention desk</CardTitle>
          <CardDescription>
            Retry in a moment. Accruing detention is still being tracked.
          </CardDescription>
        </CardHeader>
      </Card>
    );
  }

  if (entries.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>No drivers are on a dock right now</CardTitle>
          <CardDescription>
            Stops appear here the moment an arrival is recorded, with the free-time
            clock and notice deadline already running.
          </CardDescription>
        </CardHeader>
      </Card>
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

      {summary.noticesDue > 0 && (
        <div className="flex items-center gap-2 rounded-md border border-amber-500/40 bg-amber-500/10 px-3 py-2">
          <Badge className="border-none bg-amber-500/20 text-amber-700 dark:text-amber-400">
            Act now
          </Badge>
          <p className="text-sm">
            {summary.noticesDue} detention {summary.noticesDue === 1 ? "notice" : "notices"}{" "}
            must go out before the contractual deadline.
          </p>
        </div>
      )}

      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        {entries.map((entry) => (
          <DetentionDeskCard
            key={entry.occurrence.id}
            entry={entry}
            nowSeconds={nowSeconds}
            onOpen={setSelectedId}
          />
        ))}
      </div>

      {selectedId && (
        <p className="text-muted-foreground text-xs">
          Selected occurrence {selectedId}
        </p>
      )}
    </div>
  );
}
