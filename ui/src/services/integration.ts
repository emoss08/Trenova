import { http } from "@/lib/http-client";
import type {
  Integration,
  IntegrationType,
} from "@/types/integration";
import type { LimitOffsetResponse } from "@/types/server";

export async function getIntegrations() {
  const response =
    await http.get<LimitOffsetResponse<Integration>>("/integrations/");
  return response.data;
}

export async function getIntegrationByType(type: IntegrationType) {
  const response = await http.get<Integration>(`/integrations/${type}`);
  return response.data;
}
