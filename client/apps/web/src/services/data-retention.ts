import {
  DataRetentionDocument,
  UpdateDataRetentionDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import { dataRetentionSchema, type UpdateDataRetentionRequest } from "@/types/data-retention";

export async function getDataRetention(options?: { signal?: AbortSignal }) {
  const response = await requestGraphQL({
    document: DataRetentionDocument,
    operationName: "DataRetention",
    signal: options?.signal,
  });
  return safeParse(dataRetentionSchema, response.dataRetention, "DataRetention");
}

export async function updateDataRetention(request: UpdateDataRetentionRequest) {
  const response = await requestGraphQL({
    document: UpdateDataRetentionDocument,
    operationName: "UpdateDataRetention",
    variables: { input: request },
  });
  return safeParse(dataRetentionSchema, response.updateDataRetention, "DataRetention");
}
