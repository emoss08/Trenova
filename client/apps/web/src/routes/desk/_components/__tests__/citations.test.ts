import type { ToolStep } from "@/components/assistant/activity";
import { describe, expect, it } from "vitest";
import { CITATION_HREF, citeSteps, stepAnchors, withCitations } from "../conversation/citations";

function step(name: string, value: unknown, summary = ""): ToolStep {
  return {
    id: `call_${name}`,
    name,
    arguments: {},
    status: "done",
    content: `<untrusted_data>${JSON.stringify(value)}</untrusted_data>`,
    effect: "lookup",
    summary,
    durationSeconds: 1.2,
  };
}

describe("stepAnchors", () => {
  it("collects names, identifiers and counts, and leaves out ids and dates", () => {
    const anchors = stepAnchors(
      step(
        "list_billing_queue",
        {
          items: [
            { id: "bqi_01M3Z040RBC9H3QCT87PYQMNKZ", number: "BQ-24101", customer: "Acme" },
            { createdAt: "2026-10-02T10:30:00Z", total: 1 },
          ],
        },
        "15 items",
      ),
    );
    expect(anchors).toContain("BQ-24101");
    expect(anchors).toContain("Acme");
    expect(anchors).toContain("15");
    expect(anchors).not.toContain("bqi_01M3Z040RBC9H3QCT87PYQMNKZ");
    expect(anchors.some((anchor) => anchor.startsWith("2026-"))).toBe(false);
    expect(anchors).not.toContain("1");
  });
});

describe("citeSteps", () => {
  const queue = step("list_billing_queue", { items: [] }, "15 items");
  const shipment = step("get_shipment", { proNumber: "SEED-SHP-001", status: "New" });

  it("cites each step after the phrase that repeats what it found", () => {
    const text = "There are 15 items in the billing queue. SEED-SHP-001 is still New.";
    const citations = citeSteps(text, [queue, shipment]);
    expect(citations.map((citation) => citation.n)).toEqual([1, 2]);
    expect(text.slice(0, citations[0].offset)).toBe("There are 15 items");
    expect(text.slice(0, citations[1].offset)).toBe(
      "There are 15 items in the billing queue. SEED-SHP-001",
    );
  });

  it("leaves a step uncited when the reply never repeats it", () => {
    expect(citeSteps("Nothing is waiting.", [queue])).toEqual([]);
  });

  it("never cites inside code or a link", () => {
    expect(citeSteps("See `SEED-SHP-001` or [SEED-SHP-001](/shipments).", [shipment])).toEqual([]);
  });

  it("matches whole words only", () => {
    expect(citeSteps("There are 150 items.", [queue])).toEqual([]);
  });

  it("does not cite steps that look nothing up", () => {
    const ask = step("ask_user", { question: "Which SEED-SHP-001?" });
    expect(citeSteps("SEED-SHP-001 it is.", [ask])).toEqual([]);
  });
});

describe("withCitations", () => {
  it("writes each citation in as a link after its phrase", () => {
    const text = "SEED-SHP-001 is New.";
    const citations = citeSteps(text, [step("get_shipment", { proNumber: "SEED-SHP-001" })]);
    expect(withCitations(text, citations)).toBe(`SEED-SHP-001[1](${CITATION_HREF}1) is New.`);
  });
});

describe("citeSteps with similar lookups", () => {
  it("cites each lookup at what only it found, not at what both share", () => {
    const first = step("get_shipment", { proNumber: "SEED-SHP-001", status: "New" });
    const second = step("get_shipment", { proNumber: "SEED-SHP-002", status: "New" });
    const text = "SEED-SHP-001 is still New. SEED-SHP-002 is also New.";
    const citations = citeSteps(text, [first, second]);
    expect(citations.map((citation) => text.slice(0, citation.offset))).toEqual([
      "SEED-SHP-001",
      "SEED-SHP-001 is still New. SEED-SHP-002",
    ]);
  });
});
