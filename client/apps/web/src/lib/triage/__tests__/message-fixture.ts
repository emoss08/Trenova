import type { InboundMessageRow, TriageAction } from "@/lib/graphql/inbox";

/**
 * A list row shaped as `InboundMessageListFields` returns it, every field
 * present: a waiting tender with nothing parked, nobody's, no actions.
 */
export function messageRow(overrides: Partial<InboundMessageRow> = {}): InboundMessageRow {
  return {
    id: "imsg_1",
    mailboxId: "imbx_1",
    fromAddress: "dispatch@northwind.example",
    fromName: "Northwind Foods Dispatch",
    subject: "New load",
    receivedAt: 1_000_000,
    classification: "Tender",
    confidence: 0.9,
    status: "InReview",
    matchedShipmentId: null,
    matchedCustomerId: null,
    matchReason: "",
    needsReview: true,
    failureText: "",
    preview: "Can you take a load",
    attachmentCount: 0,
    matchedShipment: null,
    matchedCustomer: null,
    threadKey: "thread-1",
    threadCount: 1,
    senderDomain: "northwind.example",
    ownerId: null,
    owner: null,
    dueAt: 1_007_200,
    responseTargetMinutes: 120,
    snooze: null,
    awaitingReply: null,
    novaRead: true,
    handledByNova: false,
    actions: [],
    version: 1,
    ...overrides,
  };
}

export function action(overrides: Partial<TriageAction> = {}): TriageAction {
  return {
    id: "link",
    kind: "Link",
    done: false,
    novaCanDo: false,
    attachment: null,
    shipment: null,
    replyPurpose: null,
    refusedFileName: null,
    reply: null,
    loadDrafted: false,
    createdShipment: null,
    stopsDecision: null,
    exceptionKind: null,
    completedAt: null,
    ...overrides,
  };
}

export const SHIPMENT = {
  id: "shp_1",
  proNumber: "S-104198",
  bol: "BOL 76902",
  poNumber: "PO 88213",
  originCity: "Dallas",
  originState: "TX",
  destinationCity: "Memphis",
  destinationState: "TN",
};
