import type { ImportTemplatesResponse } from "@trenova/shared/types/formula-template";

const BASE = "/billing/configuration-files/formula-templates";

/** Every route the Formula Studio navigates to, spelled once. */
export const formulaTemplateRoutes = {
  list: BASE,
  new: `${BASE}/new`,
  edit: (templateId: string) => `${BASE}/${templateId}/edit`,
} as const;

/**
 * Where to land after an import: straight into the one template that was
 * created, or the list when there are several to choose from.
 */
export function importLandingRoute(response: ImportTemplatesResponse): string {
  const only = response.created.length === 1 ? response.created[0] : undefined;
  return only?.id ? formulaTemplateRoutes.edit(only.id) : formulaTemplateRoutes.list;
}
