import { formulaTemplate } from "@/lib/queries/formula-template";
import type { QueryClient } from "@tanstack/react-query";
import type { TariffInputDeclaration } from "@trenova/shared/types/tariff";
import type { FieldValues, Resolver } from "react-hook-form";

/** Each tariff's declared inputs, by formula template id, for the templates a form names. */
export type TemplateInputsById = ReadonlyMap<string, readonly TariffInputDeclaration[]>;

/**
 * The declared inputs of the billing version of every template named, read through the query
 * cache the forms' fields already fill. A template whose inputs cannot be read is left out:
 * its values are still checked by the server on save.
 */
export async function loadTemplateInputs(
  queryClient: QueryClient,
  templateIds: readonly string[],
): Promise<TemplateInputsById> {
  const distinct = [...new Set(templateIds.filter(Boolean))];
  const entries = await Promise.all(
    distinct.map(async (templateId) => {
      try {
        const inputs = await queryClient.ensureQueryData(formulaTemplate.inputs(templateId, null));
        return [templateId, inputs] as const;
      } catch {
        return null;
      }
    }),
  );
  return new Map(entries.filter((entry) => entry !== null));
}

/**
 * A resolver that checks a form's tariff input values against the declarations of the
 * templates the form names, so a mistyped or missing input is reported at its field before
 * the save. `build` makes the zod resolver for one set of declarations.
 */
export function tariffInputsResolver<TValues extends FieldValues>(
  queryClient: QueryClient,
  templateIdsOf: (values: TValues) => string[],
  build: (inputs: TemplateInputsById) => Resolver<TValues>,
): Resolver<TValues> {
  return async (values, context, options) => {
    const inputs = await loadTemplateInputs(queryClient, templateIdsOf(values));
    return build(inputs)(values, context, options);
  };
}
