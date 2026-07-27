import type {
  DispatchBoardDriver,
  DispatchBoardMove,
  DispatchCandidate,
  DispatchDriverMoveMatch,
} from "@/lib/graphql/dispatch-console";
import { queries } from "@/lib/queries";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatClockDurationMs, formatUnixDateTime } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { formatMiles, verdictMeta } from "./dispatch-vocabulary";
import { FindingList } from "./finding-list";
import { ScoreBreakdown } from "./score-breakdown";

const CANDIDATE_STALE_MS = 30_000;

function CandidateRow({
  candidate,
  rank,
  onAssign,
  isAssigning,
}: {
  candidate: DispatchCandidate;
  rank: number;
  onAssign: (candidate: DispatchCandidate) => void;
  isAssigning: boolean;
}) {
  const [expanded, setExpanded] = useState(false);
  const verdict = verdictMeta(candidate.verdict);

  return (
    <div
      className={cn(
        "flex flex-col gap-1.5 rounded-md border p-2",
        candidate.blocked ? "border-border bg-muted/30" : "border-border bg-card",
      )}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="flex min-w-0 items-center gap-1.5">
          {/* The rank doubles as the keyboard shortcut for assigning this candidate. */}
          {rank <= 9 && !candidate.blocked ? (
            <kbd className="flex size-4 shrink-0 items-center justify-center rounded border border-border bg-muted font-mono text-[9px]">
              {rank}
            </kbd>
          ) : null}
          <span className="truncate text-xs font-medium">{candidate.workerName}</span>
        </div>
        <Badge variant={verdict.variant} className="h-4 shrink-0 rounded px-1 text-[9px]">
          {verdict.label}
        </Badge>
      </div>

      <div className="flex flex-wrap items-center gap-2 text-[10px] text-muted-foreground">
        <span className="tabular-nums">Score {candidate.score}</span>
        <span>· {formatMiles(candidate.deadheadMiles)} empty</span>
        <span>· {formatClockDurationMs(candidate.driveRemainingMs)} drive left</span>
        {candidate.minutesOfSlack < 0 ? (
          <span className="text-red-600 dark:text-red-400">
            · {Math.abs(candidate.minutesOfSlack)}m late
          </span>
        ) : (
          <span>· {candidate.minutesOfSlack}m margin</span>
        )}
      </div>

      <FindingList findings={candidate.findings} limit={expanded ? undefined : 2} />

      {expanded ? (
        <ScoreBreakdown score={candidate.score} factors={candidate.factors} className="mt-1" />
      ) : null}

      <div className="flex items-center gap-1.5">
        <Button
          size="sm"
          variant="outline"
          className="h-6 px-2 text-[10px]"
          onClick={() => setExpanded((value) => !value)}
        >
          {expanded ? "Hide detail" : "Why this score"}
        </Button>
        <Button
          size="sm"
          className="h-6 px-2 text-[10px]"
          disabled={candidate.blocked || isAssigning}
          onClick={() => onAssign(candidate)}
        >
          {candidate.blocked ? "Ineligible" : "Assign"}
        </Button>
      </div>
    </div>
  );
}

function MoveInspector({
  move,
  onAssign,
  isAssigning,
}: {
  move: DispatchBoardMove;
  onAssign: (candidate: DispatchCandidate) => void;
  isAssigning: boolean;
}) {
  const [includeBlocked, setIncludeBlocked] = useState(false);

  const { data, isLoading } = useQuery({
    ...queries.dispatchConsole.moveCandidates({ moveId: move.moveId, includeBlocked }),
    staleTime: CANDIDATE_STALE_MS,
  });

  const candidates = data ?? [];

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-2">
      <div className="flex flex-col gap-0.5">
        <span className="font-mono text-xs font-medium">{move.proNumber}</span>
        <span className="text-[11px] text-muted-foreground">
          {move.originCity}, {move.originState} → {move.destinationCity}, {move.destinationState}
        </span>
        <span className="text-[10px] text-muted-foreground">
          Pickup {formatUnixDateTime(move.originWindowStart)}
        </span>
      </div>

      <div className="flex items-center justify-between gap-2">
        <span className="text-[10px] uppercase tracking-wide text-muted-foreground">
          {candidates.length} candidates
        </span>
        <Button
          size="sm"
          variant="outline"
          className="h-6 px-2 text-[10px]"
          onClick={() => setIncludeBlocked((value) => !value)}
        >
          {includeBlocked ? "Hide ineligible" : "Show ineligible"}
        </Button>
      </div>

      <ScrollArea className="min-h-0 flex-1">
        <div className="flex flex-col gap-1.5 pr-2">
          {isLoading
            ? Array.from({ length: 5 }, (_, index) => (
                <Skeleton key={index} className="h-24 rounded-md" />
              ))
            : candidates.map((candidate, index) => (
                <CandidateRow
                  key={candidate.workerId}
                  candidate={candidate}
                  rank={index + 1}
                  onAssign={onAssign}
                  isAssigning={isAssigning}
                />
              ))}
          {!isLoading && candidates.length === 0 ? (
            <p className="px-1 py-6 text-center text-xs text-muted-foreground">
              No eligible driver for this move.
              {!includeBlocked ? " Show ineligible drivers to see why." : ""}
            </p>
          ) : null}
        </div>
      </ScrollArea>
    </div>
  );
}

