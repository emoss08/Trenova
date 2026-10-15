import {
  AcknowledgeFactoringNoaDocument,
  DraftFactoringNoaDocument,
  FactoringNoaFieldsFragmentDoc,
  FactoringNoaOwnerDocument,
  FactoringNoasDocument,
  ReleaseFactoringNoaDocument,
  SendFactoringNoaDocument,
  type AcknowledgeFactoringNoaInput,
  type DraftFactoringNoaInput,
  type FactoringNoaFieldsFragment,
  type FactoringNoaStatus,
  type ReleaseFactoringNoaInput,
  type SendFactoringNoaInput,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData } from "@trenova/graphql/fragment-data";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

type RequestOptions = { signal?: AbortSignal };

export type FactoringNoa = FactoringNoaFieldsFragment;

export type FactoringNoaFilter = {
  factoringCompanyId?: string;
  customerId?: string;
  statuses?: FactoringNoaStatus[];
};

/** The most notices one factor or customer lists at once; a factor has one per customer. */
export const FACTORING_NOA_PAGE_SIZE = 100;

export async function listFactoringNoas(
  filter: FactoringNoaFilter,
  options?: RequestOptions,
): Promise<FactoringNoa[]> {
  const data = await requestGraphQL({
    document: FactoringNoasDocument,
    operationName: "FactoringNoas",
    variables: {
      input: {
        table: {
          first: FACTORING_NOA_PAGE_SIZE,
          sort: [{ field: "createdAt", direction: "desc" }],
        },
        factoringCompanyId: filter.factoringCompanyId ?? null,
        customerId: filter.customerId ?? null,
        statuses: filter.statuses ?? null,
      },
    },
    signal: options?.signal,
  });
  return data.factoringNoas.edges.map((edge) =>
    getFragmentData(FactoringNoaFieldsFragmentDoc, edge.node),
  );
}

export async function fetchFactoringNoaOwner(
  id: string,
  options?: RequestOptions,
): Promise<string> {
  const data = await requestGraphQL({
    document: FactoringNoaOwnerDocument,
    operationName: "FactoringNoaOwner",
    variables: { id },
    signal: options?.signal,
  });
  return data.factoringNoa.factoringCompanyId;
}

export async function draftFactoringNoa(input: DraftFactoringNoaInput): Promise<FactoringNoa> {
  const data = await requestGraphQL({
    document: DraftFactoringNoaDocument,
    operationName: "DraftFactoringNoa",
    variables: { input },
  });
  return getFragmentData(FactoringNoaFieldsFragmentDoc, data.draftFactoringNoa);
}

export async function sendFactoringNoa(input: SendFactoringNoaInput): Promise<FactoringNoa> {
  const data = await requestGraphQL({
    document: SendFactoringNoaDocument,
    operationName: "SendFactoringNoa",
    variables: { input },
  });
  return getFragmentData(FactoringNoaFieldsFragmentDoc, data.sendFactoringNoa);
}

export async function acknowledgeFactoringNoa(
  input: AcknowledgeFactoringNoaInput,
): Promise<FactoringNoa> {
  const data = await requestGraphQL({
    document: AcknowledgeFactoringNoaDocument,
    operationName: "AcknowledgeFactoringNoa",
    variables: { input },
  });
  return getFragmentData(FactoringNoaFieldsFragmentDoc, data.acknowledgeFactoringNoa);
}

export async function releaseFactoringNoa(input: ReleaseFactoringNoaInput): Promise<FactoringNoa> {
  const data = await requestGraphQL({
    document: ReleaseFactoringNoaDocument,
    operationName: "ReleaseFactoringNoa",
    variables: { input },
  });
  return getFragmentData(FactoringNoaFieldsFragmentDoc, data.releaseFactoringNoa);
}
