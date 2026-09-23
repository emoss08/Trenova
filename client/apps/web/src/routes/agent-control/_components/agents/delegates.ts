import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import type { TriggerMode } from "@/types/assistant";

/** An agent this one may hand work to, as the form and the roster draw it. */
export type DelegateSummary = {
  id: string;
  name: string;
  icon: string;
  accent: string;
  /** A disabled agent stays on the list and is refused when asked. */
  enabled: boolean;
  triggerMode: TriggerMode;
};

/** How many names the roster spells out before the rest become a count. */
const NAMED_LIMIT = 2;

/**
 * The agents on the allowlist that still exist, in the order they were
 * chosen. The server lists them in `delegates` and omits one deleted since;
 * `delegateIds` keeps the order. An id with no agent behind it is left out,
 * so the next save drops it rather than sending an agent the server cannot
 * find.
 */
export function savedDelegates(
  agent: Pick<AgentDefinitionRow, "delegateIds" | "delegates">,
): DelegateSummary[] {
  const delegates = agent.delegates ?? [];
  const byId = new Map(delegates.map((delegate) => [delegate.id, delegate]));
  const ordered = (agent.delegateIds ?? []).flatMap((id) => {
    const delegate = byId.get(id);
    return delegate ? [delegate] : [];
  });
  // An agent listed but missing from the ids — a server that orders only
  // one of them — still belongs on the list, after the ordered ones.
  const listed = new Set(ordered.map((delegate) => delegate.id));
  const rest = delegates.filter((delegate) => !listed.has(delegate.id));

  return [...ordered, ...rest].map((delegate) => ({
    id: delegate.id,
    name: delegate.name,
    icon: delegate.icon,
    accent: delegate.accent,
    enabled: delegate.enabled,
    triggerMode: delegate.triggerMode,
  }));
}

/**
 * The allowlist in one short line for a roster row: "Can ask Report Builder,
 * Dispatch desk +1". Empty for an agent that asks nobody.
 */
export function delegatesLine(delegates: readonly DelegateSummary[], t: TranslateFn): string {
  if (delegates.length === 0) {
    return "";
  }
  const shown = delegates
    .slice(0, NAMED_LIMIT)
    .map((delegate) => delegate.name)
    .join(", ");
  const hidden = delegates.length - NAMED_LIMIT;

  return hidden > 0 ? t("Can ask {0} +{1}", shown, hidden) : t("Can ask {0}", shown);
}

/**
 * Whether an agent may have an allowlist at all. Only an agent people talk
 * to hands work on: a run nobody watches has nobody the other agent's work
 * would be for, and the server refuses the list on any other trigger.
 */
export function canDelegate(triggerMode: TriggerMode): boolean {
  return triggerMode === "Chat";
}
