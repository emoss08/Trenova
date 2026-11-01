import {
  EvaluateShipmentServiceFailuresDocument,
  ResolveServiceFailureDocument,
  ReviewServiceFailureDocument,
  ServiceFailureEdi214PayloadDocument,
  ServiceFailureEdi214ReadinessDocument,
  ServiceFailureEdi214StatusDocument,
  ServiceFailureTableDocument,
  UpdateServiceFailureDocument,
  VoidServiceFailureDocument,
  type ServiceFailureEdiTrigger,
  type ServiceFailureLifecycleInput,
  type UpdateServiceFailureInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

type RequestOptions = { signal?: AbortSignal };

export type ServiceFailureLifecycleAction = "review" | "resolve" | "void";

export async function listShipmentServiceFailuresGraphQL(
  shipmentId: string,
  first: number,
  options?: RequestOptions,
) {
  const data = await requestGraphQL({
    document: ServiceFailureTableDocument,
    operationName: "ServiceFailureTable",
    variables: { input: { first }, shipmentId, includeTotalCount: true },
    signal: options?.signal,
  });
  return {
    results: data.serviceFailures.edges.map((edge) => edge.node),
    count: data.serviceFailures.totalCount ?? data.serviceFailures.edges.length,
  };
}

export async function evaluateShipmentServiceFailuresGraphQL(shipmentId: string, force = false) {
  const data = await requestGraphQL({
    document: EvaluateShipmentServiceFailuresDocument,
    operationName: "EvaluateShipmentServiceFailures",
    variables: { shipmentId, force },
  });
  return data.evaluateShipmentServiceFailures;
}

export async function updateServiceFailureGraphQL(id: string, input: UpdateServiceFailureInput) {
  const data = await requestGraphQL({
    document: UpdateServiceFailureDocument,
    operationName: "UpdateServiceFailure",
    variables: { id, input },
  });
  return data.updateServiceFailure;
}

export async function changeServiceFailureLifecycleGraphQL(
  action: ServiceFailureLifecycleAction,
  id: string,
  input: ServiceFailureLifecycleInput,
) {
  switch (action) {
    case "review": {
      const data = await requestGraphQL({
        document: ReviewServiceFailureDocument,
        operationName: "ReviewServiceFailure",
        variables: { id, input },
      });
      return data.reviewServiceFailure;
    }
    case "resolve": {
      const data = await requestGraphQL({
        document: ResolveServiceFailureDocument,
        operationName: "ResolveServiceFailure",
        variables: { id, input },
      });
      return data.resolveServiceFailure;
    }
    case "void": {
      const data = await requestGraphQL({
        document: VoidServiceFailureDocument,
        operationName: "VoidServiceFailure",
        variables: { id, input },
      });
      return data.voidServiceFailure;
    }
  }
}

export async function getServiceFailureEdi214PayloadGraphQL(id: string, options?: RequestOptions) {
  const data = await requestGraphQL({
    document: ServiceFailureEdi214PayloadDocument,
    operationName: "ServiceFailureEdi214Payload",
    variables: { id },
    signal: options?.signal,
  });
  return data.serviceFailureEdi214Payload;
}

export async function getServiceFailureEdi214ReadinessGraphQL(
  id: string,
  trigger?: ServiceFailureEdiTrigger,
  options?: RequestOptions,
) {
  const data = await requestGraphQL({
    document: ServiceFailureEdi214ReadinessDocument,
    operationName: "ServiceFailureEdi214Readiness",
    variables: { id, trigger: trigger ?? null },
    signal: options?.signal,
  });
  return data.serviceFailureEdi214Readiness;
}

export async function getServiceFailureEdi214StatusGraphQL(id: string, options?: RequestOptions) {
  const data = await requestGraphQL({
    document: ServiceFailureEdi214StatusDocument,
    operationName: "ServiceFailureEdi214Status",
    variables: { id },
    signal: options?.signal,
  });
  return data.serviceFailureEdi214Status;
}
