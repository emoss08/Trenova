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
import { GavelIcon, TriangleAlertIcon, UndoIcon } from "lucide-react";

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

export function DisciplineLadder({
  ladder,
  actions,
  canIssue,
  canRescind,
  busy,
  onIssue,
  onRescind,
}: DisciplineLadderProps) {
  const highestRank = ladder.highestLevel
    ? disciplinaryLevelMeta(ladder.highestLevel as DisciplinaryLevel).rank
    : 0;
  const suggestedRank = disciplinaryLevelMeta(ladder.suggestedLevel as DisciplinaryLevel).rank;

  return (
    <section className="flex flex-col gap-3">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div>
          <h3 className="text-sm font-semibold">Discipline</h3>
          <p className="text-muted-foreground flex flex-wrap items-center gap-x-1 text-xs">
            <span className="font-medium">
              {`Next step: ${
                DISCIPLINARY_LEVEL_LABELS[ladder.suggestedLevel as DisciplinaryLevel] ??
                ladder.suggestedLevel
              }`}
            </span>
            {ladder.atFinalStep ? (
              <span className="text-destructive">— this would end employment.</span>
            ) : null}
          </p>
        </div>
        {canIssue ? (
          <Button
            size="sm"
            variant={ladder.atFinalStep ? "destructive" : "default"}
            disabled={busy}
            onClick={onIssue}
          >
            {ladder.atFinalStep ? (
              <TriangleAlertIcon className="size-3.5" />
            ) : (
              <GavelIcon className="size-3.5" />
            )}
            Issue action
          </Button>
        ) : null}
      </div>

      <ol className="flex flex-wrap items-center gap-1.5">
        {LADDER.map((level) => {
          const meta = disciplinaryLevelMeta(level);
          const taken = meta.rank <= highestRank;
          const isNext = meta.rank === suggestedRank;
          return (
            <li key={level}>
              <span
                className={cn(
                  "flex items-center gap-1 rounded-full border px-2 py-0.5 text-[11px]",
                  taken && "border-red-500/40 bg-red-500/10 text-red-700 dark:text-red-400",
                  isNext && !taken && "border-primary text-primary border-dashed font-medium",
                  !taken && !isNext && "text-muted-foreground border-dashed",
                )}
              >
                {DISCIPLINARY_LEVEL_LABELS[level]}
              </span>
            </li>
          );
        })}
      </ol>

      {actions.length === 0 ? (
        <p className="text-muted-foreground border-border rounded-lg border border-dashed px-3 py-4 text-center text-xs">
          No disciplinary actions on record
        </p>
      ) : (
        <ul className="divide-border border-border divide-y rounded-lg border">
          {actions.map((action) => {
            const meta = disciplinaryLevelMeta(action.level as DisciplinaryLevel);
            const status = action.status as DisciplinaryStatus;
            return (
              <li
                key={action.id}
                data-testid={`discipline-${action.id}`}
                className="flex flex-wrap items-start gap-3 px-3 py-2.5"
              >
                <div className="min-w-0 flex-1">
                  <p className="flex items-center gap-2 text-sm font-medium">
                    <span className={meta.textClass}>{meta.label}</span>
                    <Badge
                      variant={
                        status === "Active"
                          ? "warning"
                          : status === "Rescinded"
                            ? "outline"
                            : "inactive"
                      }
                      className="px-1.5 py-0 text-[10px]"
                    >
                      {DISCIPLINARY_STATUS_LABELS[status] ?? status}
                    </Badge>
                    {action.acknowledgedAt ? (
                      <span className="text-muted-foreground text-[10px]">Acknowledged</span>
                    ) : null}
                  </p>
                  <p className="text-xs">{action.reason}</p>
                  <p className="text-muted-foreground text-[11px]">
                    Issued {formatUnixDate(action.issuedAt)}
                    {action.issuedBy?.name ? ` by ${action.issuedBy.name}` : ""}
                    {action.expiresAt ? ` · rolls off ${formatUnixDate(action.expiresAt)}` : ""}
                    {action.suspensionDays ? ` · ${action.suspensionDays} days` : ""}
                    {action.rescindReason ? ` · rescinded: ${action.rescindReason}` : ""}
                  </p>
                  {action.workerComment ? (
                    <p className="text-muted-foreground mt-1 text-[11px] italic">
                      &ldquo;{action.workerComment}&rdquo;
                    </p>
                  ) : null}
                </div>
                {canRescind && status === "Active" ? (
                  <Button
                    size="sm"
                    variant="ghost"
                    className="text-muted-foreground"
                    disabled={busy}
                    aria-label={`Rescind ${meta.label}`}
                    onClick={() => onRescind(action)}
                  >
                    <UndoIcon className="size-3.5" />
                    Rescind
                  </Button>
                ) : null}
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}
