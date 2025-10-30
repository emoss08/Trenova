import { http } from "@/lib/http-client";
import { type AuditEntry, type AuditEntryResponse } from "@/types/audit-entry";

export class AuditEntryAPI {
  async getAuditEntriesByResourceID(resourceId: AuditEntry["resourceId"]) {
    const response = await http.get<AuditEntryResponse>(
      `/audit-entries/resource/${resourceId}`,
    );
    return response.data;
  }
}
