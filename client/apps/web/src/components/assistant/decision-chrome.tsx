import { useAssistantAgent, useDelegateIdentity } from "@/components/agent-identity/agent-context";
import { AgentTile } from "@/components/agent-identity/agent-tile";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  AlertCircleIcon,
  Beaker02Icon,
  CheckCircleIcon,
  PauseCircleIcon,
  SlashCircle01Icon,
} from "@trenova/shared/components/icons";
import { useMemo, useState } from "react";
import { proposedByOther, type ProposalPresentation } from "./proposal-state";
import { WorkingDot } from "./voice/working-dot";

/**
 * Whether this view watched the state change. A card that moves from
 * "waiting" to "done" while someone looks at it says so with motion; a card
 * opened from history is simply in its state, with nothing to announce.
 */
export function useWatchedChange<T>(value: T): boolean {
  const [first] = useState(value);

  return value !== first;
}

/**
 * The mark of a settled decision. Work still running is the breathing dot,
 * the one loop the product allows; an outcome that lands while watched
 * settles with the confirm spring.
 */
export function OutcomeIcon({
  state,
  className,
}: {
  state: ProposalPresentation;
  className?: string;
}) {
  const watched = useWatchedChange(state);
  const glyph = cn("size-3.5 shrink-0", watched && "animate-confirm", className);

  switch (state) {
    case "failed":
      return <AlertCircleIcon key={state} aria-hidden className={cn(glyph, "text-danger")} />;
    case "done":
      return <CheckCircleIcon key={state} aria-hidden className={cn(glyph, "text-success")} />;
    case "running":
      return <WorkingDot working />;
    case "simulated":
      return (
        <Beaker02Icon key={state} aria-hidden className={cn(glyph, "text-foreground-muted")} />
      );
    case "held":
      return (
        <PauseCircleIcon key={state} aria-hidden className={cn(glyph, "text-foreground-muted")} />
      );
    default:
      return (
        <SlashCircle01Icon
          key={state}
          aria-hidden
          className={cn(glyph, "text-foreground-subtle")}
        />
      );
  }
}

/**
 * Who proposed a change, when it was not the agent the conversation is with:
 * "Proposed by Report Builder", beside that agent's mark. The conversation's
 * own agent heads the reply the card sits in, so its cards say nothing.
 */
export function ProposedBy({
  agentId,
  agentName,
}: {
  agentId?: string | null;
  agentName?: string | null;
}) {
  const t = useT();
  const conversation = useAssistantAgent();
  const fallback = useMemo(
    () => ({ id: agentId ?? "", name: agentName ?? "" }),
    [agentId, agentName],
  );
  const identity = useDelegateIdentity(fallback);
  const name = identity.name ?? "";

  if (!proposedByOther(conversation?.id, agentId) || name === "") {
    return null;
  }

  return (
    <p className="text-foreground-subtle flex min-w-0 items-center gap-1.5 text-xs">
      <AgentTile agent={identity} size="xs" />
      <span className="min-w-0 truncate">{t("Proposed by {0}", name)}</span>
    </p>
  );
}
