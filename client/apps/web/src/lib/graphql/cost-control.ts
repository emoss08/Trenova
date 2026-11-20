import type {
  CostingControlPageQuery,
  CostingControlSaveInput,
  ResolvedCostProfilePageQuery,
} from "@trenova/graphql/generated/graphql";
import {
  CostingControlPageDocument,
  ResolvedCostProfilePageDocument,
  SaveCostingControlDocument,
} from "@trenova/graphql/generated/documents/cost-control/operations";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type CostingControl = CostingControlPageQuery["costingControl"];
export type CostCategory = CostingControl["categories"][number];
export type CostCategoryGLAccountLink = CostCategory["glAccounts"][number];
export type ResolvedCostProfile = ResolvedCostProfilePageQuery["resolvedCostProfile"];

export async function getCostingControlGraphQL(options?: {
  signal?: AbortSignal;
}): Promise<CostingControl> {
  const data = await requestGraphQL({
    document: CostingControlPageDocument,
    operationName: "CostingControlPage",
    signal: options?.signal,
  });
  return data.costingControl;
}

export async function getResolvedCostProfileGraphQL(
  asOfDate?: string,
  options?: { signal?: AbortSignal },
): Promise<ResolvedCostProfile> {
  const data = await requestGraphQL({
    document: ResolvedCostProfilePageDocument,
    operationName: "ResolvedCostProfilePage",
    variables: { asOfDate },
    signal: options?.signal,
  });
  return data.resolvedCostProfile;
}

/** Saves the control's settings and the changed categories together, or none of them. */
export async function saveCostingControlGraphQL(input: CostingControlSaveInput) {
  const data = await requestGraphQL({
    document: SaveCostingControlDocument,
    operationName: "SaveCostingControl",
    variables: { input },
  });
  return data.saveCostingControl;
}
