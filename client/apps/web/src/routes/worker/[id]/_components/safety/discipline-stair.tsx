import { HoverGlideGroup } from "@/components/hover-glide";
import { RowActionsMenu, type RowAction } from "@/components/row-actions-menu";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import type { DisciplinaryLadder, WorkerDisciplinaryActionRow } from "@/lib/graphql/worker-safety";
import { ArrowNarrowRightIcon, ReverseLeftIcon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import {
  DISCIPLINARY_LEVEL_LABELS,
  DISCIPLINARY_STATUS_LABELS,
  type DisciplinaryLevel,
  type DisciplinaryStatus,
} from "@trenova/shared/types/worker-safety";
import { useEffect, useState } from "react";
import { disciplineStair, sortActions, type StairStep } from "./safety-model";

/** Each step a little taller than the last. */
const STEP_HEIGHTS = ["h-3.5", "h-5.5", "h-7.5", "h-9.5", "h-11.5", "h-13.5"];

export function actionSummary(action: WorkerDisciplinaryActionRow, t: TranslateFn): string {
  if (action.status === "Rescinded") {
    return action.rescindReason ? t("Rescinded: {0}", action.rescindReason) : t("Rescinded");
  }
  const parts = [
    action.issuedBy?.name
      ? t("{0} by {1}", formatUnixDateMedium(action.issuedAt), action.issuedBy.name)
      : formatUnixDateMedium(action.issuedAt),
    action.suspensionDays
      ? t("{0, plural, one {# day off} other {# days off}}", action.suspensionDays)
      : null,
    action.expiresAt
      ? action.status === "Expired"
        ? t("rolled off {0}", formatUnixDateMedium(action.expiresAt))
        : t("rolls off {0}", formatUnixDateMedium(action.expiresAt))
      : null,
    action.status === "Active"
      ? action.acknowledgedAt
        ? t("Acknowledged")
        : t("waiting for the acknowledgement")
      : null,
  ].filter(Boolean);
  return parts.join(" · ");
}

type DisciplineStairProps = {
  ladder: DisciplinaryLadder;
  actions: readonly WorkerDisciplinaryActionRow[];
  /** The level being issued in the pane, previewed on its step. */
  preview: DisciplinaryLevel | null;
  canIssue: boolean;
  canRescind: boolean;
  onIssue: () => void;
  onOpenAction: (action: WorkerDisciplinaryActionRow) => void;
  onOpenEvent: (safetyEventId: string) => void;
  onRescind: (action: WorkerDisciplinaryActionRow) => void;
};

/**
 * Progressive discipline as six ascending steps: the steps reached are filled, the active one
 * carries a ring, and the server's suggested next step is outlined in the warning tone. Under it,
 * every action issued, active ones first.
 */
export function DisciplineStair({
  ladder,
  actions,
  preview,
  canIssue,
  canRescind,
  onIssue,
  onOpenAction,
  onOpenEvent,
  onRescind,
}: DisciplineStairProps) {
  const t = useT();
  const rt = useRichT();
  const steps = disciplineStair(ladder, preview);
  const atTop = ladder.highestLevel === "Termination";
  const suggested = ladder.suggestedLevel as DisciplinaryLevel;
  const sorted = sortActions(actions);
  const [entered, setEntered] = useState(false);
  useEffect(() => {
    const timer = window.setTimeout(() => setEntered(true), 600);
    return () => window.clearTimeout(timer);
  }, []);

  const stepNote = (step: StairStep) => {
    const action = sorted.find(
      (candidate) => candidate.level === step.level && candidate.status === "Active",
    );
    if (action) return actionSummary(action, t);
    return step.next ? t("Suggested next step.") : t("Not reached.");
  };

  return (
    <SectionPanel title={t("Discipline")} hint={t("Progressive, rolls off after 12 months")}>
      <div className="flex flex-col gap-3 border-b border-border-subtle px-3 pt-3.5 pb-3">
        <div
          role="img"
          aria-label={t(
            "Discipline steps. {0} {1}",
            ladder.highestLevel
              ? t(
                  "Active: {0}.",
                  DISCIPLINARY_LEVEL_LABELS[ladder.highestLevel as DisciplinaryLevel].toLowerCase(),
                )
              : t("No active action."),
            atTop ? "" : t("Next: {0}.", DISCIPLINARY_LEVEL_LABELS[suggested].toLowerCase()),
          )}
          data-testid="discipline-stair"
          className="grid grid-cols-6 items-end gap-0.75"
        >
          {steps.map((step, index) => (
            <Tooltip key={step.level}>
              <TooltipTrigger
                render={
                  <div
                    data-level={step.level}
                    data-state={
                      step.preview
                        ? "preview"
                        : step.active
                          ? "active"
                          : step.reached
                            ? "reached"
                            : step.next
                              ? "next"
                              : "rest"
                    }
                    className={cn(
                      "flex min-w-0 flex-col gap-1.5",
                      !entered && "motion-safe:animate-rise",
                    )}
                    style={{ animationDelay: `${index * 40}ms` }}
                  />
                }
              >
                <span
                  className={cn(
                    "block rounded-xs border transition-[background-color,border-color,box-shadow] duration-320 ease-settle",
                    STEP_HEIGHTS[index],
                    step.preview
                      ? "border-warning-foreground bg-warning-subtle ring-1 ring-warning-foreground ring-offset-2 ring-offset-card"
                      : step.reached
                        ? "border-ink bg-ink"
                        : step.next
                          ? "border-dashed border-warning-foreground bg-warning-subtle"
                          : "border-border-subtle bg-card",
                    step.active &&
                      !step.preview &&
                      "ring-1 ring-ink ring-offset-2 ring-offset-card",
                  )}
                />
                <span
                  className={cn(
                    "min-h-7 text-2xs text-balance",
                    step.preview || (step.next && !step.reached)
                      ? "font-medium text-warning-foreground"
                      : step.reached
                        ? "text-foreground"
                        : "text-muted-foreground",
                    step.active && "font-medium",
                  )}
                >
                  {t(DISCIPLINARY_LEVEL_LABELS[step.level])}
                </span>
              </TooltipTrigger>
              <TooltipContent>
                <span className="block font-medium">
                  {t(DISCIPLINARY_LEVEL_LABELS[step.level])}
                </span>
                <span className="block">{stepNote(step)}</span>
              </TooltipContent>
            </Tooltip>
          ))}
        </div>
        <div className="flex items-center justify-between gap-3">
          <p className="text-sm text-muted-foreground">
            {atTop
              ? rt("Termination issued. <b>The last step</b>", {
                  b: (chunk) => <b className="font-medium text-foreground">{chunk}</b>,
                })
              : rt(
                  "Next step: <b>{0}</b>",
                  { b: (chunk) => <b className="font-medium text-foreground">{chunk}</b> },
                  t(DISCIPLINARY_LEVEL_LABELS[suggested]).toLowerCase(),
                )}
          </p>
          {canIssue ? (
            <Button size="sm" variant="secondary" onClick={onIssue}>
              {t("Issue action")}
            </Button>
          ) : null}
        </div>
      </div>
      {sorted.length === 0 ? (
        <SectionPanelQuiet>{t("No disciplinary actions on record.")}</SectionPanelQuiet>
      ) : (
        <ScrollArea viewportClassName="max-h-72" maskVariant="card">
          <HoverGlideGroup className="flex flex-col px-1.5 py-1">
            {sorted.map((action) => {
              const status = action.status as DisciplinaryStatus;
              const level = action.level as DisciplinaryLevel;
              const menu: RowAction[] = [];
              if (action.safetyEventId) {
                const eventId = action.safetyEventId;
                menu.push({
                  id: "event",
                  label: t("Open the event"),
                  icon: ArrowNarrowRightIcon,
                  onSelect: () => onOpenEvent(eventId),
                });
              }
              if (canRescind && status === "Active") {
                menu.push({
                  id: "rescind",
                  label: t("Rescind…"),
                  icon: ReverseLeftIcon,
                  destructive: true,
                  onSelect: () => onRescind(action),
                });
              }
              return (
                <div
                  key={action.id}
                  data-glide-target
                  data-testid={`discipline-${action.id}`}
                  className="grid grid-cols-[minmax(0,1fr)_auto] items-start gap-2 rounded-md p-2"
                >
                  <button
                    type="button"
                    onClick={() => onOpenAction(action)}
                    className="min-w-0 rounded-sm ui-focus-ring text-left"
                  >
                    <span className="flex flex-wrap items-center gap-x-2 gap-y-1">
                      <span
                        className={cn(
                          "font-medium",
                          status !== "Active" && "text-muted-foreground",
                          status === "Rescinded" && "line-through decoration-foreground-subtle",
                        )}
                      >
                        {t(DISCIPLINARY_LEVEL_LABELS[level] ?? level)}
                      </span>
                      <Badge variant="neutral" dot={status === "Active"}>
                        {t(DISCIPLINARY_STATUS_LABELS[status] ?? status)}
                      </Badge>
                    </span>
                    <span className="mt-0.5 block text-xs text-muted-foreground tabular-nums">
                      {actionSummary(action, t)}
                    </span>
                  </button>
                  <RowActionsMenu label={t("Action options")} actions={menu} />
                </div>
              );
            })}
          </HoverGlideGroup>
        </ScrollArea>
      )}
    </SectionPanel>
  );
}
