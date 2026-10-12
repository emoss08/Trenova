import { DeskOrderCardDocument, type DeskOrderCardQuery } from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type DeskOrder = NonNullable<DeskOrderCardQuery["order"]>;
export type DeskOrderCharge = DeskOrder["charges"][number];

/**
 * One order as the Desk's card reads it, without its legs, which the card
 * reads a page at a time. An order the server does not return is an error
 * rather than an empty card, so the card says it could not be read.
 */
export async function fetchDeskOrder(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<DeskOrder> {
  const data = await requestGraphQL({
    document: DeskOrderCardDocument,
    operationName: "DeskOrderCard",
    variables: { id },
    signal: options?.signal,
  });
  if (!data.order) {
    throw new Error("Order not found");
  }
  return data.order;
}
