import { describe, expect, it } from "vitest";
import { proposalLabel } from "../decision-chrome";

const t = ((text: string, ...args: unknown[]) =>
  text.replace(/\{(\d+)\}/gu, (_, index: string) => String(args[Number(index)]))) as never;

/**
 * The card in the transcript names what the agent asked to do in the words
 * of the tool and the records it is about, never "Run assign billing queue
 * billers with the values below": a person decides on a biller and eleven
 * items, not on a tool name.
 */
describe("proposalLabel", () => {
  it("names the object of the verb and counts the records it is about", () => {
    expect(
      proposalLabel(
        "assign_billing_queue_billers",
        { billingQueueItemIds: Array.from({ length: 11 }, (_, i) => `bqi_${i}`) },
        t,
      ),
    ).toBe("Assign biller · 11 billing queue items");
  });

  it("folds the records into the verb when they are what the tool names", () => {
    expect(
      proposalLabel(
        "approve_billing_queue_items",
        { billingQueueItemIds: Array.from({ length: 11 }, (_, i) => `bqi_${i}`) },
        t,
      ),
    ).toBe("Approve 11 billing queue items");
    expect(proposalLabel("approve_billing_queue_items", { billingQueueItemIds: ["bqi_1"] }, t)).toBe(
      "Approve 1 billing queue item",
    );
  });

  it("names a single record by its number rather than its id", () => {
    expect(
      proposalLabel("send_invoice", { invoiceId: "inv_01J8ZABCDEFGHJKMNPQRSTVWXY", invoiceNumber: "INV2610000016" }, t),
    ).toBe("Send invoice INV2610000016");
    expect(proposalLabel("send_invoice", { invoiceId: "inv_16" }, t)).toBe("Send invoice");
  });

  it("reads a transition as a move into its target state", () => {
    expect(
      proposalLabel("transition_item_to_in_review", { billingQueueItemIds: ["a", "b", "c"] }, t),
    ).toBe("Move 3 items into review");
    expect(proposalLabel("transition_item_to_in_review", { billingQueueItemId: "bqi_1" }, t)).toBe(
      "Move item into review",
    );
  });

  it("falls back to the humanized tool name when nothing else is known", () => {
    expect(proposalLabel("flag_for_manual_review", {}, t)).toBe("Flag for manual review");
    expect(proposalLabel("create_location", { name: "Acme DC" }, t)).toBe("Create location Acme DC");
  });
});
