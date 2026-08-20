import { DeltaValue, StatTile } from "@/components/metric-tiles";
import { apiService } from "@/services/api";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  NumberFieldGroup,
  NumberFieldInput,
  NumberField as NumberFieldRoot,
} from "@trenova/shared/components/ui/number-field";
import { Progress } from "@trenova/shared/components/ui/progress";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import {
  isTerminal,
  measuredAnything,
  problemRules,
  ruleCoverageNote,
  ruleOutcomeLabel,
  runProgress,
  shouldPoll,
  summaryHeadline,
} from "@trenova/shared/lib/simulation";
import { formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import type { RateAgreement, RateSimulation } from "@trenova/shared/types/rate";
import { ArrowRightIcon, FlaskConicalIcon, PlayIcon } from "lucide-react";
import { useState } from "react";
import { useFormContext } from "react-hook-form";
import { toast } from "sonner";
import { useLaneKeyLabels } from "./use-lane-scope-labels";

/** How often a running simulation is asked how it is getting on. */
const POLL_INTERVAL_MS = 5_000;

const DAY_SECONDS = 86_400;

/**
 * A year is the window somebody means by "against last year's freight", and it
 * is long enough that seasonality does not distort the answer.
 */
const DEFAULT_WINDOW_DAYS = 365;

/** Recent runs worth listing; older ones are noise in a side panel. */
const RUN_HISTORY_LIMIT = 5;

type SimulationPanelProps = {
  /** Absent while the agreement is being created — there is nothing to replay. */
  readonly rateAgreementId?: string;
};

/**
 * What this contract would have charged for freight that already moved.
 *
 * The run happens in the background because a year of shipments takes minutes,
 * so this starts one and then watches it. It stops watching the moment the run
 * finishes: polling a finished job forever is how a background task quietly
 * becomes a load on the database.
 */
export function SimulationPanel({ rateAgreementId }: SimulationPanelProps) {
  const { getValues } = useFormContext<RateAgreement>();
  const queryClient = useQueryClient();

  const [watching, setWatching] = useState<string | undefined>();
  const [days, setDays] = useState(DEFAULT_WINDOW_DAYS);

  const { data: history } = useQuery({
    queryKey: ["rate-simulations", rateAgreementId],
    queryFn: () => apiService.rateSimulationService.listForAgreement(rateAgreementId as string),
    enabled: Boolean(rateAgreementId),
  });

  const active = watching ?? history?.[0]?.id;

  const { data: simulation } = useQuery({
    queryKey: ["rate-simulation", active],
    queryFn: () => apiService.rateSimulationService.getById(active as string),
    enabled: Boolean(active),
    refetchInterval: (query) =>
      shouldPoll(query.state.data as RateSimulation | undefined) ? POLL_INTERVAL_MS : false,
  });

  const { mutate: startRun, isPending } = useMutation({
    mutationFn: () => {
      const now = Math.floor(Date.now() / 1000);

      return apiService.rateSimulationService.create({
        rateAgreementId: rateAgreementId as string,
        name: `${getValues("name") || "Agreement"} — last ${days} days`,
        partyType: getValues("partyType"),
        sampleFrom: now - days * DAY_SECONDS,
        sampleTo: now,
      });
    },
    onSuccess: async (created) => {
      setWatching(created.id);
      await queryClient.invalidateQueries({
        queryKey: ["rate-simulations", rateAgreementId],
      });
    },
    onError: () => toast.error("Could not start the simulation"),
  });

  const { data: results } = useQuery({
    queryKey: ["rate-simulation-results", active],
    queryFn: () =>
      apiService.rateSimulationService.listResults(active as string, { changedOnly: true }),
    enabled: Boolean(active) && isTerminal(simulation),
  });

  const problems = problemRules(simulation?.ruleCoverage);
  const displayLaneKey = useLaneKeyLabels([
    ...problems.map((row) => row.laneKey),
    ...(results ?? []).map((row) => row.laneKey),
  ]);

  if (!rateAgreementId) {
    return (
      <p className="text-muted-foreground text-sm">
        Save the agreement first. A simulation replays a contract against shipments that already
        moved, and there has to be a contract to replay.
      </p>
    );
  }

  const recentRuns = (history ?? []).slice(0, RUN_HISTORY_LIMIT);

  return (
    <div className="space-y-4">
      <div className="bg-muted/30 rounded-lg border p-3">
        <div className="mb-3">
          <p className="text-sm font-medium">Replay Historical Shipments</p>
          <p className="text-muted-foreground mt-0.5 text-xs">
            Every shipment is re-rated against its own facts — the weight it had, the lane it ran,
            the day it shipped — so the result is what would have been invoiced. Nothing it produces
            touches a shipment.
          </p>
        </div>

        <div className="flex flex-wrap items-end gap-3">
          <div className="w-32">
            <label className="text-muted-foreground mb-1.5 block text-xs font-medium">
              Window (Days)
            </label>
            <NumberFieldRoot
              value={days}
              onValueChange={(value) =>
                setDays(Math.min(Math.max(value ?? DEFAULT_WINDOW_DAYS, 1), 1095))
              }
              min={1}
              max={1095}
              step={30}
              size="sm"
            >
              <NumberFieldGroup>
                <NumberFieldInput className="text-right" />
              </NumberFieldGroup>
            </NumberFieldRoot>
          </div>
          <Button
            type="button"
            size="sm"
            onClick={() => startRun()}
            isLoading={isPending}
            loadingText="Queueing..."
            className="gap-1.5"
          >
            <PlayIcon className="size-3.5" />
            Run Simulation
          </Button>
        </div>
      </div>

      {recentRuns.length > 1 && (
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          {recentRuns.map((run) => (
            <button
              key={run.id}
              type="button"
              onClick={() => setWatching(run.id)}
              className={cn(
                "rounded-lg border p-2 text-left transition-colors",
                run.id === active
                  ? "border-primary bg-primary/5"
                  : "border-border bg-background hover:bg-muted/50",
              )}
            >
              <div className="flex items-center justify-between gap-2">
                <p className="truncate text-xs font-medium">{run.name}</p>
                <Badge
                  variant={run.status === "Failed" ? "warning" : "secondary"}
                  className="shrink-0"
                >
                  {run.status}
                </Badge>
              </div>
              {run.startedAt ? (
                <p className="text-2xs text-muted-foreground mt-0.5">
                  {formatUnixDateTimeShort(run.startedAt)}
                </p>
              ) : null}
            </button>
          ))}
        </div>
      )}

      {simulation ? (
        <SimulationReading simulation={simulation} />
      ) : (
        <div className="flex flex-col items-center justify-center rounded-lg border border-dashed py-12 text-center">
          <FlaskConicalIcon className="text-muted-foreground mb-3 size-8" />
          <p className="text-sm font-medium">No simulation has run yet</p>
          <p className="text-muted-foreground mt-1 max-w-sm text-xs">
            Run one to see what this contract would have charged for the freight you already moved —
            before it prices a single live shipment.
          </p>
        </div>
      )}

      {problems.length > 0 && (
        <div className="space-y-2">
          <div>
            <p className="text-sm font-medium">Lanes That Did Nothing</p>
            <p className="text-muted-foreground mt-0.5 text-xs">
              These are invisible in the revenue total, and they are usually why a tariff prices
              differently from how it was written.
            </p>
          </div>
          <div className="overflow-hidden rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="text-xs">Lane</TableHead>
                  <TableHead className="text-xs">Outcome</TableHead>
                  <TableHead className="text-xs">What That Means</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {problems.map((row) => (
                  <TableRow key={row.ruleId}>
                    <TableCell>
                      <span className="text-xs font-medium">
                        {row.label || displayLaneKey(row.laneKey)}
                      </span>
                      {row.label ? (
                        <p className="text-2xs text-muted-foreground font-mono">
                          {displayLaneKey(row.laneKey)}
                        </p>
                      ) : null}
                    </TableCell>
                    <TableCell className="text-xs whitespace-nowrap">
                      {ruleOutcomeLabel(row.outcome)}
                    </TableCell>
                    <TableCell className="text-muted-foreground text-xs">
                      {ruleCoverageNote(row)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </div>
      )}

      {results && results.length > 0 && (
        <div className="space-y-2">
          <div>
            <p className="text-sm font-medium">Shipments This Would Have Moved</p>
            <p className="text-muted-foreground mt-0.5 text-xs">
              Largest increases first — the shipment that will produce the phone call is what this
              list is for.
            </p>
          </div>
          <div className="overflow-hidden rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="text-xs">Pro #</TableHead>
                  <TableHead className="text-xs">Lane</TableHead>
                  <TableHead className="text-right text-xs">Billed</TableHead>
                  <TableHead className="text-right text-xs">Would Charge</TableHead>
                  <TableHead className="text-right text-xs">Delta</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {results.map((row) => (
                  <TableRow key={row.shipmentId}>
                    <TableCell className="font-mono text-xs">
                      {row.proNumber || row.shipmentId}
                    </TableCell>
                    <TableCell className="text-muted-foreground font-mono text-xs">
                      {row.laneKey ? displayLaneKey(row.laneKey) : "—"}
                    </TableCell>
                    <TableCell className="text-right font-mono text-xs tabular-nums">
                      {formatCurrency(row.beforeAmount)}
                    </TableCell>
                    <TableCell className="text-right font-mono text-xs tabular-nums">
                      {formatCurrency(row.afterAmount)}
                    </TableCell>
                    <TableCell className="text-right text-xs">
                      <DeltaValue delta={row.delta} deltaPct={row.deltaPercent} />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </div>
      )}
    </div>
  );
}

/** One run's answer, in the order somebody reads it: status, verdict, numbers. */
function SimulationReading({ simulation }: { readonly simulation: RateSimulation }) {
  const progress = runProgress(simulation);
  const finished = isTerminal(simulation);
  const summary = simulation.summary;

  return (
    <div className="space-y-3">
      <div className="bg-muted/30 rounded-lg border p-3">
        <div className="mb-1.5 flex items-center gap-2">
          <Badge variant={simulation.status === "Failed" ? "warning" : "secondary"}>
            {simulation.status}
          </Badge>
          <span className="truncate text-sm font-medium">{simulation.name}</span>
        </div>

        <p className="text-sm">{summaryHeadline(simulation)}</p>

        {!finished && progress !== null && (
          <div className="mt-2 flex items-center gap-2">
            <Progress value={progress * 100} className="h-1.5 flex-1" />
            <span className="text-muted-foreground text-xs tabular-nums">
              {Math.round(progress * 100)}%
            </span>
          </div>
        )}
      </div>

      {finished && measuredAnything(simulation) && summary && (
        <>
          <div className="grid grid-cols-2 gap-2 sm:grid-cols-5">
            <StatTile label="Shipments" value={String(summary.evaluatedCount)} />
            <StatTile label="Changed" value={String(summary.changedCount)} />
            <StatTile
              label="Increased"
              value={String(summary.increasedCount)}
              tone="text-emerald-600 dark:text-emerald-400"
            />
            <StatTile
              label="Decreased"
              value={String(summary.decreasedCount)}
              tone="text-red-600 dark:text-red-400"
            />
            <StatTile
              label="Errors"
              value={String(summary.errorCount)}
              tone={summary.errorCount > 0 ? "text-destructive" : undefined}
            />
          </div>

          <div className="bg-muted/30 flex flex-wrap items-center gap-2 rounded-lg border px-3 py-2 text-sm">
            <span className="text-muted-foreground">Total</span>
            <span className="font-mono font-medium tabular-nums">
              {formatCurrency(summary.beforeTotal)}
            </span>
            <ArrowRightIcon className="text-muted-foreground size-3.5" />
            <span className="font-mono font-medium tabular-nums">
              {formatCurrency(summary.afterTotal)}
            </span>
            <DeltaValue delta={summary.totalDelta} deltaPct={summary.totalDeltaPct} />
            <span className="text-muted-foreground ml-auto text-xs">
              Max increase {formatCurrency(summary.maxIncrease)} · Max decrease{" "}
              {formatCurrency(summary.maxDecrease)}
            </span>
          </div>
        </>
      )}
    </div>
  );
}
