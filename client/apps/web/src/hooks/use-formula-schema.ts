import {
  buildKnownIdentifiers,
  type KnownIdentifiers,
} from "@/components/formula-editor/known-identifiers";
import { formulaTemplate } from "@/lib/queries/formula-template";
import type { VariableDefinitionInput } from "@trenova/shared/types/formula-template";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";

export function useFormulaSchema(schemaId = "shipment") {
  return useQuery({
    ...formulaTemplate.schema(schemaId),
    staleTime: Infinity,
  });
}

/**
 * What the editor knows about names: the engine's schema description and the
 * template's inputs. `ready` stays false until the description has loaded, and
 * the checks hold back every "unknown name" finding until then.
 */
export function useKnownIdentifiers(
  schemaId = "shipment",
  customVariables: VariableDefinitionInput[] = [],
): KnownIdentifiers {
  const { data } = useFormulaSchema(schemaId);

  return useMemo(() => buildKnownIdentifiers(data, customVariables), [data, customVariables]);
}
