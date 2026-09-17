import type { AssistantProposal } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { presentProposal } from "../proposal-presenters";

function proposal(overrides: Partial<AssistantProposal>): AssistantProposal {
  return {
    id: "aprop_1",
    runId: "arun_1",
    threadId: "athr_1",
    sourceMessageId: "amsg_1",
    toolName: "request_missing_docs",
    arguments: {},
    rationale: "",
    confidence: 0.8,
    autonomyTier: "Propose",
    status: "Pending",
    executedAt: null,
    executionError: "",
    createdAt: 1,
    updatedAt: 1,
    ...overrides,
  } as AssistantProposal;
}

/**
 * Tool arguments follow the server's JSON schemas
 * (services/agenttoolservice/*.go). The card explains a change in the
 * approver's words, so each known tool gets a sentence and the fields that
 * matter, and an unknown tool still shows every argument rather than none.
 */
describe("presentProposal", () => {
  it("explains a document request by recipient and documents", () => {
    const view = presentProposal(
      proposal({
        toolName: "request_missing_docs",
        arguments: {
          to: ["ap@acme.example", "ops@acme.example"],
          subject: "Missing POD for S12345",
          customerName: "Acme",
          shipmentProNumber: "S12345",
          requestedDocuments: ["Proof of delivery", "Signed BOL"],
          body: "Hello, please send the documents.",
        },
      }),
    );

    expect(view.title).toBe("Request missing documents");
    expect(view.summary).toBe("Email Acme about shipment S12345 asking for 2 documents.");
    expect(view.facts).toEqual([
      { label: "To", value: "ap@acme.example, ops@acme.example" },
      { label: "Subject", value: "Missing POD for S12345" },
      { label: "Documents", value: "Proof of delivery, Signed BOL" },
    ]);
    expect(view.longText).toEqual({ label: "Message", value: "Hello, please send the documents." });
    expect(view.reversible).toBe(false);
  });

  it("shows a charge correction as the charges that would replace the current ones", () => {
    const view = presentProposal(
      proposal({
        toolName: "correct_charge_code",
        arguments: {
          billingQueueItemId: "bqi_1",
          additionalCharges: [
            { accessorialChargeId: "acc_1", amount: 125, description: "Detention" },
            { accessorialChargeId: "acc_2", amount: 40 },
          ],
        },
      }),
    );

    expect(view.title).toBe("Correct charge codes");
    expect(view.summary).toBe(
      "Replace the additional charges on billing item bqi_1 with 2 charges.",
    );
    expect(view.facts).toEqual([
      { label: "Billing item", value: "bqi_1" },
      { label: "Charge 1", value: "Detention · 125" },
      { label: "Charge 2", value: "acc_2 · 40" },
    ]);
    expect(view.reversible).toBe(true);
  });

  it("describes a move assignment by driver and equipment", () => {
    const view = presentProposal(
      proposal({
        toolName: "assign_move",
        arguments: {
          shipmentMoveId: "smv_1",
          primaryWorkerId: "wrk_1",
          secondaryWorkerId: "",
          tractorId: "trc_1",
          trailerId: "trl_1",
        },
      }),
    );

    expect(view.title).toBe("Assign move");
    expect(view.summary).toBe(
      "Put driver wrk_1 on move smv_1 with tractor trc_1 and trailer trl_1.",
    );
    expect(view.facts).toEqual([
      { label: "Move", value: "smv_1" },
      { label: "Driver", value: "wrk_1" },
      { label: "Tractor", value: "trc_1" },
      { label: "Trailer", value: "trl_1" },
    ]);
  });

  it("falls back to every argument for a tool it has no words for", () => {
    const view = presentProposal(
      proposal({
        toolName: "rotate_tires",
        arguments: { tractorId: "trc_9", mileage: 120000, notes: null },
      }),
    );

    expect(view.title).toBe("Rotate tires");
    expect(view.summary).toBe("Run rotate_tires with the parameters below.");
    expect(view.facts).toEqual([
      { label: "mileage", value: "120000" },
      { label: "notes", value: "—" },
      { label: "tractorId", value: "trc_9" },
    ]);
    expect(view.reversible).toBe(false);
  });

  it("never hides an argument the presenter did not expect", () => {
    const view = presentProposal(
      proposal({
        toolName: "assign_move",
        arguments: { shipmentMoveId: "smv_1", primaryWorkerId: "wrk_1", surprise: "yes" },
      }),
    );

    expect(view.facts).toContainEqual({ label: "surprise", value: "yes" });
  });
});
