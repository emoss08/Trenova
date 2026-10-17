import type { DocumentAsk } from "@/stores/desk-store";
import type { AssistantProposal } from "@/types/assistant";

/** Where the shipment read off a file stands, from what the server and the conversation say. */
export type ShipmentStage =
  | { stage: "none" }
  | { stage: "replying" }
  | { stage: "drafting" }
  | { stage: "notDrafted" }
  | { stage: "waiting"; proposalId: string }
  | { stage: "creating"; proposalId: string }
  | { stage: "created"; shipmentId: string; name: string }
  | { stage: "failed"; reason: string }
  | { stage: "declined" }
  | { stage: "expired" }
  | { stage: "simulated" }
  | { stage: "attached"; shipmentId: string }
  | { stage: "noFile" };

export type ShipmentStageInput = {
  documentId: string;
  attachedShipmentId: string;
  proposals: readonly AssistantProposal[];
  /** The conversation has a reply under way. */
  replying: boolean;
  /** What this card last asked about the file in this conversation, if anything. */
  ask: DocumentAsk | null;
  /** The card's question is still waiting for the conversation to send it. */
  askWaiting: boolean;
};

const CREATE_SHIPMENT = "create_shipment";

/** The newest proposal to create a shipment from this file. */
export function shipmentProposal(
  proposals: readonly AssistantProposal[],
  documentId: string,
): AssistantProposal | null {
  let newest: AssistantProposal | null = null;
  for (const proposal of proposals) {
    if (proposal.toolName !== CREATE_SHIPMENT) continue;
    if (proposal.arguments?.sourceDocumentId !== documentId) continue;
    if (newest === null || proposal.createdAt > newest.createdAt) newest = proposal;
  }
  return newest;
}

function createdShipment(proposal: AssistantProposal): { shipmentId: string; name: string } {
  const result = proposal.executionResult;
  const record = result?.record?.entityType === "shipment" ? result.record.id : "";
  return { shipmentId: record || (result?.ids.shipment ?? ""), name: result?.name ?? "" };
}

function fromProposal(proposal: AssistantProposal): ShipmentStage {
  switch (proposal.status) {
    case "Pending":
      return { stage: "waiting", proposalId: proposal.id };
    case "Approving":
    case "Accepted":
    case "Modified":
      return { stage: "creating", proposalId: proposal.id };
    case "Executed":
      return { stage: "created", ...createdShipment(proposal) };
    case "ExecutionFailed":
      return { stage: "failed", reason: proposal.executionError };
    case "Rejected":
      return { stage: "declined" };
    case "Simulated":
      return { stage: "simulated" };
    default:
      return { stage: "expired" };
  }
}

/**
 * The shipment's stage. A created shipment wins; then a file that was already a shipment when
 * it was read; then a create the card asked for after the newest proposal, which is drafting
 * while the conversation is still on it and not drafted once the reply ended without a
 * proposal; then the newest proposal itself. A reply under way on something else holds the
 * card's actions, since anything asked now would wait behind it.
 */
export function shipmentStage(input: ShipmentStageInput): ShipmentStage {
  if (input.documentId === "") return { stage: "noFile" };

  const proposal = shipmentProposal(input.proposals, input.documentId);
  if (proposal?.status === "Executed") return fromProposal(proposal);
  if (input.attachedShipmentId !== "") {
    return { stage: "attached", shipmentId: input.attachedShipmentId };
  }

  const ask = input.ask?.intent === "create" ? input.ask : null;
  if (ask && (proposal === null || ask.at > proposal.createdAt * 1000)) {
    return input.askWaiting || input.replying ? { stage: "drafting" } : { stage: "notDrafted" };
  }
  if (proposal) return fromProposal(proposal);

  return input.replying || input.askWaiting ? { stage: "replying" } : { stage: "none" };
}

/** Whether the card's own request to check fields is still with the agent. */
export function isChecking(input: Pick<ShipmentStageInput, "ask" | "askWaiting" | "replying">) {
  return input.ask?.intent === "check" && (input.askWaiting || input.replying);
}

/** Whether the card may ask the agent anything now. */
export function canAsk(stage: ShipmentStage): boolean {
  switch (stage.stage) {
    case "none":
    case "notDrafted":
    case "declined":
    case "expired":
    case "simulated":
    case "failed":
      return true;
    default:
      return false;
  }
}
