import { AGENT_ACCENTS, resolveAgentIdentity, type AgentIdentityInput } from "@/components/agent-identity/agent-identity";
import { cn } from "@trenova/shared/lib/utils";
import type { CSSProperties, ReactNode } from "react";

/**
 * The agent's accent as the thread's spine: a two pixel hairline down the
 * avatar gutter, in the agent's own accent, that everything the agent did
 * hangs off. A thread reads as one agent's work rather than as a stack of
 * cards, and two conversations with two desks tell apart at a glance.
 *
 * The line is drawn with the agent's accent token; no component here picks
 * a colour of its own.
 */
export function AgentGutter({
  agent,
  className,
  children,
}: {
  agent: AgentIdentityInput | null | undefined;
  className?: string;
  children: ReactNode;
}) {
  const accent = agent ? AGENT_ACCENTS[resolveAgentIdentity(agent).accent] : "var(--border)";

  return (
    <div
      data-slot="agent-gutter"
      className={cn("relative flex min-h-0 min-w-0 flex-1 flex-col", className)}
      style={{ "--agent-spine": accent } as CSSProperties}
    >
      <span
        aria-hidden
        className="pointer-events-none absolute inset-y-0 left-[calc(0.875rem-1px)] w-0.5 rounded-full opacity-60"
        style={{ backgroundColor: "var(--agent-spine)" }}
      />
      {children}
    </div>
  );
}

/** The accent a surface hangs off, for anything beside the gutter that wants to match it. */
export function agentSpineColor(agent: AgentIdentityInput | null | undefined): string {
  return agent ? AGENT_ACCENTS[resolveAgentIdentity(agent).accent] : "var(--border)";
}
