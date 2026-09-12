import { useT } from "@trenova/shared/i18n/use-t";
import { TrainingHealthBadge } from "@trenova/shared/components/training-health-badge";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDate } from "@trenova/shared/lib/date";
import {
  acknowledgeMyTraining,
  fetchMyTraining,
  startMyTraining,
  type PortalTraining,
} from "@trenova/shared/lib/graphql/driver-portal";
import {
  describeTrainingTiming,
  sortTrainingWorstFirst,
  trainingHealthMeta,
} from "@trenova/shared/lib/training";
import { cn } from "@trenova/shared/lib/utils";
import {
  TRAINING_DELIVERY_LABELS,
  type TrainingDelivery,
} from "@trenova/shared/types/worker-training";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckIcon, ClockIcon, ExternalLinkIcon, GraduationCapIcon } from "lucide-react";
import { useMemo } from "react";
import { toast } from "sonner";

export const DASH_TRAINING_KEY = "dash-training";

function isOpen(item: PortalTraining): boolean {
  return item.status === "Assigned" || item.status === "InProgress";
}

export function TrainingCard() {
  const t = useT();

  const training = useQuery({
    queryKey: [DASH_TRAINING_KEY],
    queryFn: ({ signal }) => fetchMyTraining({ signal }),
  });

  const items = useMemo(() => sortTrainingWorstFirst(training.data ?? []), [training.data]);
  const attention = items.filter((item) => trainingHealthMeta(item.health).blocks).length;

  if (training.isPending) {
    return <Skeleton className="h-40 w-full rounded-2xl" />;
  }
  if (!training.data) {
    return null;
  }

  return (
    <div className="rounded-2xl border border-border bg-card p-4">
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <GraduationCapIcon className="size-4 text-muted-foreground" />
          <h2 className="text-sm font-semibold">{t("Training")}</h2>
        </div>
        {attention > 0 ? (
          <Badge variant="warning">
            {t("{0} need{1} attention", attention, attention === 1 ? "s" : "")}
          </Badge>
        ) : (
          <Badge variant="active">{t("All current")}</Badge>
        )}
      </div>

      {items.length === 0 ? (
        <p className="mt-3 text-xs text-muted-foreground">{t("Nothing has been assigned to you yet.")}</p>
      ) : (
        <ul className="mt-3 divide-y divide-border border-t border-border">
          {items.map((item) => (
            <TrainingRow key={item.courseId} item={item} />
          ))}
        </ul>
      )}
      <p className="mt-3 text-xs text-muted-foreground">
        {t("Open a course, take it, then confirm here. Scored courses are closed by your carrier once they enter your result.")}
      </p>
    </div>
  );
}

function TrainingRow({ item }: { item: PortalTraining }) {
  const t = useT();

  const queryClient = useQueryClient();
  const meta = trainingHealthMeta(item.health);
  const open = isOpen(item);
  const acknowledged = Boolean(item.acknowledgedAt);
  const awaitingResult = open && acknowledged && item.scored;
  const canAcknowledge = open && !acknowledged && Boolean(item.id) && item.requiresAcknowledgement;

  const start = useMutation({
    mutationFn: () => {
      if (!item.id) throw new Error("Nothing to start");
      return startMyTraining(item.id);
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [DASH_TRAINING_KEY] }),
  });

  const acknowledge = useMutation({
    mutationFn: () => {
      if (!item.id) throw new Error("Nothing to confirm");
      return acknowledgeMyTraining(item.id);
    },
    onSuccess: async (saved) => {
      toast.success(
        saved.status === "Completed"
          ? `${item.name} is done — nice work.`
          : `Thanks — your carrier will enter your ${item.name} result.`,
      );
      await queryClient.invalidateQueries({ queryKey: [DASH_TRAINING_KEY] });
    },
    onError: (error: Error) => {
      toast.error(error.message || "Could not confirm. Try again.");
    },
  });

  return (
    <li data-testid={`dash-training-${item.courseId}`} className="flex flex-col gap-1.5 py-2.5">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="truncate text-sm font-medium">
            {item.name}
            {item.required ? (
              <span className="ml-1 text-[10px] font-normal uppercase text-muted-foreground">
                {t("Required")}
              </span>
            ) : null}
          </p>
          <p className={cn("text-xs", meta.textClass)}>
            {describeTrainingTiming({
              health: item.health,
              daysUntilDue: item.daysUntilDue,
              daysUntilExpiry: item.daysUntilExpiry,
            })}
            {open && item.dueAt ? (
              <span className="text-muted-foreground"> {t("· by {0}", formatUnixDate(item.dueAt))}</span>
            ) : null}
            {!open && item.expiresAt ? (
              <span className="text-muted-foreground">
                {t("· until {0}", formatUnixDate(item.expiresAt))}
              </span>
            ) : null}
          </p>
        </div>
        <TrainingHealthBadge health={item.health} />
      </div>

      <div className="flex flex-wrap items-center justify-between gap-2 text-xs">
        <div className="flex items-center gap-2 text-muted-foreground">
          <span>
            {TRAINING_DELIVERY_LABELS[item.delivery as TrainingDelivery] ?? item.delivery}
          </span>
          {item.durationMinutes > 0 ? (
            <span className="flex items-center gap-1">
              <ClockIcon className="size-3" />
              {t("{0} min", item.durationMinutes)}
            </span>
          ) : null}
          {awaitingResult ? (
            <span className="text-amber-600 dark:text-amber-400">{t("Waiting for your result")}</span>
          ) : null}
          {item.score ? (
            <span className="tabular-nums">{t("Score {0}%", Number(item.score).toFixed(0))}</span>
          ) : null}
        </div>
        {open || item.health === "ExpiringSoon" ? (
          <div className="flex items-center gap-2">
            {item.contentUrl && item.id ? (
              <a
                href={item.contentUrl}
                target="_blank"
                rel="noreferrer"
                className="inline-flex h-8 items-center gap-1 rounded-md border border-border px-2.5 text-xs font-medium hover:bg-muted"
                aria-label={`Open course ${item.name}`}
                onClick={() => {
                  if (open && item.status === "Assigned") start.mutate();
                }}
              >
                <ExternalLinkIcon className="size-3.5" />
                {t("Open course")}
              </a>
            ) : null}
            {canAcknowledge ? (
              <Button
                size="sm"
                className="h-8"
                aria-label={`I've completed ${item.name}`}
                disabled={acknowledge.isPending}
                onClick={() => acknowledge.mutate()}
              >
                <CheckIcon className="size-3.5" />
                {acknowledge.isPending ? t("Confirming…") : t("I've completed this")}
              </Button>
            ) : null}
          </div>
        ) : null}
      </div>
    </li>
  );
}
