import {
  ApplyUnappliedCustomerPaymentDocument,
  CustomerPaymentDetailDocument,
  CustomerPaymentTableDocument,
  PostAndApplyCustomerPaymentDocument,
  ReverseCustomerPaymentDocument,
  type ApplyCustomerPaymentInput,
  type CustomerPaymentDetailQuery,
  type CustomerPaymentTableQuery,
  type CustomerPaymentTableQueryVariables,
  type PostCustomerPaymentInput,
  type ReverseCustomerPaymentInput,
} from "@/graphql/generated/graphql";
import { requestGraphQL } from "@/lib/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";

export type CustomerPaymentRow = NonNullable<
  CustomerPaymentTableQuery["customerPayments"]["edges"]
>[number]["node"];
export type CustomerPaymentDetail = NonNullable<
  CustomerPaymentDetailQuery["customerPayment"]
>;
export type CustomerPaymentDetailApplication = NonNullable<
  CustomerPaymentDetail["applications"]
>[number];

export const customerPaymentTableGraphQLConfig = defineDataTableGraphQLConfig<
  CustomerPaymentRow,
  CustomerPaymentTableQueryVariables
>({
  document: CustomerPaymentTableDocument,
  operationName: "CustomerPaymentTable",
  connectionKey: "customerPayments",
});

export async function fetchCustomerPaymentDetail(id: string) {
  const data = await requestGraphQL({
    document: CustomerPaymentDetailDocument,
    operationName: "CustomerPaymentDetail",
    variables: { id },
  });
  return data.customerPayment;
}

export async function postAndApplyCustomerPayment(input: PostCustomerPaymentInput) {
  const data = await requestGraphQL({
    document: PostAndApplyCustomerPaymentDocument,
    operationName: "PostAndApplyCustomerPayment",
    variables: { input },
  });
  return data.postAndApplyCustomerPayment;
}

export async function applyUnappliedCustomerPayment(input: ApplyCustomerPaymentInput) {
  const data = await requestGraphQL({
    document: ApplyUnappliedCustomerPaymentDocument,
    operationName: "ApplyUnappliedCustomerPayment",
    variables: { input },
  });
  return data.applyUnappliedCustomerPayment;
}

export async function reverseCustomerPayment(input: ReverseCustomerPaymentInput) {
  const data = await requestGraphQL({
    document: ReverseCustomerPaymentDocument,
    operationName: "ReverseCustomerPayment",
    variables: { input },
  });
  return data.reverseCustomerPayment;
}
