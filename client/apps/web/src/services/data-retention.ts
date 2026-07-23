import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import { dataRetentionSchema, type UpdateDataRetentionRequest } from "@/types/data-retention";

export async function getDataRetention() {
  const response = await api.get(`/data-retention/`);
  return safeParse(dataRetentionSchema, response, "DataRetention");
}

export async function updateDataRetention(request: UpdateDataRetentionRequest) {
  const response = await api.put(`/data-retention/`, request);
  return safeParse(dataRetentionSchema, response, "DataRetention");
}
