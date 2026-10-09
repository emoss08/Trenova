import type {
  EditionDefinition,
  EditionPlan,
  EditionRouteMount,
  LoginPromptProps,
  ResolvedEdition,
} from "./types";
import { EDITION_ROUTE_MOUNTS } from "./types";
import type { ReactNode } from "react";
import type { RouteObject } from "react-router";

const NO_RESTRICTIONS: readonly string[] = Object.freeze([]);

function useNoRestrictions(): readonly string[] {
  return NO_RESTRICTIONS;
}

function EmptySlot(): ReactNode {
  return null;
}

function FallbackLoginPrompt({ fallback }: LoginPromptProps): ReactNode {
  return fallback;
}

export const NO_PLAN: EditionPlan = Object.freeze({
  useRestrictions: useNoRestrictions,
  loadRestrictions: async () => NO_RESTRICTIONS,
  restrictedRedirect: "/",
});

/** Declares an edition. Identity at runtime; it exists so the definition is type-checked. */
export function defineEdition(definition: EditionDefinition): EditionDefinition {
  return definition;
}

/**
 * Fills every part an edition leaves out with the host's default, so the host renders
 * slots and spreads routes without asking whether an edition is installed.
 */
export function resolveEdition(definition: EditionDefinition | undefined): ResolvedEdition {
  const routes = {} as Record<EditionRouteMount, readonly RouteObject[]>;
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
    protectedLoaders: definition?.protectedLoaders ?? [],
    plan: definition?.plan ?? NO_PLAN,
    messages: definition?.messages ?? {},
  };
}
