import { RowActionsMenu, type RowAction } from "@/components/row-actions-menu";
import type { PreloadIntent } from "@/components/work-pane";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { VirtualRows, type VirtualRow } from "@/components/virtual-rows";
import type { DisciplinaryLadder, WorkerDisciplinaryActionRow } from "@/lib/graphql/worker-safety";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { ArrowNarrowRightIcon, ReverseLeftIcon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  DISCIPLINARY_LEVEL_LABELS,
  DISCIPLINARY_STATUS_LABELS,
  type DisciplinaryLevel,
  type DisciplinaryStatus,
} from "@trenova/shared/types/worker-safety";
import { useState } from "react";
import { ListMoreLine } from "../list-more";
import { ShimmerLine } from "../tab-skeleton-parts";
import type { WorkerList } from "../use-worker-list";
import { actionSummary, disciplineStair, type StairStep } from "./safety-model";

/** Each step a little taller than the last. */
const STEP_HEIGHTS = ["h-3.5", "h-5.5", "h-7.5", "h-9.5", "h-11.5", "h-13.5"];

type DisciplineStairProps = {
  ladder: DisciplinaryLadder;
  /** Every action issued, a page at a time. */
  actions: WorkerList<WorkerDisciplinaryActionRow>;
  /** The level being issued in the pane, previewed on its step. */
  preview: DisciplinaryLevel | null;
  canIssue: boolean;
  canRescind: boolean;
  onIssue: () => void;
  /** Reads the issue pane's code when the button is pointed at or focused. */
  issueIntent: PreloadIntent;
  /** Reads the action pane's code when a row is pointed at or focused. */
  openActionIntent: PreloadIntent;
  /** Reads the safety event pane's code when "Open the event" comes into reach. */
  preloadEvent: () => Promise<unknown>;
  onOpenAction: (action: WorkerDisciplinaryActionRow) => void;
  onOpenEvent: (safetyEventId: string) => void;
  onRescind: (action: WorkerDisciplinaryActionRow) => void;
};

/**
 * Progressive discipline as six ascending steps: the steps reached are filled, the active one
 * carries a ring, and the server's suggested next step is outlined in the warning tone; all of it
 * comes from the ladder, not from the actions loaded. Under it, every action issued, newest first,
 * a page at a time.
 */
export function DisciplineStair({
  ladder,
  actions,
  preview,
  canIssue,
  canRescind,
  onIssue,
  issueIntent,
  openActionIntent,
  preloadEvent,
  onOpenAction,
  onOpenEvent,
  onRescind,
}: DisciplineStairProps) {
  const t = useT();
  const rt = useRichT();
  const steps = disciplineStair(ladder, preview);
  const atTop = ladder.highestLevel === "Termination";
  const suggested = ladder.suggestedLevel as DisciplinaryLevel;
  // The steps rise once. A kept tab is hidden with `display: none`, which replays a CSS
  // animation when it is shown again, so the class comes off when the last step has risen.
  const [entered, setEntered] = useState(false);

  const stepNote = (step: StairStep) => {
    const action = ladder.activeActions.find((candidate) => candidate.level === step.level);
    if (action) return actionSummary(action, t);
    return step.next ? t("Suggested next step.") : t("Not reached.");
  };

  const rows: VirtualRow[] = actions.rows.map((action) => ({
    key: action.id,
    render: () => (
      <ActionRow
        action={action}
        canRescind={canRescind}
        openIntent={openActionIntent}
        preloadEvent={preloadEvent}
        onOpen={onOpenAction}
        onOpenEvent={onOpenEvent}
        onRescind={onRescind}
      />
    ),
  }));

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
                    onAnimationEnd={index === steps.length - 1 ? () => setEntered(true) : undefined}
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
            <Button size="sm" variant="secondary" {...issueIntent} onClick={onIssue}>
              {t("Issue action")}
            </Button>
          ) : null}
        </div>
      </div>
      {actions.isPending ? (
        <div aria-busy aria-label={t("Loading the actions")} className="flex flex-col gap-2 p-3">
          <ShimmerLine className="w-3/5" />
          <ShimmerLine className="w-2/5" />
        </div>
      ) : actions.isError ? (
        <div className="p-3">
          <ErrorState
            error={actions.error}
            layout="compact"
            title={t("The disciplinary actions could not be loaded")}
            onRetry={actions.refetch}
          />
        </div>
      ) : rows.length === 0 ? (
        <SectionPanelQuiet>{t("No disciplinary actions on record.")}</SectionPanelQuiet>
      ) : (
        <VirtualRows
          rows={rows}
          mask="card"
          className="max-h-72"
          contentClassName="px-1.5 py-1"
          hoverGlide
          estimateSize={56}
          onEndReached={actions.hasMore && !actions.loadMoreFailed ? actions.loadMore : undefined}
          footer={<ListMoreLine more={actions} />}
          aria-label={t("Disciplinary actions")}
        />
      )}
    </SectionPanel>
  );
}

function ActionRow({
  action,
  canRescind,
  openIntent,
  preloadEvent,
  onOpen,
  onOpenEvent,
  onRescind,
}: {
  action: WorkerDisciplinaryActionRow;
  canRescind: boolean;
  openIntent: PreloadIntent;
  preloadEvent: () => Promise<unknown>;
  onOpen: (action: WorkerDisciplinaryActionRow) => void;
  onOpenEvent: (safetyEventId: string) => void;
  onRescind: (action: WorkerDisciplinaryActionRow) => void;
}) {
  const t = useT();
  const status = action.status as DisciplinaryStatus;
  const level = action.level as DisciplinaryLevel;
  const menu: RowAction[] = [];
  if (action.safetyEventId) {
    const eventId = action.safetyEventId;
    menu.push({
      id: "event",
      label: t("Open the event"),
      icon: ArrowNarrowRightIcon,
      preload: preloadEvent,
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
      data-glide-target
      data-testid={`discipline-${action.id}`}
      className="grid grid-cols-[minmax(0,1fr)_auto] items-start gap-2 rounded-md p-2"
    >
      <button
        type="button"
        {...openIntent}
        onClick={() => onOpen(action)}
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
}
