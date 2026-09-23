import { useAssistantAgent, useDelegateIdentity } from "@/components/agent-identity/agent-context";
import { AgentTile } from "@/components/agent-identity/agent-tile";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  CircleAlertIcon,
  CircleCheckIcon,
  CircleSlashIcon,
  FlaskConicalIcon,
  PauseCircleIcon,
  type LucideIcon,
} from "lucide-react";
import { useMemo, useState, type ReactNode } from "react";
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

type StateTone = "neutral" | "info" | "success" | "warning" | "danger";

/** Each state's phase, as a tone: waiting on a person is the only warning. */
const STATE_TONE: Record<ProposalPresentation, StateTone> = {
  awaiting: "warning",
  held: "neutral",
  running: "info",
  done: "success",
  failed: "danger",
  declined: "neutral",
  simulated: "info",
  closed: "neutral",
};

function stateLabel(state: ProposalPresentation, t: ReturnType<typeof useT>): string {
  switch (state) {
    case "awaiting":
      return t("Needs approval");
    case "held":
      return t("On hold");
    case "running":
      return t("Running");
    case "done":
      return t("Done");
    case "failed":
      return t("Did not run");
    case "declined":
      return t("Rejected");
    case "simulated":
      return t("Simulated");
    default:
      return t("Expired");
  }
}

export function DecisionStateBadge({ state }: { state: ProposalPresentation }) {
  const t = useT();
  const watched = useWatchedChange(state);

  return (
    <Badge
      key={state}
      variant={STATE_TONE[state]}
      className={cn("h-4.5 shrink-0 px-1.5 text-2xs", watched && "animate-rise")}
    >
      {stateLabel(state, t)}
    </Badge>
  );
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
      return <CircleAlertIcon key={state} aria-hidden className={cn(glyph, "text-danger")} />;
    case "done":
      return <CircleCheckIcon key={state} aria-hidden className={cn(glyph, "text-success")} />;
    case "running":
      return <WorkingDot working />;
    case "simulated":
      return (
        <FlaskConicalIcon key={state} aria-hidden className={cn(glyph, "text-foreground-muted")} />
      );
    case "held":
      return (
        <PauseCircleIcon key={state} aria-hidden className={cn(glyph, "text-foreground-muted")} />
      );
    default:
      return (
        <CircleSlashIcon key={state} aria-hidden className={cn(glyph, "text-foreground-subtle")} />
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

/**
 * The open card for a decision someone has to make: the artifact chrome in
 * miniature. The kind's mark in a sunken well, the title, and the state as
 * a badge; then the body and the buttons. A hairline and the surface radius
 * separate it from the conversation, and nothing about it glows or tints.
 */
export function DecisionFrame({
  icon: Icon,
  title,
  state,
  byline,
  children,
  footer,
}: {
  icon: LucideIcon;
  title: string;
  state: ProposalPresentation;
  /** Who proposed it, when that is not the conversation's own agent. */
  byline?: ReactNode;
  children: ReactNode;
  footer?: ReactNode;
}) {
  return (
    <section
      data-slot="decision"
      data-state={state}
      className="border-border bg-card flex min-w-0 flex-col overflow-hidden rounded-lg border"
    >
      <header className="flex h-10 shrink-0 items-center gap-2.5 px-3">
        <span className="bg-sunken text-foreground-muted flex size-6 shrink-0 items-center justify-center rounded-md">
          <Icon aria-hidden className="size-3.5" />
        </span>
        <h3 className="min-w-0 flex-1 truncate text-sm font-semibold">{title}</h3>
        <DecisionStateBadge state={state} />
      </header>
      <div className="flex min-w-0 flex-col gap-2.5 px-3 pb-3">
        {byline}
        {children}
      </div>
      {footer}
    </section>
  );
}

/**
 * A decided card, kept as a receipt: what it was, and what came of it. It
 * drops everything that existed to help decide. When the outcome arrives
 * while someone is watching, the new line rises into place and the mark
 * settles, so "approved" becoming "done" is seen happening.
 */
export function DecisionReceipt({
  state,
  summary,
  byline,
  arrived = false,
  children,
  footer,
}: {
  state: ProposalPresentation;
  summary: ReactNode;
  /** Who proposed it, when that is not the conversation's own agent. */
  byline?: ReactNode;
  /** It has just replaced the open card, while someone watched. */
  arrived?: boolean;
  children: ReactNode;
  footer?: ReactNode;
}) {
  const watched = useWatchedChange(state);

  return (
    <section
      data-slot="decision"
      data-state={state}
      className={cn(
        "border-border-subtle bg-card flex min-w-0 flex-col gap-2 rounded-lg border px-3 py-2.5 text-xs",
        arrived && "animate-rise",
      )}
    >
      <div className="flex min-w-0 items-start gap-2.5">
        <span className="bg-sunken flex size-6 shrink-0 items-center justify-center rounded-md">
          <OutcomeIcon state={state} />
        </span>
        <div className="flex min-w-0 flex-1 flex-col gap-0.5 pt-0.5">
          <div className="text-foreground text-sm leading-snug">{summary}</div>
          {byline}
          <div
            key={state}
            className={cn("text-foreground-muted leading-relaxed", watched && "animate-rise")}
          >
            {children}
          </div>
        </div>
      </div>
      {footer}
    </section>
  );
}
