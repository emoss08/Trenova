import {
  RequestSupportLogsDocument,
  SupportLogRequestsDocument,
} from "@trenova/graphql/generated/documents/support-logs/support-logs";
import type {
  SupportLogPeriod,
  SupportLogRequestsQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type SupportLogs = SupportLogRequestsQuery["supportLogRequests"];
export type SupportLogRequest = SupportLogs["requests"][number];
export type SupportLogStatus = SupportLogRequest["status"];
export type { SupportLogPeriod };

/**
 * This organization's recent requests to send the instance's logs to Trenova support, whether one
 * is in flight anywhere on the instance, and how many more it may make today.
 */
export async function fetchSupportLogs(options?: { signal?: AbortSignal }): Promise<SupportLogs> {
  const data = await requestGraphQL({
    document: SupportLogRequestsDocument,
    operationName: "SupportLogRequests",
    signal: options?.signal,
  });
  return data.supportLogRequests;
}

/** Asks for the instance's logs over the period to be sent to Trenova support. */
export async function requestSupportLogs(period: SupportLogPeriod): Promise<void> {
  await requestGraphQL({
    document: RequestSupportLogsDocument,
    operationName: "RequestSupportLogs",
    variables: { input: { period } },
  });
}
