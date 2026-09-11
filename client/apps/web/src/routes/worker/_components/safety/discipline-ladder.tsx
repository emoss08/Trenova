import { useT } from "@trenova/shared/i18n/use-t";
import { InfoPopover } from "@/components/info-popover";
import { RowActionsMenu } from "@/components/row-actions-menu";
import type { DisciplinaryLadder, WorkerDisciplinaryActionRow } from "@/lib/graphql/worker-safety";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { disciplinaryLevelMeta } from "@trenova/shared/lib/safety";
import { cn } from "@trenova/shared/lib/utils";
import {
  DISCIPLINARY_LEVEL_LABELS,
  DISCIPLINARY_STATUS_LABELS,
  disciplinaryLevelSchema,
  type DisciplinaryLevel,
  type DisciplinaryStatus,
} from "@trenova/shared/types/worker-safety";
import { CheckIcon, GavelIcon, TriangleAlertIcon, UndoIcon } from "lucide-react";

type DisciplineLadderProps = {
  ladder: DisciplinaryLadder;
  actions: WorkerDisciplinaryActionRow[];
  canIssue: boolean;
  canRescind: boolean;
  busy: boolean;
  onIssue: () => void;
  onRescind: (action: WorkerDisciplinaryActionRow) => void;
};

const LADDER: DisciplinaryLevel[] = disciplinaryLevelSchema.options;

const STATUS_VARIANT: Record<DisciplinaryStatus, "warning" | "outline" | "inactive"> = {
  Active: "warning",
  Rescinded: "outline",
  Expired: "inactive",
};

/**
 * The ladder as six rungs. Rungs already climbed are filled, the next one is
 * outlined, and the rest wait in the background — the same stepper the
 * employment process uses, read as a warning rather than a journey.
 */
export function DisciplineLadder({
  ladder,
  actions,
  canIssue,
  canRescind,
  busy,
  onIssue,
  onRescind,
}: DisciplineLadderProps) {
  const t = useT();

  const highestRank = ladder.highestLevel
    ? disciplinaryLevelMeta(ladder.highestLevel as DisciplinaryLevel).rank
    : 0;
  const suggestedRank = disciplinaryLevelMeta(ladder.suggestedLevel as DisciplinaryLevel).rank;
  const nextLabel =
    DISCIPLINARY_LEVEL_LABELS[ladder.suggestedLevel as DisciplinaryLevel] ?? ladder.suggestedLevel;

  return (
    <section className="flex flex-col gap-3">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <div className="flex items-center gap-1.5">
            <h4 className="text-muted-foreground text-[11px] font-semibold uppercase">
              {t("Discipline")}
            </h4>
            <InfoPopover title={t("Discipline ladder")}>
              {t("Six rungs from coaching to termination. The next step is one rung above the highest action still active, and drops back as actions expire or are rescinded. It is a suggestion, not a rule: any rung can be issued, and a rung that ends employment is flagged before it is.")}
            </InfoPopover>
          </div>
          <p className="mt-0.5 text-xs">
            <span className="font-medium">{`Next step: ${nextLabel}`}</span>
            {ladder.atFinalStep ? (
              <span className="text-destructive"> {t("— this would end employment.")}</span>
            ) : null}
          </p>
        </div>
        {canIssue ? (
          <Button
            size="sm"
            variant={ladder.atFinalStep ? "destructive" : "outline"}
            disabled={busy}
            onClick={onIssue}
          >
            {ladder.atFinalStep ? (
              <TriangleAlertIcon className="size-3.5" />
            ) : (
              <GavelIcon className="size-3.5" />
            )}
            {t("Issue action")}
          </Button>
        ) : null}
      </div>

      <ol className="flex items-center gap-2 rounded-lg border px-4 py-3" aria-label={t("Ladder")}>
        {LADDER.map((level, index) => {
          const meta = disciplinaryLevelMeta(level);
          const taken = meta.rank <= highestRank;
          const isNext = meta.rank === suggestedRank && !taken;
          return (
            <li key={level} className="flex flex-1 items-center gap-2 last:flex-none">
              <span
                className="flex items-center gap-1.5"
                aria-current={isNext ? "step" : undefined}
              >
                <span
                  className={cn(
                    "inline-flex size-5 items-center justify-center rounded-full border text-[10px] font-medium tabular-nums",
                    taken && "border-primary bg-primary text-primary-foreground",
                    isNext && "border-primary border-dashed",
                    !taken && !isNext && "text-muted-foreground",
                  )}
                  aria-hidden
                >
                  {taken ? <CheckIcon className="size-3" /> : meta.rank}
                </span>
                <span
                  className={cn(
                    "hidden text-xs sm:inline",
                    taken || isNext ? "font-medium" : "text-muted-foreground",
                  )}
                >
                  {DISCIPLINARY_LEVEL_LABELS[level]}
                </span>
              </span>
              {index < LADDER.length - 1 ? (
                <span
                  className={cn("h-px flex-1", taken ? "bg-primary/40" : "bg-border")}
                  aria-hidden
                />
              ) : null}
            </li>
          );
        })}
      </ol>

      {actions.length === 0 ? (
        <p className="text-muted-foreground rounded-lg border border-dashed px-3 py-4 text-center text-xs">
          {t("No disciplinary actions on record")}
        </p>
      ) : (
        <ul className="divide-border divide-y rounded-lg border">
          {actions.map((action) => {
            const meta = disciplinaryLevelMeta(action.level as DisciplinaryLevel);
            const status = action.status as DisciplinaryStatus;
            return (
              <li
                key={action.id}
                data-testid={`discipline-${action.id}`}
                className="flex items-start gap-3 px-3 py-2.5"
              >
                <span
                  className="bg-accent inline-flex size-6 shrink-0 items-center justify-center rounded-md text-[10px] font-medium tabular-nums"
                  aria-hidden
                >
                  {meta.rank}
                </span>
                <div className="min-w-0 flex-1">
                  <p className="flex flex-wrap items-center gap-2 text-sm font-medium">
                    {meta.label}
                    <Badge variant={STATUS_VARIANT[status] ?? "outline"}>
                      {DISCIPLINARY_STATUS_LABELS[status] ?? status}
                    </Badge>
                    {action.acknowledgedAt ? (
                      <span className="text-2xs text-muted-foreground uppercase">{t("Acknowledged")}</span>
                    ) : null}
                  </p>
                  <p className="text-xs">{action.reason}</p>
                  <p className="text-muted-foreground text-xs">
                    {t("Issued {0}{1}{2}{3}{4}", formatUnixDate(action.issuedAt), action.issuedBy?.name ? ` by ${action.issuedBy.name}` : "", action.expiresAt ? ` · rolls off ${formatUnixDate(action.expiresAt)}` : "", action.suspensionDays ? ` · ${action.suspensionDays} days` : "", action.rescindReason ? ` · rescinded: ${action.rescindReason}` : "")}
                  </p>
                  {action.workerComment ? (
                    <p className="text-muted-foreground mt-1 text-xs">“{action.workerComment}”</p>
                  ) : null}
                </div>
                <RowActionsMenu
                  label={`Actions for ${meta.label}`}
                  actions={
                    canRescind && status === "Active"
                      ? [
                          {
                            id: "rescind",
                            label: `Rescind ${meta.label}`,
                            icon: UndoIcon,
                            disabled: busy,
                            destructive: true,
                            onSelect: () => onRescind(action),
                          },
                        ]
                      : []
                  }
                />
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}
