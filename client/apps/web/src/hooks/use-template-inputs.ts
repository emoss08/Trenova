import { formulaTemplate } from "@/lib/queries/formula-template";
import { useQueries, useQuery } from "@tanstack/react-query";
import type { TariffInputDeclaration } from "@trenova/shared/types/tariff";

/** The inputs a template's billing version declares; idle while no template is chosen. */
export function useTemplateInputs(templateId: string | null | undefined) {
  return useQuery({
    ...formulaTemplate.inputs(templateId ?? "", null),
    enabled: Boolean(templateId),
  });
}

/**
 * The declared inputs of several templates at once (an agreement's lanes), by template id.
 * `complete` is true once every template's inputs have loaded.
 */
export function useTemplatesInputs(templateIds: readonly string[]): {
  byTemplate: ReadonlyMap<string, readonly TariffInputDeclaration[]>;
  complete: boolean;
  isError: boolean;
} {
  const distinct = [...new Set(templateIds.filter(Boolean))];
  return useQueries({
    queries: distinct.map((templateId) => formulaTemplate.inputs(templateId, null)),
    combine: (results) => {
      const byTemplate = new Map<string, readonly TariffInputDeclaration[]>();
      results.forEach((result, index) => {
        const templateId = distinct[index];
        if (templateId && result.data) byTemplate.set(templateId, result.data);
      });
      return {
        byTemplate,
        complete: results.every((result) => result.isSuccess),
        isError: results.some((result) => result.isError),
      };
    },
  });
}
