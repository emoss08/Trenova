import { describe, expect, it } from "vitest";
import { documentContentSchema } from "@trenova/shared/types/document";

/**
 * The extraction worker records the model's reading in
 * `structuredData.aiDiagnostics.acceptanceStatus` with one of four values
 * (`documentanalysis/types.go`): `not_attempted`, `pending` while the
 * model is still reading a document whose text is already in, `accepted` and
 * `rejected`. A value the parse refuses throws away the whole content, its
 * text included, so every one of them has to parse.
 */
describe("documentContentSchema", () => {
  const base = {
    id: "dcon_1",
    documentId: "doc_1",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    status: "Indexed",
    contentText: "RATE CONFIRMATION\nLoad 88213",
    pageCount: 1,
    sourceKind: "native",
    detectedLanguage: "en",
    detectedDocumentKind: "RateConfirmation",
    classificationConfidence: 0.92,
    failureCode: "",
    failureMessage: "",
    version: 2,
    createdAt: 1_792_000_000,
    updatedAt: 1_792_000_030,
    lastExtractedAt: 1_792_000_030,
    converter: "docling",
    converterVersion: "2.31.0",
  };

  it.each(["not_attempted", "pending", "accepted", "rejected"])(
    "keeps the text of a document whose model reading is %s",
    (acceptanceStatus) => {
      const content = documentContentSchema.parse({
        ...base,
        structuredData: { schemaVersion: 1, aiDiagnostics: { acceptanceStatus } },
      });

      expect(content.contentText).toBe("RATE CONFIRMATION\nLoad 88213");
      expect(content.structuredData.aiDiagnostics?.acceptanceStatus).toBe(acceptanceStatus);
    },
  );
});
