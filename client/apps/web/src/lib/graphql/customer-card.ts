import {
  DeskCustomerCardDocument,
  DeskCustomerRateAgreementsDocument,
  type DeskCustomerCardQuery,
  type DeskCustomerRateAgreementsQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type DeskCustomer = NonNullable<DeskCustomerCardQuery["customer"]>;
export type DeskCustomerBilling = NonNullable<DeskCustomer["billingProfile"]>;
export type DeskCustomerRateAgreement =
  DeskCustomerRateAgreementsQuery["rateAgreements"]["edges"][number]["node"];

/** A customer's latest records of one kind, newest first, with how many there are in all. */
export type CustomerRecent<T> = { rows: T[]; total: number };

/**
 * One customer as the Desk's card reads it. A customer the server does not
 * return is an error rather than an empty card, so the card says it could not
 * be read.
 */
export async function fetchDeskCustomer(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<DeskCustomer> {
  const data = await requestGraphQL({
    document: DeskCustomerCardDocument,
    operationName: "DeskCustomerCard",
    variables: { id },
    signal: options?.signal,
  });
  if (!data.customer) {
    throw new Error("Customer not found");
  }
  return data.customer;
}

/** The rate agreements written for a customer, newest first. */
export async function fetchCustomerRateAgreements(
  customerId: string,
  limit: number,
  options?: { signal?: AbortSignal },
): Promise<CustomerRecent<DeskCustomerRateAgreement>> {
  const data = await requestGraphQL({
    document: DeskCustomerRateAgreementsDocument,
    operationName: "DeskCustomerRateAgreements",
    variables: {
      input: {
        first: limit,
        fieldFilters: [{ field: "customerId", operator: "eq", value: customerId }],
        sort: [{ field: "effectiveFrom", direction: "desc" }],
      },
    },
    signal: options?.signal,
  });
  const rows = (data.rateAgreements.edges ?? []).map((edge) => edge.node);
  return { rows, total: data.rateAgreements.totalCount ?? rows.length };
}
