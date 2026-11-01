import {
  CreateDocumentPacketRuleDocument,
  DeleteDocumentPacketRuleDocument,
  UpdateDocumentPacketRuleDocument,
  type DocumentPacketRuleInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { DocumentPacketRule } from "@/types/document-packet-rule";

export function toDocumentPacketRuleInput(data: DocumentPacketRule): DocumentPacketRuleInput {
  return {
    resourceType: data.resourceType,
    documentTypeId: data.documentTypeId,
    required: data.required,
    allowMultiple: data.allowMultiple,
    displayOrder: data.displayOrder,
    expirationRequired: data.expirationRequired,
    expirationWarningDays: data.expirationWarningDays,
    version: data.version ?? undefined,
  };
}

export async function createDocumentPacketRule(data: DocumentPacketRule) {
  const response = await requestGraphQL({
    document: CreateDocumentPacketRuleDocument,
    operationName: "CreateDocumentPacketRule",
    variables: { input: toDocumentPacketRuleInput(data) },
  });

  return response.createDocumentPacketRule;
}

export async function updateDocumentPacketRule(id: string, data: DocumentPacketRule) {
  const response = await requestGraphQL({
    document: UpdateDocumentPacketRuleDocument,
    operationName: "UpdateDocumentPacketRule",
    variables: { id, input: toDocumentPacketRuleInput(data) },
  });

  return response.updateDocumentPacketRule;
}

export async function deleteDocumentPacketRule(id: string): Promise<void> {
  await requestGraphQL({
    document: DeleteDocumentPacketRuleDocument,
    operationName: "DeleteDocumentPacketRule",
    variables: { id },
  });
}
