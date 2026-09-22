import type { InboundClassification, InboundMessageStatus } from "@/lib/graphql/inbox";

/**
 * The one thing most likely to be done next with a message.
 *
 * It is worked out from what the message is and what it was matched to, not
 * asked of a model: a suggestion that changes between two looks at the same
 * message is one nobody can learn to trust. The reading pane leads with it,
 * and every other action stays one key away.
 */
export type NextStep =
  | { kind: "none" }
  | { kind: "linkByHand" }
  | { kind: "reviewDraft"; documentId: string; fileName: string }
  | {
      kind: "attachToShipment";
      documentId: string;
      fileName: string;
      shipmentId: string;
      proNumber: string;
    }
  | { kind: "openShipment"; shipmentId: string; proNumber: string }
  | { kind: "askDesk"; reason: "buildLoad" | "answerStatus" };

export type NextStepMessage = {
  status: InboundMessageStatus;
  classification?: InboundClassification | null;
  failureText: string;
  matchedShipment?: { id: string; proNumber: string } | null;
  attachments: { id: string; documentId?: string | null; failureText: string; fileName: string }[];
};

const PAPERWORK: ReadonlySet<InboundClassification> = new Set([
  "ProofOfDelivery",
  "RateConfirmation",
  "Invoice",
]);

function readableDocument(message: NextStepMessage) {
  for (const attachment of message.attachments) {
    if (attachment.failureText === "" && attachment.documentId) {
      return { documentId: attachment.documentId, fileName: attachment.fileName };
    }
  }

  return null;
}

export function suggestNextStep(message: NextStepMessage): NextStep {
  if (message.status !== "InReview" && message.status !== "Quarantined") {
    return { kind: "none" };
  }
  if (message.status === "Quarantined" || message.failureText !== "") {
    return { kind: "linkByHand" };
  }

  const classification = message.classification ?? null;
  const shipment = message.matchedShipment ?? null;
  const document = readableDocument(message);

  if (classification === null) {
    return { kind: "linkByHand" };
  }

  if (classification === "Tender") {
    return document === null
      ? { kind: "askDesk", reason: "buildLoad" }
      : { kind: "reviewDraft", ...document };
  }

  if (PAPERWORK.has(classification)) {
    if (shipment === null) {
      return { kind: "linkByHand" };
    }
    if (document !== null) {
      return {
        kind: "attachToShipment",
        ...document,
        shipmentId: shipment.id,
        proNumber: shipment.proNumber,
      };
    }

    return { kind: "openShipment", shipmentId: shipment.id, proNumber: shipment.proNumber };
  }

  if (classification === "StatusRequest") {
    return { kind: "askDesk", reason: "answerStatus" };
  }

  if (classification === "DetentionDispute") {
    return shipment === null
      ? { kind: "linkByHand" }
      : { kind: "openShipment", shipmentId: shipment.id, proNumber: shipment.proNumber };
  }

  return { kind: "none" };
}
