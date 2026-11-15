import type {
  EditionDefinition,
  EditionRoute,
  EditionPlan,
  EditionRouteMount,
  LoginPromptProps,
  ResolvedEdition,
} from "./types";
import { EDITION_ROUTE_MOUNTS } from "./types";
import type { ReactNode } from "react";

function EmptySlot(): ReactNode {
  return null;
}

function FallbackLoginPrompt({ fallback }: LoginPromptProps): ReactNode {
  return fallback;
}

export const NO_PLAN: EditionPlan = Object.freeze({});

/** Declares an edition. Identity at runtime; it exists so the definition is type-checked. */
export function defineEdition(definition: EditionDefinition): EditionDefinition {
  return definition;
}

/**
 * Fills every part an edition leaves out with the host's default, so the host renders
 * slots and spreads routes without asking whether an edition is installed.
 */
export function resolveEdition(definition: EditionDefinition | undefined): ResolvedEdition {
  const routes = {} as Record<EditionRouteMount, readonly EditionRoute[]>;
  for (const mount of EDITION_ROUTE_MOUNTS) {
    routes[mount] = definition?.routes?.[mount] ?? [];
  }

  return {
    id: definition?.id ?? "self-hosted",
    name: definition?.name ?? "Trenova",
    routes,
    adminLinks: definition?.adminLinks ?? [],
    navItems: definition?.navItems ?? [],
    slots: {
      AppBanner: definition?.slots?.AppBanner ?? EmptySlot,
      RootHost: definition?.slots?.RootHost ?? EmptySlot,
      LoginPrompt: definition?.slots?.LoginPrompt ?? FallbackLoginPrompt,
    },
    protectedGates: definition?.protectedGates ?? [],
    plan: definition?.plan ?? NO_PLAN,
    messages: definition?.messages ?? {},
  };
}
