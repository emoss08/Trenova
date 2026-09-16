import type { AIProvider } from "@/types/ai-provider";
import type { IntegrationCatalogItem } from "@/types/integration";

/** The catalog type the marketplace opens the provider manager for. */
export const AI_PROVIDERS_CATALOG_TYPE = "AIProviders";

/**
 * The marketplace card for AI providers, built from the provider list.
 *
 * Providers are rows in their own table rather than an integration credential,
 * because an organization holds several and routes work between them. The
 * marketplace is still where an administrator goes to connect anything, so the
 * card is assembled here in the same shape the server builds the others: the
 * category, status and filters all behave as if it had come from the catalog.
 */
export function aiProvidersCatalogItem(providers: readonly AIProvider[]): IntegrationCatalogItem {
  const configured = providers.length > 0;
  const enabled = providers.some((provider) => provider.enabled);

  return {
    type: AI_PROVIDERS_CATALOG_TYPE,
    name: "AI Providers",
    description:
      "Model endpoints for the assistant, agents and operational insights. Connect a hosted API, a gateway, or a model server on your own hardware, and choose which one handles each kind of work.",
    category: "ArtificialIntelligence",
    categoryLabel: "AI & Automation",
    logoUrl: "",
    links: [],
    color: "#7c3aed",
    glowFrom: "#7c3aed",
    glowTo: "#0ea5e9",
    featured: true,
    sortOrder: 29,
    primaryActionLabel: actionLabel(providers.length),
    enabled,
    configured,
    status: {
      connection: enabled ? "connected" : "disconnected",
      connectionLabel: enabled ? "Connected" : "Disconnected",
      configuration: configured ? "configured" : "needs_setup",
      configurationLabel: configured ? "Configured" : "Needs Setup",
    },
    configSpec: [],
    supportsTestConnect: true,
  };
}

function actionLabel(count: number): string {
  if (count === 0) {
    return "Add Provider";
  }

  return count === 1 ? "Manage 1 Provider" : `Manage ${count} Providers`;
}
