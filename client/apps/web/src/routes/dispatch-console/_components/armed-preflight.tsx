import type { DispatchBoardMove, DispatchCandidate } from "@/lib/graphql/dispatch-console";
import { dispatchConsoleQueries } from "@/lib/queries/dispatch-console";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { AlertTriangleIcon, CheckCircleIcon, XOctagonIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatClockDurationMs, formatMinutesSpan, formatUnixTime } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { useEffect, useRef } from "react";
import { formatMiles, hosStrategyMeta, verdictMeta } from "./dispatch-vocabulary";
import { FindingList } from "./finding-list";

function Figure({ value, label, loading }: { value: string; label: string; loading: boolean }) {
  return (
    <div className="border-border min-w-0 border-r px-2.5 py-1.75 last:border-r-0">
      <span
        className={cn(
          "block truncate font-mono text-sm leading-4.5 font-medium tabular-nums",
          loading && "ui-shimmer text-transparent",
        )}
      >
        {value}
      </span>
      <span className="text-muted-foreground block text-2xs">{label}</span>
    </div>
  );
}

/**
 * The pre-flight, opened in place under the driver it is for. A digit or Assign opens it
 * with focus waiting on confirm, so a clean pairing is two keys; confirm stays off until
 * the preview has checked the pairing, and a reassign names who it replaces. Nothing is
 * written until the dispatcher confirms, and it stays open until the write answers.
 */
export function ArmedPreflight({
  move,
  candidate,
  isAssigning,
  onConfirm,
  onCancel,
}: {
  move: DispatchBoardMove;
  candidate: DispatchCandidate;
  isAssigning: boolean;
  onConfirm: (params: { tractorId: string; trailerId?: string }) => Promise<void>;
  onCancel: () => void;
}) {
  const t = useT();
  const confirmRef = useRef<HTMLButtonElement>(null);

  const { data, isPending, isError, error, refetch } = useQuery(
    dispatchConsoleQueries.assignmentPreview({
      moveId: move.moveId,
      workerId: candidate.workerId,
    }),
  );

  const tractorId = data?.tractorId ?? candidate.tractorId;
  const blockedByPolicy = Boolean(data?.requiresOverride);
  const ready = Boolean(data) && Boolean(tractorId) && !blockedByPolicy;
  const reassigning = move.isCovered && move.assignedWorkerName !== "";

  // Focus waits on confirm once there is something to confirm, so Enter commits and Esc
  // (handled by the inspector) backs out.
  useEffect(() => {
    if (ready) confirmRef.current?.focus({ preventScroll: true });
  }, [ready]);

  const score = data?.candidate;
  const verdict = score ? verdictMeta(score.verdict) : null;
  const block = score?.findings.find((finding) => finding.severity === "Block");
  const strategy = hosStrategyMeta(score?.hosStrategy);
  const lead = block
    ? block.message
    : strategy && score?.hosStrategy !== "currentClocks"
      ? score && score.hosRestStartDeadline > 0
        ? t(
            "{0}. Rest must start by {1}.",
            strategy.phrase,
            formatUnixTime(score.hosRestStartDeadline),
          )
        : strategy.phrase
      : score?.verdict === "unknown"
        ? t("No hours-of-service feed for this driver; judged on schedule alone.")
        : t("Legal on current clocks.");
  const slack = score?.minutesOfSlack ?? 0;
  const otherFindings = score?.findings.filter((finding) => finding !== block) ?? [];

  return (
    <div
      className="animate-expand-in border-border bg-card mx-1 mt-0.5 mb-1.5 rounded-md border p-3 text-xs"
      aria-label={t("Pre-flight for {0}", candidate.workerName)}
      role="group"
    >
      <div className="text-sm leading-4.5 font-semibold">
        {reassigning
          ? t("Reassign {0} to {1}", move.proNumber, candidate.workerName)
          : t("Assign {0} to {1}", move.proNumber, candidate.workerName)}
      </div>
      {reassigning ? (
        <div className="text-muted-foreground mt-0.5">
          {move.assignedTractorCode
            ? t("Replaces {0} · {1}", move.assignedWorkerName, move.assignedTractorCode)
            : t("Replaces {0}", move.assignedWorkerName)}
        </div>
      ) : null}

      {isError ? (
        <ErrorState
          className="mt-2"
          error={error}
          layout="compact"
          title={t("Couldn't check this pairing")}
          description={t("Nothing can be assigned until the pairing is checked.")}
          onRetry={() => void refetch()}
        />
      ) : (
        <>
          <div className="mt-2 flex items-start gap-1.5">
            {isPending ? (
              <span className="ui-shimmer rounded-sm text-transparent">
                {t("Checking hours of service")}
              </span>
            ) : (
              <>
                {block ? (
                  <XOctagonIcon
                    className="text-danger-foreground mt-px size-3.5 shrink-0"
                    aria-hidden
                  />
                ) : score?.verdict === "tight" ? (
                  <AlertTriangleIcon
                    className="text-warning-foreground mt-px size-3.5 shrink-0"
                    aria-hidden
                  />
                ) : (
                  <CheckCircleIcon
                    className="text-success-foreground mt-px size-3.5 shrink-0"
                    aria-hidden
                  />
                )}
                <span className="animate-rise">
                  {verdict ? <span className="font-medium">{t(verdict.label)}. </span> : null}
                  {lead}
                </span>
              </>
            )}
          </div>

          <div className="border-border my-2.5 grid grid-cols-3 rounded-md border">
            <Figure
              loading={isPending}
              value={score ? formatMiles(score.deadheadMiles) : "00 mi"}
              label={t("Empty miles")}
            />
            <Figure
              loading={isPending}
              value={score ? formatClockDurationMs(score.driveRemainingMs) : "0:00"}
              label={t("Drive left")}
            />
            <Figure
              loading={isPending}
              value={score ? formatMinutesSpan(slack) : "00m"}
              label={slack < 0 ? t("Late at pickup") : t("Pickup margin")}
            />
          </div>

          {otherFindings.length > 0 ? <FindingList findings={otherFindings} limit={3} /> : null}
          {!isPending && !tractorId ? (
            <p className="text-danger-foreground">
              {t("This driver has no tractor assigned; assign one before dispatching.")}
            </p>
          ) : null}
          {blockedByPolicy ? (
            <p className="text-danger-foreground">
              {t(
                "Your organization enforces this rule as a block, so this pairing can't be assigned.",
              )}
            </p>
          ) : null}
        </>
      )}

      <div className="mt-2.5 flex items-center gap-2">
        <Button
          ref={confirmRef}
          size="sm"
          disabled={!ready || isAssigning}
          isLoading={isAssigning}
          shortcut="↵"
          onClick={() =>
            tractorId
              ? void onConfirm({ tractorId, trailerId: data?.trailerId ?? undefined })
              : undefined
          }
        >
          {reassigning ? t("Reassign driver") : t("Assign driver")}
        </Button>
        <Button
          size="sm"
          variant="outline"
          onClick={onCancel}
          disabled={isAssigning}
          shortcut="Esc"
        >
          {t("Cancel")}
        </Button>
        <span className="text-muted-foreground ml-auto text-2xs">
          {isPending ? t("Loading preview") : t("Nothing is written until you confirm")}
        </span>
      </div>
    </div>
  );
}
