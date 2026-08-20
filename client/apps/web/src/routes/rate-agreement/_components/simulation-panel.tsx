import { apiService } from "@/services/api";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
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
import type { RateAgreement, RateSimulation } from "@trenova/shared/types/rate";
import { CircleAlertIcon, LoaderCircleIcon, PlayIcon } from "lucide-react";
import { useCallback, useState } from "react";
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

  const [days, setDays] = useState(DEFAULT_WINDOW_DAYS);

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
      toast.success("Simulation queued");
      await queryClient.invalidateQueries({
        queryKey: ["rate-simulations", rateAgreementId],
      });
    },
    onError: () => toast.error("Could not start the simulation"),
  });

  const run = useCallback(() => startRun(), [startRun]);

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

  const progress = runProgress(simulation);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-end gap-3">
        <div className="flex flex-col gap-1">
          <span className="text-muted-foreground text-xs">Replay the last</span>
          <div className="flex items-center gap-2">
            <Input
              type="number"
              className="h-8 w-24"
              value={days}
              onChange={(event) => setDays(Number(event.target.value) || DEFAULT_WINDOW_DAYS)}
            />
            <span className="text-muted-foreground text-xs">days of shipments</span>
          </div>
        </div>
        <Button type="button" size="sm" disabled={isPending} onClick={run}>
          {isPending ? (
            <LoaderCircleIcon className="mr-1 size-3.5 animate-spin" />
          ) : (
            <PlayIcon className="mr-1 size-3.5" />
          )}
          Run simulation
        </Button>
      </div>

      <p className="text-muted-foreground text-xs">
        Every shipment is re-rated against its own facts — the weight it had, the lane it ran, the
        day it shipped — so this is what would have been invoiced. Nothing it produces touches a
        shipment.
      </p>

      {simulation && (
        <div className="rounded-md border p-3">
          <div className="mb-2 flex items-center gap-2">
            <Badge variant={simulation.status === "Failed" ? "warning" : "secondary"}>
              {simulation.status}
            </Badge>
            <span className="text-sm font-medium">{simulation.name}</span>
          </div>

          <p className="text-sm">{summaryHeadline(simulation)}</p>

          {progress !== null && !isTerminal(simulation) && (
            <p className="text-muted-foreground mt-1 text-xs">
              {Math.round(progress * 100)}% of the window replayed
            </p>
          )}

          {isTerminal(simulation) && measuredAnything(simulation) && simulation.summary && (
            <dl className="mt-3 grid grid-cols-2 gap-3 text-xs sm:grid-cols-4">
              <SummaryStat label="Shipments" value={String(simulation.summary.evaluatedCount)} />
              <SummaryStat label="Changed" value={String(simulation.summary.changedCount)} />
              <SummaryStat
                label="Revenue delta"
                value={`${simulation.summary.totalDelta} (${simulation.summary.totalDeltaPct}%)`}
              />
              <SummaryStat
                label="Largest single move"
                value={`${simulation.summary.maxIncrease} / ${simulation.summary.maxDecrease}`}
              />
            </dl>
          )}
        </div>
      )}

      {problems.length > 0 && (
        <div className="flex flex-col gap-2">
          <h4 className="text-sm font-medium">Lanes that did nothing</h4>
          <p className="text-muted-foreground text-xs">
            These do not show up in the revenue total, and they are usually why a tariff prices
            differently from how it was written.
          </p>
          {problems.map((row) => (
            <Alert key={row.ruleId}>
              <CircleAlertIcon className="size-4" />
              <AlertDescription>
                <span className="font-medium">{row.label || displayLaneKey(row.laneKey)}</span>{" "}
                <span className="text-muted-foreground">— {ruleOutcomeLabel(row.outcome)}</span>
                <p className="text-muted-foreground mt-0.5 text-xs">{ruleCoverageNote(row)}</p>
              </AlertDescription>
            </Alert>
          ))}
        </div>
      )}

      {results && results.length > 0 && (
        <div className="flex flex-col gap-2">
          <h4 className="text-sm font-medium">Shipments this would have moved</h4>
          <div className="overflow-x-auto rounded-md border">
            <table className="w-full border-collapse text-sm">
              <thead>
                <tr className="bg-muted/50 text-muted-foreground text-xs">
                  <th className="border-b px-3 py-2 text-left font-medium">Shipment</th>
                  <th className="border-b px-3 py-2 text-left font-medium">Lane</th>
                  <th className="border-b px-3 py-2 text-right font-medium">Billed</th>
                  <th className="border-b px-3 py-2 text-right font-medium">Would charge</th>
                  <th className="border-b px-3 py-2 text-right font-medium">Change</th>
                </tr>
              </thead>
              <tbody>
                {results.map((row) => (
                  <tr key={row.shipmentId}>
                    <td className="border-b px-3 py-2 font-mono text-xs">
                      {row.proNumber || row.shipmentId}
                    </td>
                    <td className="text-muted-foreground border-b px-3 py-2 font-mono text-xs">
                      {row.laneKey ? displayLaneKey(row.laneKey) : "—"}
                    </td>
                    <td className="border-b px-3 py-2 text-right font-mono text-xs">
                      {row.beforeAmount}
                    </td>
                    <td className="border-b px-3 py-2 text-right font-mono text-xs">
                      {row.afterAmount}
                    </td>
                    <td
                      className={`border-b px-3 py-2 text-right font-mono text-xs ${
                        row.delta > 0 ? "text-warning" : "text-foreground"
                      }`}
                    >
                      {row.delta} ({row.deltaPercent}%)
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}

function SummaryStat({ label, value }: { readonly label: string; readonly value: string }) {
  return (
    <div>
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="font-mono text-sm">{value}</dd>
    </div>
  );
}
