import { usePermission } from "@/hooks/use-permission";
import { isWindowMonths, type WindowMonths } from "@/lib/fleet-safety-console";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { XIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { BasicsCard } from "./basics-card";
import { RankList, TerminalsPanel } from "./fleet-safety-panels";
import { FleetSafetyOverview } from "./fleet-safety-overview";
import { FleetSafetySkeleton } from "./fleet-safety-skeleton";
import { fleetSafetyQuery } from "./queries";
import { TrendCard } from "./trend-card";

const WINDOW_ITEMS = [
  { value: "3", label: "3 months" },
  { value: "6", label: "6 months" },
  { value: "12", label: "12 months" },
  { value: "24", label: "24 months" },
] satisfies { value: WindowMonths; label: string }[];

export default function FleetSafetyConsole() {
  const { allowed: canRead } = usePermission(Resource.WorkerSafetyEvent, Operation.Read);
  const [windowMonths, setWindowMonths] = useState<WindowMonths>("12");
  const [fleetCodeId, setFleetCodeId] = useState("");

  const fleet = useQuery({
    ...fleetSafetyQuery({ windowMonths: Number(windowMonths), fleetCodeId: fleetCodeId || null }),
    enabled: canRead,
  });
  const summary = fleet.data;
  const selectedTerminal = useMemo(
    () => summary?.terminals.find((terminal) => terminal.fleetCodeId === fleetCodeId) ?? null,
    [summary, fleetCodeId],
  );

  // Safety events carry accidents, citations and discipline. Somebody without
  // the grant sees nothing rather than an empty page implying a clean fleet.
  if (!canRead) return null;

  if (fleet.isLoading && !summary) return <FleetSafetySkeleton />;

  if (fleet.isError) {
    return (
      <div className="text-destructive rounded-lg border border-dashed p-4 text-sm">
        The fleet could not be read. {fleet.error.message}
      </div>
    );
  }

  if (!summary) return null;

  return (
    <div className="flex flex-col gap-4">
      <FleetSafetyOverview summary={summary} />

      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <SegmentedControl<WindowMonths>
            items={WINDOW_ITEMS}
            value={windowMonths}
            onValueChange={(value) => setWindowMonths(isWindowMonths(value) ? value : "12")}
            aria-label="Counting window"
          />
          {selectedTerminal ? (
            <Button
              size="xs"
              variant="outline"
              aria-label={`Clear terminal ${selectedTerminal.code}`}
              onClick={() => setFleetCodeId("")}
            >
              <span
                aria-hidden
                className="size-2 rounded-full"
                style={{ backgroundColor: selectedTerminal.color || "var(--muted-foreground)" }}
              />
              {selectedTerminal.code}
              <XIcon className="size-3" />
            </Button>
          ) : (
            <span className="text-muted-foreground text-xs">
              Every terminal · choose one below to narrow the page
            </span>
          )}
        </div>
        <p className="text-muted-foreground text-xs">
          As of {formatUnixDate(summary.asOf)}
          {fleet.isFetching ? " · refreshing" : ""}
        </p>
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <BasicsCard basics={summary.basics} inferred={summary.basicsInferred} />
        <TrendCard
          trend={summary.trend}
          kinds={summary.kinds}
          windowMonths={summary.windowMonths}
        />
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        <TerminalsPanel
          terminals={summary.terminals}
          totalWorkers={summary.workers}
          selected={fleetCodeId}
          onSelect={setFleetCodeId}
        />
        <RankList
          title="Needs attention"
          kind="worst"
          empty="Nobody is carrying points."
          rows={summary.worst}
        />
        <RankList
          title="Best records"
          kind="best"
          empty="No drivers to rank."
          rows={summary.best}
        />
      </div>
    </div>
  );
}
