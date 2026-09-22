import { describe, expect, it } from "vitest";
import { suggestNextStep, type NextStepMessage } from "../next-step";

function message(overrides: Partial<NextStepMessage>): NextStepMessage {
  return {
    status: "InReview",
    classification: null,
    failureText: "",
    matchedShipment: null,
    attachments: [],
    ...overrides,
  };
}

const pdf = { id: "att1", documentId: "doc_1", failureText: "", fileName: "tender.pdf" };
const refused = {
  id: "att2",
  documentId: null,
  failureText: "Not an accepted file type",
  fileName: "x.exe",
};
const shipment = { id: "shp_1", proNumber: "PRO-9" };

describe("suggestNextStep", () => {
  it("says nothing about a message nobody has to act on", () => {
    expect(suggestNextStep(message({ status: "Actioned", classification: "Tender" }))).toEqual({
      kind: "none",
    });
    expect(suggestNextStep(message({ status: "Ignored" }))).toEqual({ kind: "none" });
  });

  it("asks a person to file a message the pipeline could not read", () => {
    expect(
      suggestNextStep(message({ status: "Quarantined", failureText: "Could not parse" })),
    ).toEqual({ kind: "linkByHand" });
  });

  it("offers the drafted shipment for a tender that came with a readable document", () => {
    expect(
      suggestNextStep(message({ classification: "Tender", attachments: [refused, pdf] })),
    ).toEqual({ kind: "reviewDraft", documentId: "doc_1", fileName: "tender.pdf" });
  });

  it("hands a tender with nothing readable attached to the desk", () => {
    expect(suggestNextStep(message({ classification: "Tender", attachments: [refused] }))).toEqual({
      kind: "askDesk",
      reason: "buildLoad",
    });
  });

  it("files paperwork against the shipment it was matched to", () => {
    for (const classification of ["ProofOfDelivery", "RateConfirmation", "Invoice"] as const) {
      expect(
        suggestNextStep(message({ classification, matchedShipment: shipment, attachments: [pdf] })),
      ).toEqual({
        kind: "attachToShipment",
        documentId: "doc_1",
        fileName: "tender.pdf",
        shipmentId: "shp_1",
        proNumber: "PRO-9",
      });
    }
  });

  it("asks which shipment unmatched paperwork belongs to", () => {
    expect(
      suggestNextStep(message({ classification: "ProofOfDelivery", attachments: [pdf] })),
    ).toEqual({ kind: "linkByHand" });
  });

  it("has the desk draft an answer to a status request", () => {
    expect(
      suggestNextStep(message({ classification: "StatusRequest", matchedShipment: shipment })),
    ).toEqual({ kind: "askDesk", reason: "answerStatus" });
  });

  it("opens the shipment behind a detention dispute, or asks which one it is", () => {
    expect(
      suggestNextStep(message({ classification: "DetentionDispute", matchedShipment: shipment })),
    ).toEqual({ kind: "openShipment", shipmentId: "shp_1", proNumber: "PRO-9" });
    expect(suggestNextStep(message({ classification: "DetentionDispute" }))).toEqual({
      kind: "linkByHand",
    });
  });

  it("lets a person decide what an unreadable or other message is", () => {
    expect(suggestNextStep(message({ classification: "Other" }))).toEqual({ kind: "none" });
    expect(suggestNextStep(message({ classification: null }))).toEqual({ kind: "linkByHand" });
  });
});
