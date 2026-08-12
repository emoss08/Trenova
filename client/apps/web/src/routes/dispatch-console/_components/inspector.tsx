import { RateConfirmationActions } from "@/components/carrier-assignment/rate-confirmation-actions";
import { PermissionGate } from "@/components/permission-gate";
import { isTypingTarget } from "@/lib/dom";
import type {
  DispatchBoardDriver,
  DispatchBoardMove,
  DispatchCandidate,
  DispatchDriverMoveMatch,
} from "@/lib/graphql/dispatch-console";
import { dispatchConsoleQueries } from "@/lib/queries/dispatch-console";
import { useDispatchConsoleStore } from "@/stores/dispatch-console-store";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatClockDurationMs, formatUnixDateTime } from "@trenova/shared/lib/date";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery } from "@tanstack/react-query";
import { Building2Icon, SendIcon } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { formatMiles, verdictMeta } from "./dispatch-vocabulary";
import { FindingList } from "./finding-list";
import { ScoreBreakdown } from "./score-breakdown";
import { TenderHistory } from "./tender-history";
import { TenderLivePanel } from "./tender-live-panel";
import type { DispatchActions } from "./use-dispatch-actions";

const CANDIDATE_STALE_MS = 30_000;
const MAX_HOTKEY_RANK = 9;

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
  const missingTractor = !candidate.tractorId;

  return (
    <div
      className={cn(
        "flex flex-col gap-1.5 rounded-md border p-2",
        candidate.blocked ? "bg-muted/30" : "bg-card",
      )}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="flex min-w-0 items-center gap-1.5">
          {rank <= MAX_HOTKEY_RANK && !candidate.blocked && !missingTractor && (
            <Kbd className="size-4 min-w-4 shrink-0 text-[9px]">{rank}</Kbd>
          )}
          <span className="truncate text-xs font-medium">{candidate.workerName}</span>
        </div>
        <Badge variant={verdict.variant} className="h-4 shrink-0 rounded px-1 text-[9px]">
          {verdict.label}
        </Badge>
      </div>

      <div className="flex flex-wrap items-center gap-x-2 gap-y-0.5 text-[10px] text-muted-foreground">
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

      {expanded && (
        <ScoreBreakdown score={candidate.score} factors={candidate.factors} className="mt-1" />
      )}

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
          disabled={candidate.blocked || missingTractor || isAssigning}
          title={
            missingTractor && !candidate.blocked
              ? "This driver has no tractor assigned; assign one before dispatching."
              : undefined
          }
          onClick={() => onAssign(candidate)}
        >
          {candidate.blocked ? "Ineligible" : missingTractor ? "No tractor" : "Assign"}
        </Button>
      </div>
    </div>
  );
}

function CarrierCoverageCard({ move }: { move: DispatchBoardMove }) {
  const openCarrierAssign = useDispatchConsoleStore.use.openCarrierAssign();
  const openCarrierCancel = useDispatchConsoleStore.use.openCarrierCancel();

  return (
    <div className="flex flex-col gap-1.5 border-b bg-muted/30 px-2.5 py-2">
      <div className="flex items-center gap-1.5">
        <Building2Icon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
        <span className="truncate text-xs font-medium">{move.assignedCarrierName}</span>
        <Badge variant="active" className="h-4 shrink-0 rounded px-1 text-[9px]">
          Carrier
        </Badge>
      </div>
      {move.carrierTotalCost != null && (
        <span className="text-[10px] text-muted-foreground tabular-nums">
          Total cost {formatCurrency(move.carrierTotalCost)}
        </span>
      )}
      {move.carrierAssignmentId && (
        <RateConfirmationActions
          moveId={move.moveId}
          carrierAssignmentId={move.carrierAssignmentId}
        />
      )}
      <div className="flex flex-wrap items-center gap-1.5">
        <Button
          size="sm"
          variant="outline"
          className="h-6 px-2 text-[10px]"
          title="Broker this move to a different carrier — the current assignment is replaced"
          onClick={() => openCarrierAssign(move)}
        >
          Replace carrier
        </Button>
        <Button
          size="sm"
          variant="outline"
          className="h-6 px-2 text-[10px]"
          onClick={() => openCarrierCancel(move)}
        >
          Cancel carrier assignment
        </Button>
      </div>
    </div>
  );
}

