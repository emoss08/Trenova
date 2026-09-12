import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDate } from "@trenova/shared/lib/date";
import {
  acknowledgeMyDisciplinaryAction,
  fetchMyDisciplinaryActions,
  fetchMyRecognitions,
  fetchMySafetyScorecard,
  type PortalDisciplinaryAction,
} from "@trenova/shared/lib/graphql/driver-portal";
import { safetyRatingMeta, scoreRingValue, summariseInspections } from "@trenova/shared/lib/safety";
import { cn } from "@trenova/shared/lib/utils";
import {
  DISCIPLINARY_LEVEL_LABELS,
  RECOGNITION_KIND_LABELS,
  type DisciplinaryLevel,
  type RecognitionKind,
  type SafetyRating,
} from "@trenova/shared/types/worker-safety";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AwardIcon, CheckIcon, ShieldCheckIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

export const DASH_SAFETY_KEY = "dash-safety";
export const DASH_RECOGNITIONS_KEY = "dash-recognitions";
export const DASH_DISCIPLINE_KEY = "dash-discipline";

export function SafetyCard() {
  const t = useT();

  const scorecard = useQuery({
    queryKey: [DASH_SAFETY_KEY],
    queryFn: ({ signal }) => fetchMySafetyScorecard({ signal }),
  });
  const recognitions = useQuery({
    queryKey: [DASH_RECOGNITIONS_KEY],
    queryFn: ({ signal }) => fetchMyRecognitions({ signal }),
  });
  const actions = useQuery({
    queryKey: [DASH_DISCIPLINE_KEY],
    queryFn: ({ signal }) => fetchMyDisciplinaryActions({ signal }),
  });

  if (scorecard.isPending) {
    return <Skeleton className="h-44 w-full rounded-2xl" />;
  }
  if (!scorecard.data) {
    return null;
  }

  const card = scorecard.data;
  const meta = safetyRatingMeta(card.rating as SafetyRating);
  const percent = Math.round(scoreRingValue(card.score) * 100);

  return (
    <div className="rounded-2xl border border-border bg-card p-4">
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <ShieldCheckIcon className="size-4 text-muted-foreground" />
          <h2 className="text-sm font-semibold">{t("Safety")}</h2>
        </div>
        <Badge variant={meta.badgeVariant}>{t(meta.label)}</Badge>
      </div>

      <div className="mt-3 flex items-center gap-3">
        <span className={cn("text-3xl font-semibold tabular-nums", meta.textClass)}>
          {card.score}
        </span>
        <div className="min-w-0 flex-1">
          <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
            <div
              className={cn(
                "h-full rounded-full",
                meta.ringTone === "success"
                  ? "bg-green-500"
                  : meta.ringTone === "warning"
                    ? "bg-amber-500"
                    : "bg-red-500",
              )}
              style={{ width: `${percent}%` }}
            />
          </div>
          <p className="mt-1 text-xs text-muted-foreground">
            {t("{0} point{1} on your record", card.activePoints, card.activePoints === 1 ? "" : "s")}
          </p>
        </div>
      </div>

      <dl className="mt-3 grid grid-cols-2 gap-2 text-xs">
        <Fact label={t("Inspections")} value={summariseInspections(card)} />
        <Fact
          label={t("Last event")}
          value={
            card.daysSinceLastEvent == null
              ? "Nothing on record"
              : `${card.daysSinceLastEvent} days ago`
          }
        />
      </dl>

      {recognitions.data && recognitions.data.length > 0 ? (
        <ul className="mt-3 flex flex-col gap-2 border-t border-border pt-3">
          {recognitions.data.slice(0, 3).map((recognition) => (
            <li key={recognition.id} className="flex items-start gap-2">
              <AwardIcon className="mt-0.5 size-4 shrink-0 text-amber-500" />
              <div className="min-w-0">
                <p className="text-sm font-medium">{t(recognition.title)}</p>
                {recognition.message ? (
                  <p className="text-xs text-muted-foreground">{recognition.message}</p>
                ) : null}
                <p className="text-[11px] text-muted-foreground">
                  {RECOGNITION_KIND_LABELS[recognition.kind as RecognitionKind] ?? recognition.kind}
                  {" · "}
                  {formatUnixDate(recognition.occurredAt)}
                </p>
              </div>
            </li>
          ))}
        </ul>
      ) : null}

      {actions.data && actions.data.length > 0 ? (
        <ul className="mt-3 flex flex-col gap-2 border-t border-border pt-3">
          {actions.data.map((action) => (
            <DisciplineRow key={action.id} action={action} />
          ))}
        </ul>
      ) : null}
    </div>
  );
}

function Fact({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-border bg-muted/30 p-2">
      <dt className="text-[11px] uppercase text-muted-foreground">{label}</dt>
      <dd className="mt-0.5 font-medium">{value}</dd>
    </div>
  );
}

function DisciplineRow({ action }: { action: PortalDisciplinaryAction }) {
  const t = useT();

  const queryClient = useQueryClient();
  const [comment, setComment] = useState("");
  const level = DISCIPLINARY_LEVEL_LABELS[action.level as DisciplinaryLevel] ?? action.level;
  const acknowledged = Boolean(action.acknowledgedAt);

  const acknowledge = useMutation({
    mutationFn: () => acknowledgeMyDisciplinaryAction(action.id, comment || undefined),
    onSuccess: async () => {
      toast.success(t("Thanks — your carrier can see that you have read it."));
      await queryClient.invalidateQueries({ queryKey: [DASH_DISCIPLINE_KEY] });
    },
    onError: (error: Error) => {
      toast.error(error.message || "Could not send that. Try again.");
    },
  });

  return (
    <li data-testid={`dash-discipline-${action.id}`} className="flex flex-col gap-1.5">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="text-sm font-medium">{level}</p>
          <p className="text-xs">{action.reason}</p>
          <p className="text-[11px] text-muted-foreground">
            {formatUnixDate(action.issuedAt)}
            {action.issuedBy?.name ? ` · ${action.issuedBy.name}` : ""}
            {action.suspensionDays ? t("· {0} days", action.suspensionDays) : ""}
          </p>
        </div>
        {acknowledged ? (
          <Badge variant="outline" className="shrink-0 px-1.5 py-0 text-[10px]">
            {t("Read")}
          </Badge>
        ) : null}
      </div>
      {action.details ? <p className="text-xs text-muted-foreground">{action.details}</p> : null}
      {acknowledged ? (
        action.workerComment ? (
          <p className="rounded-md bg-muted/40 px-2 py-1 text-xs">{action.workerComment}</p>
        ) : null
      ) : (
        <div className="flex flex-col gap-1.5">
          <label className="text-[11px] text-muted-foreground" htmlFor={`reply-${action.id}`}>
            {t("Your response")}
          </label>
          <textarea
            id={`reply-${action.id}`}
            className="min-h-16 rounded-md border border-border bg-background p-2 text-xs"
            placeholder={t("Optional — anything you want on the record")}
            value={comment}
            onChange={(event) => setComment(event.target.value)}
          />
          <Button
            size="sm"
            className="h-8 self-start"
            disabled={acknowledge.isPending}
            onClick={() => acknowledge.mutate()}
          >
            <CheckIcon className="size-3.5" />
            {acknowledge.isPending ? t("Sending…") : t("I've read this")}
          </Button>
        </div>
      )}
    </li>
  );
}
