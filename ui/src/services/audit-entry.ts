import { http } from "@/lib/http-client";
import { type AuditEntry } from "@/types/audit-entry";

type AuditEntryResponse = {
  items: AuditEntry[];
  total: number;
};

export async function getAuditEntriesByResourceID(
  resourceId: AuditEntry["resourceId"],
) {
  const response = await http.get<AuditEntryResponse>(
    `/audit-logs/resource/${resourceId}`,
  );
  return response.data;
}
