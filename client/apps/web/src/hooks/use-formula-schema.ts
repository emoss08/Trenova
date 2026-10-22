import {
  buildKnownIdentifiers,
  type KnownIdentifiers,
} from "@/components/formula-editor/known-identifiers";
import { formulaTemplate } from "@/lib/queries/formula-template";
import { formulaCatalog } from "@trenova/shared/rules/formula";
import type { RulesRuntime } from "@trenova/shared/rules/runtime";
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
 * What the editor knows about names: the loaded rules module's catalog, the
 * record's schema and the template's inputs. `ready` stays false until the
 * schema has loaded, and the checks hold back every "unknown name" finding
 * until then.
 */
export function useKnownIdentifiers(
  rules: RulesRuntime,
  schemaId = "shipment",
  customVariables: readonly VariableDefinitionInput[] = [],
): KnownIdentifiers {
  const { data } = useFormulaSchema(schemaId);

  return useMemo(
    () => buildKnownIdentifiers(formulaCatalog(rules), data, customVariables),
    [rules, data, customVariables],
  );
}
