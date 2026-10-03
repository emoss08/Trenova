import { describe, expect, it } from "vitest";
import { withArtifactRefs } from "../artifact-ref";

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
