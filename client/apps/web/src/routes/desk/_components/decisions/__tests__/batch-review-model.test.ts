import { describe, expect, it } from "vitest";
import { field, message, preview, record } from "@/components/assistant/__tests__/preview-fixtures";
import type { ProposalPreview } from "@/lib/graphql/agent-preview";
import { approvableRows, batchReview, type BatchPreviewEntry } from "../batch-review-model";

function driverMessage(id: string, driver: string, pro: string, worker: string): ProposalPreview {
  return preview({
    proposalId: id,
    tool: "worker_send_message",
    digest: `sha256:${id}`,
    reviewRequired: true,
    changes: [
      record({
        resource: "worker",
        record: { entityType: "worker", id: worker },
        entityId: worker,
        label: driver,
        operation: "Send",
        fields: [
          field({ path: "priority", label: "Priority", before: null, after: "medium" }),
          field({ path: "shipmentId", label: "Shipment", before: null, after: pro }),
        ],
        message: message({
          channel: "Dash",
          from: "Billing",
          to: [driver],
          subject: `Signed POD needed: PRO ${pro}`,
          body: `Hi ${driver.split(" ")[0]}, please send the signed proof of delivery for PRO ${pro}. A clear photo of the signed paperwork in Dash is fine.`,
        }),
      }),
    ],
  });
}

const entry = (p: ProposalPreview): BatchPreviewEntry => ({
  proposalId: p.proposalId,
  preview: p,
  error: null,
});

describe("batchReview", () => {
  it("shows what every message shares once and what each fills in on its row", () => {
    const review = batchReview([
      entry(driverMessage("ap_1", "Laura Wright", "S1001", "wrk_1")),
      entry(driverMessage("ap_2", "Theresa Aguilar", "S1002", "wrk_2")),
      entry(driverMessage("ap_3", "Sam Ortiz", "S1003", "wrk_3")),
    ]);

    expect(review.layout).toBe("compact");
    expect(review.envelope?.channel).toBe("Dash");
    expect(review.envelope?.from).toBe("Billing");
    expect(review.shared.map((shared) => shared.key)).toEqual(["0:priority"]);
    expect(review.subject?.parts).toEqual([
      { kind: "text", text: "Signed POD needed: PRO " },
      { kind: "slot", slot: 0 },
    ]);
    expect(review.body?.slots).toBe(2);
    expect(review.rows.map((row) => row.title)).toEqual([
      "Laura Wright",
      "Theresa Aguilar",
      "Sam Ortiz",
    ]);
    expect(review.rows.map((row) => row.to)).toEqual([
      ["Laura Wright"],
      ["Theresa Aguilar"],
      ["Sam Ortiz"],
    ]);
    expect(review.rows.map((row) => row.subjectValues)).toEqual([["S1001"], ["S1002"], ["S1003"]]);
    expect(review.rows.map((row) => row.bodyValues)).toEqual([
      ["Laura", "S1001"],
      ["Theresa", "S1002"],
      ["Sam", "S1003"],
    ]);
    expect(review.rows[1].fields.map((own) => [own.key, own.field.after])).toEqual([
      ["0:shipmentId", "S1002"],
    ]);
  });

  it("shows each change whole when the changes differ in shape", () => {
    const email = driverMessage("ap_2", "Theresa Aguilar", "S1002", "wrk_2");
    email.changes[0].message = message({ channel: "Email", to: ["ops@acme.test"] });

    const review = batchReview([
      entry(driverMessage("ap_1", "Laura Wright", "S1001", "wrk_1")),
      entry(email),
    ]);

    expect(review.layout).toBe("whole");
    expect(review.envelope).toBeNull();
    expect(review.shared).toEqual([]);
    expect(review.rows.map((row) => row.proposalId)).toEqual(["ap_1", "ap_2"]);
  });

  it("shows every body whole when the messages share too little wording", () => {
    const first = driverMessage("ap_1", "Laura Wright", "S1001", "wrk_1");
    const second = driverMessage("ap_2", "Theresa Aguilar", "S1002", "wrk_2");
    second.changes[0].message = message({
      ...second.changes[0].message!,
      body: "Your settlement for last week is ready to review in Dash.",
    });

    const review = batchReview([entry(first), entry(second)]);

    expect(review.layout).toBe("compact");
    expect(review.body).toBeNull();
    expect(review.rows.map((row) => row.bodyValues)).toEqual([null, null]);
    expect(review.rows[1].body).toBe("Your settlement for last week is ready to review in Dash.");
  });

  it("keeps a row whose preview failed, in its place, out of what is alike", () => {
    const review = batchReview([
      entry(driverMessage("ap_1", "Laura Wright", "S1001", "wrk_1")),
      { proposalId: "ap_2", preview: null, error: "This change could not be previewed." },
      entry(driverMessage("ap_3", "Sam Ortiz", "S1003", "wrk_3")),
    ]);

    expect(review.layout).toBe("compact");
    expect(review.rows.map((row) => [row.proposalId, row.error])).toEqual([
      ["ap_1", null],
      ["ap_2", "This change could not be previewed."],
      ["ap_3", null],
    ]);
    expect(review.rows[1].bodyValues).toBeNull();
    expect(review.rows[2].bodyValues).toEqual(["Sam", "S1003"]);
  });

  it("reads a value withheld from the reader as its own, never as alike", () => {
    const first = driverMessage("ap_1", "Laura Wright", "S1001", "wrk_1");
    const second = driverMessage("ap_2", "Theresa Aguilar", "S1002", "wrk_2");
    second.changes[0].fields[0] = field({
      path: "priority",
      label: "Priority",
      before: null,
      after: null,
      withheld: true,
    });

    const review = batchReview([entry(first), entry(second)]);

    expect(review.shared).toEqual([]);
    expect(review.rows[1].fields.map((own) => own.key)).toEqual(["0:priority", "0:shipmentId"]);
  });
});

describe("approvableRows", () => {
  it("carries each read row's digest and leaves out the stale, the failed and the unticked", () => {
    const stale = driverMessage("ap_3", "Sam Ortiz", "S1003", "wrk_3");
    stale.stale = true;
    const review = batchReview([
      entry(driverMessage("ap_1", "Laura Wright", "S1001", "wrk_1")),
      { proposalId: "ap_2", preview: null, error: "Proposal not found" },
      entry(stale),
      entry(driverMessage("ap_4", "Ana Ruiz", "S1004", "wrk_4")),
      entry(driverMessage("ap_5", "Bo Chen", "S1005", "wrk_5")),
    ]);

    expect(approvableRows(review, new Set(["ap_5"]))).toEqual([
      { proposalId: "ap_1", digest: "sha256:ap_1" },
      { proposalId: "ap_4", digest: "sha256:ap_4" },
    ]);
  });
});
