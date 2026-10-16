import { documentSchema } from "@trenova/shared/types/document";
import { invalidationFor } from "@trenova/shared/hooks/realtime-patching";
import { describe, expect, it } from "vitest";
import {
  DOCUMENT_SIGNATURE_PHASE,
  documentNeedsSignature,
  documentSignatureFormSchema,
  documentSignatureFormValues,
  documentSignatureSetByPerson,
} from "../document-signature";

/*
Fixtures follow the Go document JSON: signatureStatus is always sent
(Unknown by default), signedAt is null unless Signed, signatureSource is ""
until extraction or a person answers, and documentType carries
requiresSignature when the request asked for the type.
*/
function serverDocument(overrides: Record<string, unknown> = {}) {
  return {
    id: "doc_1",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    lineageId: "doc_1",
    versionNumber: 1,
    isCurrentVersion: true,
    fileName: "pod.pdf",
    originalName: "pod.pdf",
    fileSize: 2048,
    fileType: "application/pdf",
    storagePath: "org_1/bu_1/pod.pdf",
    status: "Active",
    resourceId: "shp_1",
    resourceType: "shipment",
    processingProfile: "inbound_attachment",
    isPublic: false,
    uploadedById: "usr_1",
    previewStatus: "Ready",
    contentStatus: "Indexed",
    hasExtractedText: true,
    shipmentDraftStatus: "Unavailable",
    documentTypeId: "dt_pod",
    documentType: {
      id: "dt_pod",
      code: "POD",
      name: "Proof of delivery",
      documentClassification: "Public",
      documentCategory: "Shipment",
      requiresSignature: true,
    },
    signatureStatus: "Unknown",
    signedAt: null,
    signatureSource: "",
    signatureUpdatedById: "",
    signatureUpdatedAt: null,
    version: 3,
    createdAt: 1_792_000_000,
    updatedAt: 1_792_000_000,
    ...overrides,
  };
}

describe("document signature on the wire", () => {
  it("reads a document nobody has answered for as unknown with no source", () => {
    const doc = documentSchema.parse(serverDocument());

    expect(doc.signatureStatus).toBe("Unknown");
    expect(doc.signatureSource).toBeNull();
    expect(documentSignatureSetByPerson(doc)).toBe(false);
  });

  it("reads a document from before the column as unknown", () => {
    const { signatureStatus: _status, signatureSource: _source, ...older } = serverDocument();
    const doc = documentSchema.parse(older);

    expect(doc.signatureStatus).toBe("Unknown");
    expect(doc.signatureSource).toBeNull();
  });

  it("keeps a person's answer and its date", () => {
    const doc = documentSchema.parse(
      serverDocument({
        signatureStatus: "Signed",
        signedAt: 1_791_950_000,
        signatureSource: "Person",
        signatureUpdatedById: "usr_2",
      }),
    );

    expect(doc.signatureStatus).toBe("Signed");
    expect(doc.signedAt).toBe(1_791_950_000);
    expect(documentSignatureSetByPerson(doc)).toBe(true);
  });
});

describe("documentNeedsSignature", () => {
  it("asks only about the current version of a type that needs a signature", () => {
    expect(documentNeedsSignature(documentSchema.parse(serverDocument()))).toBe(true);
    expect(
      documentNeedsSignature(documentSchema.parse(serverDocument({ isCurrentVersion: false }))),
    ).toBe(false);
    expect(
      documentNeedsSignature(
        documentSchema.parse(
          serverDocument({
            documentType: { ...serverDocument().documentType, requiresSignature: false },
          }),
        ),
      ),
    ).toBe(false);
    expect(
      documentNeedsSignature(documentSchema.parse(serverDocument({ documentType: null }))),
    ).toBe(false);
  });
});

describe("signature phases", () => {
  it("puts an unsigned POD in front of a person and a signed one out of the way", () => {
    expect(DOCUMENT_SIGNATURE_PHASE).toEqual({
      Signed: "complete",
      Unsigned: "attention",
      Unknown: "awaiting",
    });
  });
});

describe("documentSignatureFormSchema", () => {
  it("accepts a signed document with or without a date", () => {
    expect(
      documentSignatureFormSchema.safeParse({ status: "Signed", signedAt: 1_791_950_000 }).success,
    ).toBe(true);
    expect(
      documentSignatureFormSchema.safeParse({ status: "Signed", signedAt: null }).success,
    ).toBe(true);
  });

  it("refuses a signing date on anything but a signed document", () => {
    const result = documentSignatureFormSchema.safeParse({
      status: "Unsigned",
      signedAt: 1_791_950_000,
    });

    expect(result.success).toBe(false);
    expect(result.error?.issues[0]?.path).toEqual(["signedAt"]);
  });

  it("starts from the document's answer and drops a stray date", () => {
    expect(
      documentSignatureFormValues({ signatureStatus: "Signed", signedAt: 1_791_950_000 }),
    ).toEqual({ status: "Signed", signedAt: 1_791_950_000 });
    expect(
      documentSignatureFormValues({ signatureStatus: "Unsigned", signedAt: 1_791_950_000 }),
    ).toEqual({ status: "Unsigned", signedAt: null });
  });
});

describe("document realtime invalidation", () => {
  it("refreshes document lists and billing readiness when a document changes", () => {
    const invalidation = invalidationFor({
      organizationId: "org_1",
      businessUnitId: "bu_1",
      resource: "document",
      action: "update",
      recordId: "doc_1",
      entity: { resourceId: "shp_1", resourceType: "shipment" },
    });

    expect(invalidation.roots).toEqual(["documents", ["shipment", "billingReadiness"]]);
  });
});
