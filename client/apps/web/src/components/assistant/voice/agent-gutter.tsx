import {
  AGENT_ACCENTS,
  resolveAgentIdentity,
  type AgentIdentityInput,
} from "@/components/agent-identity/agent-identity";
import { cn } from "@trenova/shared/lib/utils";
import type { CSSProperties, ReactNode } from "react";

/**
 * The agent's accent over the thread it is working in.
 *
 * This used to be a two pixel bar down the avatar gutter, and the bar was
 * wrong for three reasons. It ran the full height of a scrolling column, so
 * it was the most prominent thing on screen while saying the least. It was a
 * hard edge at full saturation next to prose, which is the one place a strong
 * vertical is hardest to ignore. And it claimed a column of horizontal space
 * on a 400px panel to carry one bit of information.
 *
 * The accent is light now rather than a line: a wash at the head of the
 * column, brightest where the agent's mark is and gone within a couple of
 * hundred pixels, which puts the colour where the eye already is and lets the
 * text below it be text. It brightens while the agent is working and fades
 * when it stops, so the same signal carries both who and whether.
 */
export function AgentGutter({
  agent,
  working = false,
  className,
  children,
}: {
  agent: AgentIdentityInput | null | undefined;
  /** Lifts the wash while a turn is running. */
  working?: boolean;
  className?: string;
  children: ReactNode;
}) {
  const accent = agentSpineColor(agent);

  return (
    <div
      data-slot="agent-gutter"
      data-working={working}
      className={cn("ui-agent-glow relative flex min-h-0 min-w-0 flex-1 flex-col", className)}
      style={
        {
          "--agent-accent": accent,
          // Always faintly lit, because the wash is the agent's identity
          // here and not only its activity. Held to a fraction of the
          // working brightness so the two states stay tellable apart.
          "--agent-glow-rest": 0.5,
          // A column scrolls, so the wash is measured rather than a
          // percentage of a height that has no limit.
          "--agent-glow-extent": "200px",
        } as CSSProperties
      }
    >
      {children}
    </div>
  );
}

/** The accent a surface hangs off, for anything beside it that wants to match. */
export function agentSpineColor(agent: AgentIdentityInput | null | undefined): string {
  return agent ? AGENT_ACCENTS[resolveAgentIdentity(agent).accent] : "var(--border)";
}
