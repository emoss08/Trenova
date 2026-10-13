import {
  buildKnownIdentifiers,
  FALLBACK_SCHEMA,
  type KnownIdentifiers,
} from "@/components/formula-editor/known-identifiers";
import { formulaTemplate } from "@/lib/queries/formula-template";
import type {
  FormulaSchemaResponse,
  VariableDefinitionInput,
} from "@trenova/shared/types/formula-template";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";

const FALLBACK_RESPONSE: FormulaSchemaResponse = {
  schemaId: "shipment",
  variables: FALLBACK_SCHEMA.variables,
  functions: FALLBACK_SCHEMA.functions,
};

export function useFormulaSchema(schemaId = "shipment") {
  return useQuery({
    ...formulaTemplate.schema(schemaId),
    staleTime: Infinity,
    placeholderData: FALLBACK_RESPONSE,
  });
}

export function useKnownIdentifiers(
  schemaId = "shipment",
  customVariables: VariableDefinitionInput[] = [],
): KnownIdentifiers {
  const { data } = useFormulaSchema(schemaId);

  return useMemo(() => buildKnownIdentifiers(data, customVariables), [data, customVariables]);
}
