import type { AssistantProposal } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import {
  canAsk,
  isChecking,
  shipmentProposal,
  shipmentStage,
  type ShipmentStageInput,
} from "../shipment-stage";

const DOCUMENT = "doc_1";

function proposal(overrides: Partial<AssistantProposal> = {}): AssistantProposal {
  return {
    id: "aprop_1",
    toolName: "create_shipment",
    arguments: { sourceDocumentId: DOCUMENT },
    status: "Pending",
    createdAt: 1_000,
    executionError: "",
    executionResult: null,
    ...overrides,
  } as AssistantProposal;
}

function input(overrides: Partial<ShipmentStageInput> = {}): ShipmentStageInput {
  return {
    documentId: DOCUMENT,
    attachedShipmentId: "",
    proposals: [],
    replying: false,
    ask: null,
    askWaiting: false,
    ...overrides,
  };
}

describe("shipmentStage", () => {
  it("is not created before anything is asked", () => {
    expect(shipmentStage(input())).toEqual({ stage: "none" });
  });

  it("holds while the conversation replies to something else", () => {
    expect(shipmentStage(input({ replying: true }))).toEqual({ stage: "replying" });
  });

  it("is drafting from the ask until the agent proposes it, and says so when it did not", () => {
    const ask = { threadId: "athr_1", intent: "create" as const, at: 2_000_000 };

    expect(shipmentStage(input({ ask, askWaiting: true }))).toEqual({ stage: "drafting" });
    expect(shipmentStage(input({ ask, replying: true }))).toEqual({ stage: "drafting" });
    expect(shipmentStage(input({ ask }))).toEqual({ stage: "notDrafted" });
  });

  it("follows the proposal for this file through approval", () => {
    const stage = (status: AssistantProposal["status"], extra: Partial<AssistantProposal> = {}) =>
      shipmentStage(input({ proposals: [proposal({ status, ...extra })] }));

    expect(stage("Pending")).toEqual({ stage: "waiting", proposalId: "aprop_1" });
    expect(stage("Approving")).toEqual({ stage: "creating", proposalId: "aprop_1" });
    expect(stage("Rejected")).toEqual({ stage: "declined" });
    expect(stage("Expired")).toEqual({ stage: "expired" });
    expect(stage("Superseded")).toEqual({ stage: "expired" });
    expect(stage("Simulated")).toEqual({ stage: "simulated" });
    expect(stage("ExecutionFailed", { executionError: "The customer is inactive." })).toEqual({
      stage: "failed",
      reason: "The customer is inactive.",
    });
    expect(
      stage("Executed", {
        executionResult: {
          action: "Created",
          kind: "shipment",
          name: "S-104260",
          ids: { shipment: "shp_ids" },
          record: { entityType: "shipment", id: "shp_1" },
        },
      }),
    ).toEqual({ stage: "created", shipmentId: "shp_1", name: "S-104260" });
  });

  it("drafts again when the card asked after the newest proposal", () => {
    const declined = proposal({ status: "Rejected", createdAt: 1_000 });
    const ask = { threadId: "athr_1", intent: "create" as const, at: 2_000 * 1000 };

    expect(shipmentStage(input({ proposals: [declined], ask, replying: true }))).toEqual({
      stage: "drafting",
    });
    expect(
      shipmentStage(input({ proposals: [declined], ask: { ...ask, at: 500 * 1000 } })),
    ).toEqual({ stage: "declined" });
  });

  it("puts a created shipment before one the file already was, and that before a draft", () => {
    const executed = proposal({
      status: "Executed",
      executionResult: {
        action: "Created",
        kind: "shipment",
        name: "",
        ids: { shipment: "shp_new" },
        record: null,
      },
    });

    expect(shipmentStage(input({ proposals: [executed], attachedShipmentId: "shp_old" }))).toEqual({
      stage: "created",
      shipmentId: "shp_new",
      name: "",
    });
    expect(
      shipmentStage(input({ proposals: [proposal()], attachedShipmentId: "shp_old" })),
    ).toEqual({ stage: "attached", shipmentId: "shp_old" });
  });

  it("has nothing to draft from when the file is gone", () => {
    expect(shipmentStage(input({ documentId: "" }))).toEqual({ stage: "noFile" });
  });
});

describe("shipmentProposal", () => {
  it("takes the newest create_shipment proposal for this file only", () => {
    const proposals = [
      proposal({ id: "old", createdAt: 1 }),
      proposal({ id: "other_file", createdAt: 9, arguments: { sourceDocumentId: "doc_2" } }),
      proposal({ id: "other_tool", createdAt: 9, toolName: "update_shipment" }),
      proposal({ id: "new", createdAt: 5 }),
    ];

    expect(shipmentProposal(proposals, DOCUMENT)?.id).toBe("new");
    expect(shipmentProposal([], DOCUMENT)).toBeNull();
  });
});

describe("what may be asked", () => {
  it("asks nothing while a draft is under way, waiting or done", () => {
    expect(canAsk({ stage: "none" })).toBe(true);
    expect(canAsk({ stage: "declined" })).toBe(true);
    expect(canAsk({ stage: "drafting" })).toBe(false);
    expect(canAsk({ stage: "waiting", proposalId: "p" })).toBe(false);
    expect(canAsk({ stage: "created", shipmentId: "s", name: "" })).toBe(false);
    expect(canAsk({ stage: "noFile" })).toBe(false);
  });

  it("is checking while the card's own check is with the agent", () => {
    const ask = { threadId: "athr_1", intent: "check" as const, at: 1 };

    expect(isChecking({ ask, askWaiting: true, replying: false })).toBe(true);
    expect(isChecking({ ask, askWaiting: false, replying: false })).toBe(false);
    expect(
      isChecking({ ask: { ...ask, intent: "create" }, askWaiting: true, replying: true }),
    ).toBe(false);
  });
});
