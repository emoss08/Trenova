import { HoverGlideGroup } from "@/components/hover-glide";
import type { DispatchBoardMove, DispatchCandidate } from "@/lib/graphql/dispatch-console";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { ChevronDownIcon } from "@trenova/shared/components/icons";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostBox, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatClockDurationMs, formatMinutesSpan } from "@trenova/shared/lib/date";
import { cn, formatCentsWhole } from "@trenova/shared/lib/utils";
import { useId, useState } from "react";
import { ArmedPreflight } from "./armed-preflight";
import { CostBreakdown } from "./cost-breakdown";
import { preferenceFloorOf } from "./cost-vocabulary";
import { formatMiles } from "./dispatch-vocabulary";
import { FindingList, FindingSummary, findingKey } from "./finding-list";

export const MAX_HOTKEY_RANK = 9;
const INITIAL_VISIBLE = 6;

/** A driver can be picked from here only with a tractor and nothing blocking them. */
export function isPickable(candidate: DispatchCandidate): boolean {
  return !candidate.blocked && Boolean(candidate.tractorId);
}

/**
 * The findings every ranked driver shares, such as a pickup window that has already
 * closed: they are about the move, not the driver, so the list says them once.
 */
export function sharedFindings(candidates: readonly DispatchCandidate[]) {
  if (candidates.length < 2) return [];
  const [first, ...others] = candidates;
  return first.findings.filter((finding) =>
    others.every((other) =>
      other.findings.some((candidate) => findingKey(candidate) === findingKey(finding)),
    ),
  );
}

function initials(name: string): string {
  const parts = name.trim().split(/\s+/);
  const letters = `${parts[0]?.charAt(0) ?? ""}${parts.length > 1 ? (parts.at(-1)?.charAt(0) ?? "") : ""}`;
  return letters.toUpperCase() || "?";
}

function CandidateSketch() {
  return (
    <div className="border-border/70 bg-card rounded-md border text-left">
      {[0, 1, 2, 3].map((row) => (
        <div
          key={row}
          className="border-border/60 grid grid-cols-[0.875rem_1.375rem_minmax(0,1fr)_1.5rem] items-center gap-2.5 border-b border-dashed px-3 py-2.5 last:border-0"
        >
          <GhostBox />
          <span className="bg-muted block size-5 rounded-full" />
          <span className="flex flex-col gap-1.5">
            <GhostLine className={cn("h-2", ["w-4/5", "w-3/5", "w-full", "w-1/2"][row])} />
            <GhostLine className={["w-full", "w-1/2", "w-4/5", "w-3/5"][row]} />
          </span>
          <GhostLine className="h-2 w-5 justify-self-end" />
        </div>
      ))}
    </div>
  );
}

/** The facts a dispatcher reads first about a driver for a move, leaving out what is not known. */
function candidateFacts(candidate: DispatchCandidate, t: ReturnType<typeof useT>): string {
  return [
    t("{0} empty", formatMiles(candidate.deadheadMiles)),
    candidate.hosKnown
      ? t("{0} drive left", formatClockDurationMs(candidate.driveRemainingMs))
      : t("Hours unknown"),
    candidate.slackKnown
      ? candidate.minutesOfSlack < 0
        ? t("{0} late", formatMinutesSpan(candidate.minutesOfSlack))
        : t("{0} margin", formatMinutesSpan(candidate.minutesOfSlack))
      : null,
  ]
    .filter(Boolean)
    .join(" · ");
}

