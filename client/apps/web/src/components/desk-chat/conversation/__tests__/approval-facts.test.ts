import { field, preview, record } from "@/components/assistant/__tests__/preview-fixtures";
import type { PreviewRef } from "@/lib/graphql/agent-preview";
import { translate } from "@trenova/shared/i18n/runtime";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import {
  approvalFacts,
  OUTCOME_FIELD,
  recordName,
  scopePhrase,
  valueText,
} from "../approval-facts";

const t = ((template: string, ...args: unknown[]) =>
  template.replace(/\{(\d+)\}/g, (_, index: string) => String(args[Number(index)]))) as TranslateFn;

const BILLER_ID = "usr_01M49Z0BSKQ6A3XQ72VD71DDY8";

function ref(overrides: Partial<PreviewRef> = {}): PreviewRef {
  return {
    resource: "user",
    id: BILLER_ID,
    label: null,
    record: null,
    withheld: false,
    ...overrides,
  };
}

/*
billing_assign_biller as the server served it: the item named by its
number, the biller a reference to a person. The SDL lets a reference's label be
absent (the record is gone, or its kind had no label when the preview was
recorded) and lets a record's label be empty.
*/
function assignment(afterRef: PreviewRef, label = "INV2610000079") {
  return preview({
    tool: "billing_assign_biller",
    changes: [
      record({
        resource: "billing_queue",
        record: null,
        entityId: "bqi_01M4K7PFD01FT0SCJQKW5VPTJF",
        label,
        fields: [
          field({
            path: "assignedBillerId",
            label: "Assigned biller",
            before: null,
            after: BILLER_ID,
            afterRef,
          }),
          field({
            path: OUTCOME_FIELD,
            label: "What happens",
            before: null,
            after: "Would assign",
          }),
        ],
      }),
    ],
  });
}

describe("valueText", () => {
  it("names the record a reference points at", () => {
    expect(valueText(BILLER_ID, ref({ label: "Avery Biller" }), t)).toBe("Avery Biller");
  });

  it("never shows the id of a reference with no name", () => {
    const text = valueText(BILLER_ID, ref(), t);

    expect(text).not.toContain(BILLER_ID);
    expect(text).toBe("A record that is no longer there");
  });

  it("says a reference the reader may not see is hidden, not its id", () => {
    const text = valueText(BILLER_ID, ref({ withheld: true }), t);

    expect(text).not.toContain(BILLER_ID);
    expect(text).toBe("Hidden by your data access");
  });

  it("shows a plain value as it is", () => {
    expect(valueText("InReview", null, t)).toBe("InReview");
    expect(valueText(null, null, t)).toBe("None");
  });
});

describe("recordName", () => {
  it("names a record by its label, or by its kind when it has none", () => {
    expect(recordName({ label: "INV2610000079", resource: "billing_queue" })).toBe("INV2610000079");
    expect(recordName({ label: "", resource: "worker_pto" })).toBe("Worker time off");
  });
});

describe("approvalFacts", () => {
  it("says who a biller assignment names, never their id", () => {
    const facts = approvalFacts(assignment(ref({ label: "Avery Biller" })), t);

    expect(facts.field).toEqual({
      label: "Assigned biller",
      before: "None",
      after: "Avery Biller",
    });
  });

  it("keeps an unnamed biller's id off the card", () => {
    const facts = approvalFacts(assignment(ref()), t);

    expect(facts.field?.after).not.toContain(BILLER_ID);
  });

  it("names a refused record by its kind when it has no label, never its id", () => {
    const refused = assignment(ref({ label: "Avery Biller" }), "");
    refused.changes[0].fields[1] = field({
      path: OUTCOME_FIELD,
      after: "Refused: it is already posted",
    });

    const facts = approvalFacts(refused, t);

    expect(facts.refused).toHaveLength(1);
    expect(facts.refused[0].label).not.toContain("bqi_");
    expect(facts.refused[0].reason).toBe("it is already posted");
  });
});

/*
The card used to say only "on 1 shipment", so a dispatcher had to open Review to
learn which load a change was for. One existing record is named by its own label;
several are counted; a record being created carries only a placeholder label.
*/
describe("scopePhrase", () => {
  // The catalog's own formatter, so counts read as the card prints them.
  const t = translate as TranslateFn;

  it("names the one existing record a change acts on", () => {
    const facts = approvalFacts(preview({ changes: [record({ label: "S-240201" })] }), t);

    expect(facts.record).toBe("S-240201");
    expect(scopePhrase(facts, t)).toBe("on S-240201");
  });

  it("counts several records instead of naming the first", () => {
    const facts = approvalFacts(
      preview({
        changes: [
          record({ label: "S-240201", entityId: "shp_1" }),
          record({ label: "S-240202", entityId: "shp_2" }),
        ],
      }),
      t,
    );

    expect(facts.record).toBe("");
    expect(scopePhrase(facts, t)).toBe("on 2 shipments");
  });

  it("counts a record it may not show rather than naming the one it may", () => {
    const facts = approvalFacts(
      preview({ changes: [record({ label: "S-240201" })], withheldCount: 1 }),
      t,
    );

    expect(scopePhrase(facts, t)).toBe("on 2 shipments");
  });

  it("falls back to the count when the record has no label", () => {
    const facts = approvalFacts(preview({ changes: [record({ label: "" })] }), t);

    expect(scopePhrase(facts, t)).toBe("on 1 shipment");
  });

  it("does not name a record being created by its placeholder label", () => {
    const facts = approvalFacts(
      preview({
        changes: [
          record({ record: null, entityId: null, label: "New shipment", operation: "Create" }),
        ],
      }),
      t,
    );

    expect(scopePhrase(facts, t)).toBeNull();
  });
});
