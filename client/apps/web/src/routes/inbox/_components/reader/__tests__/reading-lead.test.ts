import { describe, expect, it } from "vitest";
import { leadFacts } from "../reading-lead";

const fact = (key: string | null) => ({ key, label: key ?? "", value: "x" });

/** Each kind of message leads with what its reader looks for first. */
describe("what leads a reading", () => {
  it("leads a tender with its lane and when to answer by, and keeps the rest as read", () => {
    const facts = [fact("tender"), fact("loads"), fact("respondBy"), fact("lane"), fact("weight")];

    const { lead, rest } = leadFacts("Tender", facts);

    expect(lead.map((row) => row.key)).toEqual(["lane", "respondBy"]);
    expect(rest.map((row) => row.key)).toEqual(["tender", "loads", "weight"]);
  });

  it("falls to the next key it prefers when the first was not read", () => {
    const { lead } = leadFacts("Invoice", [fact("invoice"), fact("terms")]);

    expect(lead.map((row) => row.key)).toEqual(["invoice", "terms"]);
  });

  it("leads with nothing for a kind with no lead, or a reading with none of its keys", () => {
    const facts = [fact("tender"), fact(null)];

    expect(leadFacts("Other", facts)).toEqual({ lead: [], rest: facts });
    expect(leadFacts(null, facts)).toEqual({ lead: [], rest: facts });
    expect(leadFacts("DetentionDispute", facts)).toEqual({ lead: [], rest: facts });
  });
});
