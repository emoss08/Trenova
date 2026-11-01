import {
  CreateDocumentTypeDocument,
  UpdateDocumentTypeDocument,
  type DocumentTypeInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { DocumentType } from "@trenova/shared/types/document-type";

export function toDocumentTypeInput(values: DocumentType): DocumentTypeInput {
  return {
    code: values.code,
    name: values.name,
    description: values.description || null,
    color: values.color ?? null,
    documentClassification: values.documentClassification,
    documentCategory: values.documentCategory,
    version: values.version,
  };
}

export async function createDocumentType(values: DocumentType) {
  const data = await requestGraphQL({
    document: CreateDocumentTypeDocument,
    operationName: "CreateDocumentType",
    variables: { input: toDocumentTypeInput(values) },
  });

  return data.createDocumentType;
}

export async function updateDocumentType(id: string, values: DocumentType) {
  const data = await requestGraphQL({
    document: UpdateDocumentTypeDocument,
    operationName: "UpdateDocumentType",
    variables: { id, input: toDocumentTypeInput(values) },
  });

  return data.updateDocumentType;
}
