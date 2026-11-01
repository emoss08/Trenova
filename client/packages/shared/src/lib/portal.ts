import { api } from "@trenova/shared/lib/api";
import type { PortalDocument } from "@trenova/shared/lib/graphql/driver-portal";

export async function uploadMyLoadDocument(
  shipmentId: string,
  file: File,
  documentTypeId?: string,
) {
  const formData = new FormData();
  formData.append("file", file);
  if (documentTypeId) {
    formData.append("documentTypeId", documentTypeId);
  }
  return api.upload<PortalDocument>(
    `/portal/loads/${encodeURIComponent(shipmentId)}/documents/`,
    formData,
  );
}

export async function uploadMyProfileDocument(file: File, documentTypeId?: string) {
  const formData = new FormData();
  formData.append("file", file);
  if (documentTypeId) {
    formData.append("documentTypeId", documentTypeId);
  }
  return api.upload<PortalDocument>("/portal/profile/documents/", formData);
}

export async function uploadMyCredentialDocument(credentialId: string, file: File) {
  const formData = new FormData();
  formData.append("file", file);
  return api.upload<PortalDocument>(
    `/portal/credentials/${encodeURIComponent(credentialId)}/document/`,
    formData,
  );
}

export async function uploadMyExpenseReceipt(expenseId: string, file: File) {
  const formData = new FormData();
  formData.append("file", file);
  return api.upload<{ id: string; receiptDocumentId: string | null }>(
    `/portal/expenses/${encodeURIComponent(expenseId)}/receipt/`,
    formData,
  );
}
