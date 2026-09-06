import {
  CostingControlPageDocument,
  ResolvedCostProfilePageDocument,
  UpdateCostCategoryDocument,
  UpdateCostingControlDocument,
  type CostCategoryUpdateInput,
  type CostingControlInput,
  type CostingControlPageQuery,
  type ResolvedCostProfilePageQuery,
} from "@trenova/graphql/generated/graphql";
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

export async function updateCostingControlGraphQL(input: CostingControlInput) {
  const data = await requestGraphQL({
    document: UpdateCostingControlDocument,
    operationName: "UpdateCostingControl",
    variables: { input },
  });
  return data.updateCostingControl;
}

export async function updateCostCategoryGraphQL(input: CostCategoryUpdateInput) {
  const data = await requestGraphQL({
    document: UpdateCostCategoryDocument,
    operationName: "UpdateCostCategory",
    variables: { input },
  });
  return data.updateCostCategory;
}
