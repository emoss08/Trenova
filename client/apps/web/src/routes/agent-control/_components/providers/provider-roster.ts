import type { AIProviderRow } from "@/lib/graphql/ai-provider";

/**
 * Providers in the order work is offered to them: lowest priority number
 * first, and among equals the enabled ones before the disabled, then by
 * name. The list is the routing order, which is the one fact about
 * providers an administrator needs to see at a glance.
 */
export function sortProvidersByRouting(providers: readonly AIProviderRow[]): AIProviderRow[] {
  return [...providers].sort((a, b) => {
    if (a.priority !== b.priority) {
      return a.priority - b.priority;
    }
    if (a.enabled !== b.enabled) {
      return a.enabled ? -1 : 1;
    }
    return a.name.localeCompare(b.name);
  });
}

/** Providers whose name, model or endpoint contains the query, case ignored. */
export function filterProviders(
  providers: readonly AIProviderRow[],
  query: string,
): AIProviderRow[] {
  const needle = query.trim().toLowerCase();
  if (needle === "") {
    return [...providers];
  }

  return providers.filter((provider) =>
    `${provider.name} ${provider.model} ${provider.baseUrl}`.toLowerCase().includes(needle),
  );
}
