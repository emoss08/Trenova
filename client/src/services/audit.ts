import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  listByResourceIdSchema,
  type ListByResourceIdResponse,
} from "@/types/audit-entry";

export class AuditService {
  public async listByResourceId(
    resourceId: string,
    { limit, offset }: { limit: number; offset: number },
  ) {
    const response = await api.get<ListByResourceIdResponse>(
      `/audit-entries/resource/${resourceId}/?limit=${limit}&offset=${offset}`,
    );

    return safeParse(listByResourceIdSchema, response, "Audit Entry");
  }
}
