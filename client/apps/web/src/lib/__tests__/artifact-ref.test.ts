import { describe, expect, it } from "vitest";
import { keptWhileWriting, withArtifactRefs } from "../artifact-ref";

const queue = { id: "art_1", title: "Billing queue items" };
const REF = "[Billing queue items](artifact:art_1)";

describe("withArtifactRefs", () => {
  it("moves a badge set on its own line after a table into the opening sentence", () => {
    const text = `There are **9 items**.\n\n| A | B |\n| - | - |\n| 1 | 2 |\n\n${REF}`;
    expect(withArtifactRefs(text, [queue])).toBe(
      `There are **9 items**. ${REF}\n\n| A | B |\n| - | - |\n| 1 | 2 |`,
    );
  });

  it("names an artifact the reply left out in its opening sentence", () => {
    expect(withArtifactRefs("Nine items.\n\n- one\n- two", [queue])).toBe(
      `Nine items. ${REF}\n\n- one\n- two`,
    );
  });

  it("leaves a badge already inside a sentence where it is", () => {
    const text = `See ${REF} for the rest.`;
    expect(withArtifactRefs(text, [queue])).toBe(text);
  });

  it("does not treat code as a sentence, and keeps its blank lines", () => {
    const text = "```text\na\n\n\nb\n```\n\nDone.";
    expect(withArtifactRefs(text, [queue])).toBe(`\`\`\`text\na\n\n\nb\n\`\`\`\n\nDone. ${REF}`);
  });

  it("keeps a reply with no sentence and puts the badge at its end", () => {
    expect(withArtifactRefs("| A |\n| - |\n| 1 |", [queue])).toBe(`| A |\n| - |\n| 1 |\n\n${REF}`);
  });
});

/*
A lookup the reply never names is not drawn while the reply is written.

A dispatcher asked for drivers to rank. The agent listed shipments and opened
six of them, each saved as a table or card as its tool finished, and answered
with a table of its own. Every one was drawn under the reply as it streamed
and gone when the turn ended, because the server keeps a lookup only when the
reply points to it.
*/
describe("keptWhileWriting", () => {
  it("leaves out lookups and keeps what the turn made on purpose", () => {
    const made = [
      { id: "art_1", kind: "table_view", title: "Shipments" },
      { id: "art_2", kind: "entity_card", title: "SEED-SHP-001" },
      { id: "art_3", kind: "document", title: "Ranking" },
    ];
    expect(keptWhileWriting(made).map((artifact) => artifact.id)).toEqual(["art_3"]);
  });
});
