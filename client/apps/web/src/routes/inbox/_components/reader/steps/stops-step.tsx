import { describeApiError } from "@/lib/api-error-message";
import type { InboundMessageDetail, TriageAction, TriageStopsDecision } from "@/lib/graphql/inbox";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixTime } from "@trenova/shared/lib/date";
import { cn, formatCurrencyWhole } from "@trenova/shared/lib/utils";
import { useState } from "react";
import { useTriageActions } from "../../use-triage-actions";
import { useReaderPanel, useStepSubmit } from "../reader-panel";
import { stopRows, stopsReading } from "../stops-reading";
import { StepBody, StepError } from "./step-parts";

const FORMAT = { time: (seconds: number) => formatUnixTime(seconds) };

/**
 * Checking a detention dispute against the stop times: the window, when the
 * truck arrived and by whose clock, check-in and departure, what that means
 * for the charge, and the two answers. Either answer has Nova draft the
 * matching reply, which opens in the composer.
 */
export function StopsStep({
  message,
  action,
  replyStepId,
}: {
  message: InboundMessageDetail;
  action: TriageAction;
  /** The reply step the decision's draft lands in. */
  replyStepId: string | null;
}) {
  const t = useT();
  const actions = useTriageActions();
  const panel = useReaderPanel();
  const query = useQuery(queries.inbox.stopTimes(message.id));
  const [pending, setPending] = useState<TriageStopsDecision | null>(null);
  const [error, setError] = useState<string | null>(null);
  const stops = query.data ?? null;

  const decide = (decision: TriageStopsDecision) => {
    if (pending !== null) {
      return;
    }
    setPending(decision);
    setError(null);
    actions
      .decideStops(message.id, decision)
      .then(
        () => {
          if (replyStepId !== null) {
            panel.dispatch({ type: "open", id: replyStepId });
          }
        },
        (failure: unknown) => setError(describeApiError(failure)),
      )
      .finally(() => setPending(null));
  };
  useStepSubmit(action.id, action.done || stops === null || pending !== null ? null : () => decide("Waive"));

  if (query.isLoading) {
    return (
      <StepBody>
        <Skeleton className="h-24 w-full" />
      </StepBody>
    );
  }
  if (stops === null) {
    return (
      <StepBody>
        <p className="text-foreground-subtle text-sm">
          {query.isError
            ? t("The stop times could not be loaded.")
            : t("There are no stop times on record for this shipment.")}
        </p>
      </StepBody>
    );
  }

  const amount =
    stops.billedDetentionAmount === null ? Number.NaN : Number(stops.billedDetentionAmount);

  return (
    <StepBody>
      <dl className="flex flex-col">
        {stopRows(t, stops, FORMAT).map((row) => (
          <div
            key={row.key}
            className={cn(
              "flex justify-between gap-2 py-1 text-sm",
              row.late ? "text-danger" : "text-foreground-subtle",
            )}
          >
            <dt>{row.label}</dt>
            <dd className={cn("font-mono text-xs font-medium tabular-nums", !row.late && "text-foreground")}>
              {row.value}
            </dd>
          </div>
        ))}
      </dl>
      <p className="text-foreground-muted flex gap-1.75 text-sm leading-normal">
        <AssistMark aria-hidden className="text-accent-violet mt-0.75 size-3 shrink-0" />
        <span>{stopsReading(t, stops, message.matchedCustomer?.name ?? null, FORMAT)}</span>
      </p>
      {error !== null && <StepError>{error}</StepError>}
      {!action.done && (
        <div className="flex flex-col gap-1.5">
          <Button
            size="sm"
            isLoading={pending === "Waive"}
            loadingText={t("Waiving…")}
            disabled={pending !== null}
            onClick={() => decide("Waive")}
          >
            {Number.isNaN(amount) ? t("Waive it") : t("Waive {0}", formatCurrencyWhole(amount))}
          </Button>
          <Button
            size="sm"
            variant="outline"
            isLoading={pending === "StandBy"}
            loadingText={t("Drafting…")}
            disabled={pending !== null}
            onClick={() => decide("StandBy")}
          >
            {t("Stand by it, send the log")}
          </Button>
        </div>
      )}
    </StepBody>
  );
}
