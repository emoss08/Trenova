import type { DeskDock } from "@/components/desk-chat/desk-thread";
import type { AssistantThread, CaseChecklist, CaseRecord, StepAbility } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Link } from "react-router";
import { stepLabel, stepPage, stepPageLabel, stepPrompt } from "./case-labels";

/**
 * A checklist step as something to do. The button is the step's own; who
 * takes it follows from what the agents hold. The conversation's agent is
 * asked when it can; when it cannot but another agent the person may use
 * can, the same press hands the step to that agent inside this conversation,
 * and its work and answer land in the thread; when no agent they may use
 * can, it opens the page where the person takes it themselves. A step is
 * never sent to an agent that can only say no.
 */
export function CaseStepAction({
  step,
  ability,
  record,
  checklist,
  thread,
  dock,
  primary,
}: {
  step: string;
  ability: StepAbility | undefined;
  record: CaseRecord;
  checklist: CaseChecklist | null;
  thread: AssistantThread;
  dock: DeskDock;
  /** The next step, drawn as the card's main action. */
  primary: boolean;
}) {
  const t = useT();
  const className = primary ? "dk-bt dk-sm dk-cs-go" : "dk-cs-step";
  const via = ability?.via ?? "Agent";

  if (via === "Person") {
    const page = stepPage(step, record);
    if (!page) {
      return null;
    }
    return (
      <Link
        to={page}
        className={cn(className, "dk-cs-self")}
        title={t("No agent you can use can take this step. Do it here.")}
      >
        {stepPageLabel(step, record, t)}
      </Link>
    );
  }

  const directed = via === "Ask" && ability?.agentId ? ability : null;
  return (
    <button
      type="button"
      className={className}
      disabled={dock.busy || !thread.canContinue}
      title={
        dock.busy
          ? t("Waits for the reply under way")
          : directed
            ? t("{0} takes this step", directed.agentName)
            : undefined
      }
      onClick={() =>
        dock.ask(
          stepPrompt(step, record, checklist, t),
          directed ? { directedAgentId: directed.agentId } : undefined,
        )
      }
    >
      {stepLabel(step, t, checklist)}
    </button>
  );
}

/**
 * Whether no agent the person may use can take any step the checklist
 * offers, so every step opens where the person does it.
 */
export function onlyPeopleCanTake(
  checklist: CaseChecklist | null,
  abilities: Record<string, StepAbility>,
): boolean {
  const steps = new Set<string>();
  if (checklist?.next) steps.add(checklist.next);
  for (const item of checklist?.items ?? []) {
    if (item.step && (item.state === "Blocked" || item.state === "Pending")) steps.add(item.step);
  }
  if (steps.size === 0) {
    return false;
  }
  for (const step of steps) {
    if (abilities[step]?.via !== "Person") {
      return false;
    }
  }

  return true;
}