function TenderCoverageCard({
  move,
  actions,
}: {
  move: DispatchBoardMove;
  actions: DispatchActions;
}) {
  const openCarrierAssign = useDispatchConsoleStore.use.openCarrierAssign();

  const { data: liveTender, isLoading } = useQuery({
    ...dispatchConsoleQueries.liveTender(move.moveId),
  });

  // The board summary can lag the live query: a tender that just resolved still
  // flags the move while liveTender comes back null. Rendering the frame anyway
  // would leave an empty bordered strip in the inspector.
  if (!isLoading && !liveTender) {
    return null;
  }

  return (
    <div className="flex max-h-72 flex-col gap-1.5 overflow-y-auto border-b bg-muted/30 px-2.5 py-2">
      {isLoading ? (
        <Skeleton className="h-20 rounded-md" />
      ) : liveTender ? (
        <TenderLivePanel
          tender={liveTender}
          isTendering={actions.isTendering}
          onCancelTender={actions.cancelTender}
          onRecordResponse={actions.recordOfferResponse}
          onAssignManually={() => openCarrierAssign(move)}
        />
      ) : null}
    </div>
  );
}

function MoveInspector({
  move,
  onAssign,
  isAssigning,
  hotkeysEnabled,
  actions,
}: {
  move: DispatchBoardMove;
  onAssign: (candidate: DispatchCandidate) => void;
  isAssigning: boolean;
  hotkeysEnabled: boolean;
  actions: DispatchActions;
}) {
  const [includeBlocked, setIncludeBlocked] = useState(false);
  const openCarrierAssign = useDispatchConsoleStore.use.openCarrierAssign();
  const openTender = useDispatchConsoleStore.use.openTender();
  const isCarrierCovered = move.coverageType === "carrier";
  // A carrier-covered move cannot take a driver on top; the carrier assignment has to be
  // canceled first, so driver assignment is held off rather than allowed to fail.
  const assignLocked = isAssigning || isCarrierCovered;

  const { data, isLoading } = useQuery({
    ...dispatchConsoleQueries.moveCandidates({ moveId: move.moveId, includeBlocked }),
    staleTime: CANDIDATE_STALE_MS,
  });

  const candidates = useMemo(() => data ?? [], [data]);

  // The rank shown next to each candidate doubles as its keyboard shortcut: pressing the
  // digit assigns without touching the mouse.
  useEffect(() => {
    if (!hotkeysEnabled || assignLocked) return;

    function onKeyDown(event: KeyboardEvent) {
      if (event.metaKey || event.ctrlKey || event.altKey) return;
      if (isTypingTarget(event.target)) return;
      const rank = Number.parseInt(event.key, 10);
      if (Number.isNaN(rank) || rank < 1 || rank > MAX_HOTKEY_RANK) return;
      const candidate = candidates[rank - 1];
      if (!candidate || candidate.blocked || !candidate.tractorId) return;
      event.preventDefault();
      onAssign(candidate);
    }

    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [candidates, hotkeysEnabled, assignLocked, onAssign]);

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="flex flex-col gap-0.5 border-b px-2.5 py-2">
        <span className="font-mono text-xs font-semibold">{move.proNumber}</span>
        <span className="text-[11px] text-muted-foreground">
          {move.originCity}, {move.originState} → {move.destinationCity}, {move.destinationState}
        </span>
        <span className="text-[10px] text-muted-foreground">
          Pickup{" "}
          {move.originWindowStart > 0 ? formatUnixDateTime(move.originWindowStart) : "unscheduled"}
        </span>
      </div>

      {isCarrierCovered && <CarrierCoverageCard move={move} />}

      {move.liveTender && <TenderCoverageCard move={move} actions={actions} />}

      <div className="flex items-center justify-between gap-2 border-b px-2.5 py-1.5">
        <span className="text-[10px] tracking-wide text-muted-foreground uppercase">
          {candidates.length} candidates
        </span>
        <div className="flex items-center gap-1.5">
          {!move.isCovered && (
            <PermissionGate resource={Resource.ShipmentMove} operation={Operation.Assign}>
              <Button
                size="sm"
                variant="outline"
                className="h-6 px-2 text-[10px]"
                disabled={isAssigning}
                onClick={() => openCarrierAssign(move)}
              >
                <Building2Icon className="size-3" aria-hidden />
                Assign to carrier
              </Button>
            </PermissionGate>
          )}
          {!move.isCovered && (
            <PermissionGate resource={Resource.Tender} operation={Operation.Create}>
              <Button
                size="sm"
                variant="outline"
                className="h-6 px-2 text-[10px]"
                disabled={isAssigning}
                onClick={() => openTender(move)}
              >
                <SendIcon className="size-3" aria-hidden />
                Tender to carriers
              </Button>
            </PermissionGate>
          )}
          <Button
            size="sm"
            variant="outline"
            className="h-6 px-2 text-[10px]"
            onClick={() => setIncludeBlocked((value) => !value)}
          >
            {includeBlocked ? "Hide ineligible" : "Show ineligible"}
          </Button>
        </div>
      </div>

      <ScrollArea
        className="min-h-0 flex-1"
        viewportClassName="min-h-0"
        maskVariant="card"
        maskHeight={18}
      >
        <div className="flex flex-col gap-1.5 p-2">
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
                  isAssigning={assignLocked}
                />
              ))}
          {!isLoading && candidates.length === 0 && (
            <p className="px-1 py-6 text-center text-xs text-muted-foreground">
              No eligible driver for this move.
              {!includeBlocked ? " Show ineligible drivers to see why." : ""}
            </p>
          )}

          <TenderHistory shipmentId={move.shipmentId} className="mt-2" />
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
      className="flex flex-col gap-1 rounded-md border bg-card p-2 text-left transition-colors hover:border-brand/40"
    >
      <div className="flex items-start justify-between gap-2">
        <span className="truncate font-mono text-xs font-semibold">{match.move.proNumber}</span>
        <Badge
          variant={verdict.variant}
          className="h-4 shrink-0 rounded px-1 text-[9px] tabular-nums"
        >
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
    ...dispatchConsoleQueries.driverMoves({ workerId: driver.workerId, limit: 20 }),
    staleTime: CANDIDATE_STALE_MS,
  });

  const matches = data ?? [];

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="flex flex-col gap-0.5 border-b px-2.5 py-2">
        <span className="text-xs font-semibold">
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

      <div className="flex flex-col gap-2 p-2.5">
        <FindingList findings={driver.findings} />

        {driver.commitments.length > 0 && (
          <div className="flex flex-col gap-1">
            <span className="text-[10px] tracking-wide text-muted-foreground uppercase">
              Committed
            </span>
            {driver.commitments.map((commitment) => (
              <div
                key={commitment.moveId}
                className="flex items-center justify-between gap-2 rounded border bg-muted/30 px-2 py-1"
              >
                <span className="truncate font-mono text-[10px]">{commitment.proNumber}</span>
                <span className="shrink-0 text-[10px] text-muted-foreground">
                  to {commitment.destinationCity}, {commitment.destinationState}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>

      <span className="border-b px-2.5 pb-1.5 text-[10px] tracking-wide text-muted-foreground uppercase">
        Best fit ({matches.length})
      </span>

      <ScrollArea
        className="min-h-0 flex-1"
        viewportClassName="min-h-0"
        maskVariant="card"
        maskHeight={18}
      >
        <div className="flex flex-col gap-1.5 p-2">
          {isLoading
            ? Array.from({ length: 4 }, (_, index) => (
                <Skeleton key={index} className="h-16 rounded-md" />
              ))
            : matches.map((match) => (
                <DriverMatchRow key={match.move.moveId} match={match} onSelectMove={onSelectMove} />
              ))}
          {!isLoading && matches.length === 0 && (
            <p className="px-1 py-6 text-center text-xs text-muted-foreground">
              No open moves suit this driver right now.
            </p>
          )}
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
  hotkeysEnabled,
  actions,
}: {
  selectedMove: DispatchBoardMove | null;
  selectedDriver: DispatchBoardDriver | null;
  onAssign: (candidate: DispatchCandidate) => void;
  onSelectMove: (moveId: string) => void;
  isAssigning: boolean;
  hotkeysEnabled: boolean;
  actions: DispatchActions;
}) {
  return (
    <section className="flex min-h-0 flex-col overflow-hidden rounded-lg border bg-card">
      <header className="flex items-center justify-between border-b px-2.5 py-1.5">
        <h2 className="text-[10.5px] font-semibold tracking-wide text-muted-foreground uppercase">
          {selectedMove ? "Rank drivers" : selectedDriver ? "Find work" : "Inspector"}
        </h2>
      </header>

      {selectedMove ? (
        <MoveInspector
          move={selectedMove}
          onAssign={onAssign}
          isAssigning={isAssigning}
          hotkeysEnabled={hotkeysEnabled}
          actions={actions}
        />
      ) : selectedDriver ? (
        <DriverInspector driver={selectedDriver} onSelectMove={onSelectMove} />
      ) : (
        <div className="flex flex-1 flex-col items-center justify-center gap-1 p-6 text-center">
          <p className="text-xs font-medium">Nothing selected</p>
          <p className="text-[11px] text-muted-foreground">
            Select a move to rank drivers for it, or a driver to find them work.
          </p>
        </div>
      )}
    </section>
  );
}
