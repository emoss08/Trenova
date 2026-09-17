import type { AITask, AITaskDescriptor } from "@/types/ai-provider";

/** The slice of a provider the router's candidate rule reads. */
export type RoutingProvider = {
  enabled: boolean;
  trusted: boolean;
  tasks: readonly AITask[];
};

export type AIReadiness = {
  /** Whether any provider is configured at all, enabled or not. */
  hasProviders: boolean;
  /** Tasks no enabled, suitably trusted provider is assigned to. */
  uncovered: AITaskDescriptor[];
  /**
   * Whether a conversation can be answered. Only chat itself is required: the
   * scope guard falls back to its deterministic rules when no classifier is
   * routed, so a missing ScopeClassification provider weakens the guard
   * without stopping the assistant.
   */
  assistantReady: boolean;
  /** Whether insight narration routes; insights themselves work without it. */
  narrationReady: boolean;
};

/**
 * Whether the work behind each feature has somewhere to go.
 *
 * This applies the same rule the server's router does when it picks a provider
 * — enabled, assigned to the task, and trusted where the task can reach the
 * ledger — so the banner never says a feature is ready when its first call
 * would come back "no provider configured".
 */
export function assessReadiness(
  providers: readonly RoutingProvider[],
  tasks: readonly AITaskDescriptor[],
): AIReadiness {
  const covered = new Set<AITask>();

  for (const descriptor of tasks) {
    const served = providers.some(
      (provider) =>
        provider.enabled &&
        provider.tasks.includes(descriptor.task) &&
        (!descriptor.requiresTrust || provider.trusted),
    );
    if (served) {
      covered.add(descriptor.task);
    }
  }

  return {
    hasProviders: providers.length > 0,
    uncovered: tasks.filter((descriptor) => !covered.has(descriptor.task)),
    assistantReady: covered.has("AssistantChat"),
    narrationReady: covered.has("OperationalInsights"),
  };
}