function DriverMatchRow({
  match,
  onSelectMove,
}: {
  match: DispatchDriverMoveMatch;
  onSelectMove: (moveId: string) => void;
}) {
  const verdict = verdictMeta(match.score.verdict);

  return (
    <button
      type="button"
      onClick={() => onSelectMove(match.move.moveId)}
      className="flex flex-col gap-1 rounded-md border border-border bg-card p-2 text-left transition-colors hover:border-brand/40"
    >
      <div className="flex items-start justify-between gap-2">
        <span className="truncate font-mono text-xs font-medium">{match.move.proNumber}</span>
        <Badge variant={verdict.variant} className="h-4 shrink-0 rounded px-1 text-[9px]">
          {match.score.score}
        </Badge>
      </div>
      <span className="truncate text-[11px]">
        {match.move.originCity}, {match.move.originState} → {match.move.destinationCity},{" "}
        {match.move.destinationState}
      </span>
      <span className="text-[10px] text-muted-foreground">
        {formatUnixDateTime(match.move.originWindowStart)} ·{" "}
        {formatMiles(match.score.deadheadMiles)} empty
      </span>
    </button>
  );
}

function DriverInspector({
  driver,
  onSelectMove,
}: {
  driver: DispatchBoardDriver;
  onSelectMove: (moveId: string) => void;
}) {
  const { data, isLoading } = useQuery({
    ...queries.dispatchConsole.driverMoves({ workerId: driver.workerId, limit: 20 }),
    staleTime: CANDIDATE_STALE_MS,
  });

  const matches = data ?? [];

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-2">
      <div className="flex flex-col gap-0.5">
        <span className="text-xs font-medium">
          {driver.firstName} {driver.lastName}
        </span>
        <span className="text-[11px] text-muted-foreground">
          {driver.tractorCode ? `${driver.tractorCode} · ` : ""}
          {driver.formattedLocation || `${driver.city}, ${driver.stateAbbreviation}`}
        </span>
        <span className="text-[10px] text-muted-foreground">
          Available {formatUnixDateTime(driver.projectedTimeAvailable)}
        </span>
      </div>

      <FindingList findings={driver.findings} />

      {driver.commitments.length > 0 ? (
        <div className="flex flex-col gap-1">
          <span className="text-[10px] uppercase tracking-wide text-muted-foreground">
            Committed
          </span>
          {driver.commitments.map((commitment) => (
            <div
              key={commitment.moveId}
              className="flex items-center justify-between gap-2 rounded border border-border bg-muted/30 px-2 py-1"
            >
              <span className="truncate font-mono text-[10px]">{commitment.proNumber}</span>
              <span className="shrink-0 text-[10px] text-muted-foreground">
                to {commitment.destinationCity}, {commitment.destinationState}
              </span>
            </div>
          ))}
        </div>
      ) : null}

      <span className="text-[10px] uppercase tracking-wide text-muted-foreground">
        Best fit ({matches.length})
      </span>

      <ScrollArea className="min-h-0 flex-1">
        <div className="flex flex-col gap-1.5 pr-2">
          {isLoading
            ? Array.from({ length: 4 }, (_, index) => (
                <Skeleton key={index} className="h-16 rounded-md" />
              ))
            : matches.map((match) => (
                <DriverMatchRow
                  key={match.move.moveId}
                  match={match}
                  onSelectMove={onSelectMove}
                />
              ))}
          {!isLoading && matches.length === 0 ? (
            <p className="px-1 py-6 text-center text-xs text-muted-foreground">
              No open moves suit this driver right now.
            </p>
          ) : null}
        </div>
      </ScrollArea>
    </div>
  );
}

/**
 * The inspector matches in both directions, because dispatchers work in both: covering a
 * load, and finding work for an idle truck.
 */
export function Inspector({
  selectedMove,
  selectedDriver,
  onAssign,
  onSelectMove,
  isAssigning,
}: {
  selectedMove: DispatchBoardMove | null;
  selectedDriver: DispatchBoardDriver | null;
  onAssign: (candidate: DispatchCandidate) => void;
  onSelectMove: (moveId: string) => void;
  isAssigning: boolean;
}) {
  return (
    <section className="flex h-full min-h-0 flex-col gap-2 rounded-md border border-border bg-background p-2">
      <header className="flex items-baseline justify-between gap-2">
        <h2 className="text-xs font-semibold uppercase tracking-wide">Inspector</h2>
      </header>

      {selectedMove ? (
        <MoveInspector move={selectedMove} onAssign={onAssign} isAssigning={isAssigning} />
      ) : selectedDriver ? (
        <DriverInspector driver={selectedDriver} onSelectMove={onSelectMove} />
      ) : (
        <p className="px-1 py-8 text-center text-xs text-muted-foreground">
          Select a move to rank drivers, or a driver to find them work.
        </p>
      )}
    </section>
  );
}
