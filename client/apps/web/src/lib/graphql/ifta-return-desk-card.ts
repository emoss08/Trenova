import type { DeskIftaReturnCardQuery } from "@trenova/graphql/generated/graphql";
import { DeskIftaReturnCardDocument } from "@trenova/graphql/generated/documents/ifta-return/desk";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

/** One IFTA return as the Desk card draws it: the quarter, its totals, its problems and every line. */
export type DeskIftaReturnCard = DeskIftaReturnCardQuery["iftaReturn"];
export type DeskIftaReturnLine = DeskIftaReturnCard["lines"][number];
export type DeskIftaReturnProblem = DeskIftaReturnCard["problems"][number];

export async function fetchDeskIftaReturnCard(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<DeskIftaReturnCard> {
  const data = await requestGraphQL({
    document: DeskIftaReturnCardDocument,
    operationName: "DeskIftaReturnCard",
    variables: { id },
    signal: options?.signal,
  });
  return data.iftaReturn;
}