function CandidateRow({
  candidate,
  rank,
  proposed,
  plan,
  isArmed,
  canAssign,
  onArm,
  sharedKeys,
  preferenceFloorCents,
}: {
  candidate: DispatchCandidate;
  preferenceFloorCents: number;
  rank: number;
  proposed: boolean;
  /** Where the last plan stands on this driver: its pick, or what choosing them adds. */
  plan: CandidatePlanStanding | null;
  isArmed: boolean;
  canAssign: boolean;
  onArm: (candidate: DispatchCandidate) => void;
  /** Findings every ranked driver has, said once above the list rather than on each row. */
  sharedKeys: ReadonlySet<string>;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const findings = candidate.findings.filter((finding) => !sharedKeys.has(findingKey(finding)));

  const pickable = isPickable(candidate);
  const hotkey = canAssign && pickable && rank <= MAX_HOTKEY_RANK;

  return (
    <div
      data-glide-target
      className={cn(
        "relative mx-1 flex flex-col gap-1.5 rounded-md p-2 text-xs",
        isArmed && "bg-surface-hover",
        !pickable && "text-muted-foreground",
      )}
    >
      <div className="grid grid-cols-[1rem_1.5rem_minmax(0,1fr)_auto] items-center gap-2.5">
        {hotkey ? <Kbd className="size-4 min-w-4 text-3xs">{rank}</Kbd> : <span aria-hidden />}
        <span
          aria-hidden
          className="bg-muted text-muted-foreground flex size-6 items-center justify-center rounded-full text-3xs font-medium"
        >
          {initials(candidate.workerName)}
        </span>
        <div className="min-w-0">
          <div className="flex min-w-0 items-center gap-1.5 font-medium">
            <span className="truncate">{candidate.workerName}</span>
            {proposed ? <NovaBadge size="xs" label={t("Nova's pick")} /> : null}
            {plan === null ? null : plan.pick ? (
              <span className="text-muted-foreground inline-flex shrink-0 items-center gap-0.5 text-2xs font-normal">
                <AssistMark className="size-3" aria-hidden />
                {t("Plan's pick")}
              </span>
            ) : (
              <span className="text-muted-foreground shrink-0 text-2xs font-normal tabular-nums">
                {t("+{0} to the plan", formatCentsWhole(plan.extraCents))}
              </span>
            )}
          </div>
          <div className="text-muted-foreground truncate text-2xs">
            {candidateFacts(candidate, t)}
          </div>
        </div>
        <div className="flex items-center gap-2.5">
          <span className="text-right font-mono text-xs tabular-nums" title={t("Cost to cover")}>
            {formatCentsWhole(candidate.costCents - preferenceFloorCents)}
          </span>
          {canAssign ? (
            <Button
              size="xs"
              variant="outline"
              disabled={!pickable}
              title={
                !candidate.tractorId && !candidate.blocked
                  ? t("This driver has no tractor assigned; assign one before dispatching.")
                  : undefined
              }
              onClick={() => onArm(candidate)}
            >
              {candidate.blocked
                ? t("Ineligible")
                : !candidate.tractorId
                  ? t("No tractor")
                  : t("Assign")}
            </Button>
          ) : null}
        </div>
      </div>
      {findings.length > 0 ? (
        <div className="pl-[3.25rem]">
          {open ? <FindingList findings={findings} /> : <FindingSummary findings={findings} />}
        </div>
      ) : null}
      <div className="pl-[3.25rem]">
        <Button
          variant="link"
          size="xxs"
          aria-expanded={open}
          onClick={() => setOpen((value) => !value)}
        >
          {open ? t("Hide detail") : t("Why this cost")}
        </Button>
        {open ? (
          <CostBreakdown
            candidate={candidate}
            preferenceFloorCents={preferenceFloorCents}
            className="mt-1"
          />
        ) : null}
      </div>
    </div>
  );
}

type CandidatePlanStanding = { pick: true } | { pick: false; extraCents: number };

/**
 * What the move's last plan says about each driver: the one it chose, and for the others it
 * weighed, the exact extra cost to the whole plan of choosing them instead.
 */
function planStandings(move: DispatchBoardMove): Map<string, CandidatePlanStanding> {
  const standings = new Map<string, CandidatePlanStanding>();
  const price = move.isCovered ? null : move.planPrice;
  if (!price) return standings;
  for (const alternative of price.alternatives) {
    standings.set(alternative.workerId, { pick: false, extraCents: alternative.extraCents });
  }
  if (price.workerId) standings.set(price.workerId, { pick: true });
  return standings;
}

export type CandidatesRead = {
  candidates: readonly DispatchCandidate[];
  isPending: boolean;
  isError: boolean;
  error: unknown;
  refetch: () => void;
};

/**
 * The drivers ranked for a move. The rank is the digit that opens a driver's pre-flight,
 * so covering a load never needs the mouse; whatever opens, nothing commits until the
 * pre-flight is confirmed.
 */
export function CandidateList({
  move,
  read,
  includeBlocked,
  onToggleBlocked,
  armedWorkerId,
  proposedWorkerId,
  canAssign,
  isAssigning,
  onArm,
  onDisarm,
  onConfirm,
  onTender,
  open,
  onOpenChange,
}: {
  move: DispatchBoardMove;
  read: CandidatesRead;
  includeBlocked: boolean;
  onToggleBlocked: () => void;
  armedWorkerId: string | null;
  proposedWorkerId: string | null;
  canAssign: boolean;
  isAssigning: boolean;
  onArm: (candidate: DispatchCandidate) => void;
  onDisarm: () => void;
  onConfirm: (
    candidate: DispatchCandidate,
    params: { tractorId: string; trailerId?: string },
  ) => Promise<void>;
  onTender: (() => void) | null;
  /** Whether the drivers are shown; the heading's chevron folds them away. */
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const bodyId = useId();
  const [showAll, setShowAll] = useState(false);

  const { candidates } = read;
  const armedIndex = candidates.findIndex((candidate) => candidate.workerId === armedWorkerId);
  const visibleCount = Math.max(INITIAL_VISIBLE, armedIndex + 1);
  // The first drivers always show; the rest stay mounted under them and open and close by
  // height, so "more drivers" eases open rather than jumping, and folds back the same way.
  const first = candidates.slice(0, visibleCount);
  const rest = candidates.slice(visibleCount);
  const restId = useId();
  const shared = sharedFindings(candidates);
  const sharedKeys = new Set(shared.map(findingKey));
  const standings = planStandings(move);

  const renderCandidate = (candidate: DispatchCandidate, index: number) => (
    <div key={candidate.workerId}>
      <CandidateRow
        candidate={candidate}
        rank={index + 1}
        proposed={candidate.workerId === proposedWorkerId}
        plan={standings.get(candidate.workerId) ?? null}
        isArmed={candidate.workerId === armedWorkerId}
        canAssign={canAssign}
        onArm={onArm}
        sharedKeys={sharedKeys}
        preferenceFloorCents={preferenceFloorOf(move)}
      />
      {candidate.workerId === armedWorkerId ? (
        <ArmedPreflight
          key={`${move.moveId}:${candidate.workerId}`}
          move={move}
          candidate={candidate}
          isAssigning={isAssigning}
          onCancel={onDisarm}
          onConfirm={(params) => onConfirm(candidate, params)}
        />
      ) : null}
    </div>
  );
  const eligible = candidates.filter(isPickable).length;

  return (
    <section
      className="bg-card border-border overflow-hidden rounded-md border"
      aria-label={t("Drivers for this move")}
    >
      <header
        className={cn(
          "flex h-8.5 items-center gap-1.5 border-b pr-3 pl-1.5 text-xs font-semibold transition-[border-color] duration-280 ease-settle motion-reduce:transition-none",
          open ? "border-border" : "border-transparent",
        )}
      >
        <Button
          variant="ghost"
          size="icon-xs"
          aria-expanded={open}
          aria-controls={bodyId}
          aria-label={open ? t("Hide the drivers") : t("Show the drivers")}
          title={open ? t("Hide the drivers") : t("Show the drivers")}
          onClick={() => onOpenChange(!open)}
        >
          <ChevronDownIcon
            className={cn(
              "text-muted-foreground size-3.5 transition-transform duration-280 ease-settle motion-reduce:transition-none",
              !open && "-rotate-90",
            )}
            aria-hidden
          />
        </Button>
        {t("Drivers for this move")}
        <span className="text-muted-foreground ml-auto flex items-center gap-1 text-2xs font-normal">
          {read.isPending || read.isError ? null : (
            <span className="tabular-nums">
              {t("{0, plural, one {# eligible} other {# eligible}}", eligible)}
            </span>
          )}
          {open ? (
            <>
              <span aria-hidden>·</span>
              <Button variant="link" size="xxs" onClick={onToggleBlocked}>
                {includeBlocked ? t("Hide ineligible") : t("Show ineligible")}
              </Button>
            </>
          ) : null}
        </span>
      </header>

      {/* Opens and closes by height, as a row the queue gives up does; closed, it is inert
          so the drivers it hides take no focus. */}
      <div
        id={bodyId}
        inert={!open}
        className={cn(
          "grid transition-[grid-template-rows,opacity] duration-280 ease-settle motion-reduce:transition-none",
          open ? "grid-rows-[1fr] opacity-100" : "grid-rows-[0fr] opacity-0",
        )}
      >
        <div className="min-h-0 overflow-hidden">
          {read.isPending ? (
            <div
              className="flex flex-col gap-1 p-1"
              aria-busy="true"
              aria-label={t("Ranking drivers")}
            >
              {Array.from({ length: 4 }, (_, index) => (
                <div key={index} className="ui-shimmer h-11 rounded-md" />
              ))}
            </div>
          ) : read.isError ? (
            <ErrorState
              error={read.error}
              layout="sheet"
              sketch={<CandidateSketch />}
              title={t("Couldn't rank drivers")}
              description={t("The ranking didn't load. Drivers may still fit this move.")}
              onRetry={read.refetch}
            />
          ) : candidates.length === 0 ? (
            <EmptySheet
              sketch={<CandidateSketch />}
              title={t("No eligible driver")}
              description={
                includeBlocked
                  ? t(
                      "No driver on duty can take this move on the clocks and equipment they have now.",
                    )
                  : t(
                      "Every driver on duty was checked against this move and none can take it now. Show the ineligible ones to see why.",
                    )
              }
              action={
                <div className="flex flex-wrap justify-center gap-2">
                  {!includeBlocked ? (
                    <Button size="sm" variant="outline" onClick={onToggleBlocked}>
                      {t("Show ineligible")}
                    </Button>
                  ) : null}
                  {onTender ? (
                    <Button size="sm" variant="outline" onClick={onTender}>
                      {t("Tender to carriers")}
                    </Button>
                  ) : null}
                </div>
              }
            />
          ) : (
            <>
              {shared.length > 0 ? (
                <div className="border-border flex min-w-0 items-center gap-1.5 border-b px-3 py-1.5">
                  <span className="text-muted-foreground shrink-0 text-2xs">
                    {t("Every driver:")}
                  </span>
                  <FindingSummary findings={shared} />
                </div>
              ) : null}
              <HoverGlideGroup className="py-1">
                {first.map((candidate, index) => renderCandidate(candidate, index))}
                {rest.length > 0 ? (
                  <div
                    id={restId}
                    inert={!showAll}
                    className={cn(
                      "grid transition-[grid-template-rows,opacity] duration-280 ease-settle motion-reduce:transition-none",
                      showAll ? "grid-rows-[1fr] opacity-100" : "grid-rows-[0fr] opacity-0",
                    )}
                  >
                    <div className="min-h-0 overflow-hidden">
                      {rest.map((candidate, index) =>
                        renderCandidate(candidate, first.length + index),
                      )}
                    </div>
                  </div>
                ) : null}
              </HoverGlideGroup>
              {rest.length > 0 ? (
                <div className="border-border border-t px-1.5 py-1.5">
                  <Button
                    variant="ghost"
                    size="xxs"
                    className="text-brand gap-1"
                    aria-expanded={showAll}
                    aria-controls={restId}
                    onClick={() => setShowAll((value) => !value)}
                  >
                    <ChevronDownIcon
                      className={cn(
                        "size-3.5 transition-transform duration-280 ease-settle motion-reduce:transition-none",
                        showAll && "rotate-180",
                      )}
                      aria-hidden
                    />
                    {showAll
                      ? t("Show fewer")
                      : t("{0, plural, one {# more driver} other {# more drivers}}", rest.length)}
                  </Button>
                </div>
              ) : null}
            </>
          )}
        </div>
      </div>
    </section>
  );
}
