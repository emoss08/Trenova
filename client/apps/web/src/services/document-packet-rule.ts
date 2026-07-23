import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import { documentPacketRuleSchema, type DocumentPacketRule } from "@/types/document-packet-rule";
import type { GenericLimitOffsetResponse } from "@trenova/shared/types/server";

export class DocumentPacketRuleService {
  async getAll(params?: string) {
    const response = await api.get<GenericLimitOffsetResponse<DocumentPacketRule>>(
      `/document-packet-rules/${params ? `?${params}` : ""}`,
    );
    return safeParse(documentPacketRuleSchema.array(), response.results, "DocumentPacketRules");
  }

  async getById(id: string) {
    const response = await api.get<DocumentPacketRule>(`/document-packet-rules/${id}/`);
    return safeParse(documentPacketRuleSchema, response, "DocumentPacketRule");
  }

  async create(data: DocumentPacketRule) {
    const response = await api.post<DocumentPacketRule>("/document-packet-rules/", data);
    return safeParse(documentPacketRuleSchema, response, "DocumentPacketRule");
  }

  async update(id: string, data: DocumentPacketRule) {
    const response = await api.put<DocumentPacketRule>(`/document-packet-rules/${id}/`, data);
    return safeParse(documentPacketRuleSchema, response, "DocumentPacketRule");
  }

  async delete(id: string) {
    await api.delete(`/document-packet-rules/${id}/`);
  }
}
