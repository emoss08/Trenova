/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
