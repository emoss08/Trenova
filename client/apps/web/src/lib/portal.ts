import { api } from "@/lib/api";

export type PortalLoadDocument = {
  id: string;
  fileName: string;
  fileSize: number;
  status: string;
  documentTypeName: string;
  createdAt: number;
};

export type PortalShipmentDocumentType = {
  id: string;
  code: string;
  name: string;
  color: string;
};

export async function fetchPortalShipmentDocumentTypes() {
  return api.get<PortalShipmentDocumentType[]>("/portal/document-types/");
}

export async function fetchMyLoadDocuments(shipmentId: string) {
  return api.get<PortalLoadDocument[]>(
    `/portal/loads/${encodeURIComponent(shipmentId)}/documents/`,
  );
}

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
  return api.upload<PortalLoadDocument>(
    `/portal/loads/${encodeURIComponent(shipmentId)}/documents/`,
    formData,
  );
}

export async function fetchPortalWorkerDocumentTypes() {
  return api.get<PortalShipmentDocumentType[]>("/portal/profile/document-types/");
}

export async function fetchMyProfileDocuments() {
  return api.get<PortalLoadDocument[]>("/portal/profile/documents/");
}

export async function uploadMyProfileDocument(file: File, documentTypeId?: string) {
  const formData = new FormData();
  formData.append("file", file);
  if (documentTypeId) {
    formData.append("documentTypeId", documentTypeId);
  }
  return api.upload<PortalLoadDocument>("/portal/profile/documents/", formData);
}

export async function uploadMyExpenseReceipt(expenseId: string, file: File) {
  const formData = new FormData();
  formData.append("file", file);
  return api.upload<{ id: string; receiptDocumentId: string | null }>(
    `/portal/expenses/${encodeURIComponent(expenseId)}/receipt/`,
    formData,
  );
}
