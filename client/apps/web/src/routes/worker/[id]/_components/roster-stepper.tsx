import { useRosterContextStore, useRosterStepper } from "@/routes/worker/roster-context-store";
import {
  workerPath,
  type WorkerTab,
  type WorkerTarget,
  type WorkerView,
} from "@/routes/worker/use-worker-state";
import { useNavigate } from "@tanstack/react-router";
import { ChevronDownIcon, ChevronUpIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";

export type RosterStepperProps = {
  workerId: string;
  /** Where the page stands, so the next worker opens on the same tab and view. */
  tab: WorkerTab;
  view: WorkerView | null;
};

/**
 * Previous and next through the roster the person opened this worker from, with where they are in
 * it ("3 of 14") and the roster's filter in the tooltip. Drawn only when the roster's page is still
 * in memory. Stepping changes the page's address, so unsaved edits ask first; if the person stays,
 * the roster's place is put back.
 */
export function RosterStepper({ workerId, tab, view }: RosterStepperProps) {
  const t = useT();
  const navigate = useNavigate();
  const stepper = useRosterStepper(workerId);
  if (!stepper.available) return null;

  const step = async (direction: "previous" | "next") => {
    const before = useRosterContextStore.getState().context;
    const next = await stepper.step(direction);
    if (!next) return;
    const target = (view ? { tab, view } : { tab }) as WorkerTarget;
    await navigate({ to: workerPath(next, target) });
    if (!window.location.pathname.endsWith(`/${encodeURIComponent(next)}`)) {
      useRosterContextStore.getState().setContext(before);
    }
  };

  const position =
    stepper.position === null
      ? null
      : stepper.total === null
        ? formatNumber(stepper.position)
        : t("{0} of {1}", stepper.position, stepper.total);

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <div
            role="group"
            aria-label={t("Step through the roster")}
            className="flex items-center gap-0.5"
          />
        }
      >
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          aria-label={t("Previous worker")}
          disabled={!stepper.canPrevious || stepper.stepping !== null}
          onClick={() => void step("previous")}
        >
          {stepper.stepping === "previous" ? (
            <Spinner className="size-3.5" />
          ) : (
            <ChevronUpIcon className="size-3.5" />
          )}
        </Button>
        {position ? (
          <span className="min-w-11 text-center text-xs text-muted-foreground tabular-nums">
            {position}
          </span>
        ) : null}
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          aria-label={t("Next worker")}
          disabled={!stepper.canNext || stepper.stepping !== null}
          onClick={() => void step("next")}
        >
          {stepper.stepping === "next" ? (
            <Spinner className="size-3.5" />
          ) : (
            <ChevronDownIcon className="size-3.5" />
          )}
        </Button>
      </TooltipTrigger>
      {stepper.label ? (
        <TooltipContent>{t("Stepping through: {0}", stepper.label)}</TooltipContent>
      ) : null}
    </Tooltip>
  );
}
