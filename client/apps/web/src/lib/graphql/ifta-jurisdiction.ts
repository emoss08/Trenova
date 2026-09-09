import {
  IftaJurisdictionsDocument,
  type IftaJurisdictionFieldsFragment,
  type IftaJurisdictionsQuery,
  type IftaJurisdictionsQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { queryOptions } from "@tanstack/react-query";

export type IftaJurisdiction = IftaJurisdictionFieldsFragment;

export const IFTA_JURISDICTIONS_KEY = "ifta-jurisdictions";

const ONE_DAY_MS = 24 * 60 * 60 * 1000;

export async function fetchIftaJurisdictions(
  membersOnly = false,
  options?: { signal?: AbortSignal },
): Promise<IftaJurisdiction[]> {
  const data = await requestGraphQL<IftaJurisdictionsQuery, IftaJurisdictionsQueryVariables>({
    document: IftaJurisdictionsDocument,
    operationName: "IftaJurisdictions",
    variables: { membersOnly },
    signal: options?.signal,
  });
  return data.iftaJurisdictions as IftaJurisdiction[];
}

export function iftaJurisdictionsQuery(membersOnly = false) {
  return queryOptions({
    queryKey: [IFTA_JURISDICTIONS_KEY, membersOnly] as const,
    queryFn: ({ signal }) => fetchIftaJurisdictions(membersOnly, { signal }),
    staleTime: ONE_DAY_MS,
    gcTime: ONE_DAY_MS,
  });
}
