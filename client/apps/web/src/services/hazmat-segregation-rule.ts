import {
  CreateHazmatSegregationRuleDocument,
  UpdateHazmatSegregationRuleDocument,
  type HazmatSegregationRuleInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { HazmatSegregationRule } from "@/types/hazmat-segregation-rule";

export function toHazmatSegregationRuleInput(
  data: HazmatSegregationRule,
): HazmatSegregationRuleInput {
  return {
    status: data.status,
    name: data.name,
    description: data.description || null,
    classA: data.classA,
    classB: data.classB,
    hazmatAId: data.hazmatAId ?? null,
    hazmatBId: data.hazmatBId ?? null,
    segregationType: data.segregationType,
    minimumDistance: data.minimumDistance ?? null,
    distanceUnit: data.distanceUnit || null,
    hasExceptions: data.hasExceptions,
    exceptionNotes: data.exceptionNotes || null,
    referenceCode: data.referenceCode || null,
    regulationSource: data.regulationSource || null,
    version: data.version,
  };
}

export class HazmatSegregationRuleService {
  async create(data: HazmatSegregationRule) {
    const response = await requestGraphQL({
      document: CreateHazmatSegregationRuleDocument,
      operationName: "CreateHazmatSegregationRule",
      variables: { input: toHazmatSegregationRuleInput(data) },
    });

    return response.createHazmatSegregationRule;
  }

  async update(id: string, data: HazmatSegregationRule) {
    const response = await requestGraphQL({
      document: UpdateHazmatSegregationRuleDocument,
      operationName: "UpdateHazmatSegregationRule",
      variables: { id, input: toHazmatSegregationRuleInput(data) },
    });

    return response.updateHazmatSegregationRule;
  }
}
