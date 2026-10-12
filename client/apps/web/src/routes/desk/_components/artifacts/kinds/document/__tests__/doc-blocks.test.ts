import { describe, expect, it } from "vitest";
import {
  editedBlockMarkdown,
  inlineParts,
  joinDocBlocks,
  parseDocBlocks,
  plainText,
  replaceDocBlock,
  wordCount,
} from "../doc-blocks";

const brief = `Three loads will be late tonight. **Acme and Bluewater need a heads-up**[^1].

## What's happening

Storms over central Iowa have slowed I-80
since about 6 PM[^2].

| Load | Delay |
| --- | ---: |
| SEED-SHP-002 | +2 h |

1. Send the drafted notice
2. Re-check ETAs at 8 PM`;

describe("parseDocBlocks", () => {
  it("reads a summary, headings, paragraphs, tables and lists", () => {
    const blocks = parseDocBlocks(brief);
    expect(blocks.map((block) => block.type)).toEqual(["sum", "h", "p", "table", "ol"]);
    expect(blocks[1].text).toBe("What's happening");
    expect(blocks[2].text).toBe("Storms over central Iowa have slowed I-80 since about 6 PM[^2].");
    expect(blocks[3].rows).toEqual([["SEED-SHP-002", "+2 h"]]);
    expect(blocks[4].items).toEqual(["Send the drafted notice", "Re-check ETAs at 8 PM"]);
  });

  it("keeps each block as written, so the body is a substring the server can find", () => {
    for (const block of parseDocBlocks(brief)) {
      expect(brief).toContain(block.raw);
    }
    expect(joinDocBlocks(parseDocBlocks(brief))).toBe(brief);
  });
});

describe("replaceDocBlock", () => {
  it("changes one block and leaves the rest as they were", () => {
    const next = replaceDocBlock(brief, "b2", "Storms slowed I-80 from 6 PM[^2].");
    expect(next).toContain("Storms slowed I-80 from 6 PM[^2].");
    expect(next).toContain("| SEED-SHP-002 | +2 h |");
    expect(next.startsWith("Three loads will be late tonight.")).toBe(true);
  });
});

describe("inlineParts", () => {
  it("finds emphasis and citation marks", () => {
    expect(inlineParts("A **b**[^3] and *c*.")).toEqual([
      { kind: "text", text: "A " },
      { kind: "bold", text: "b" },
      { kind: "cite", n: 3 },
      { kind: "text", text: " and " },
      { kind: "italic", text: "c" },
      { kind: "text", text: "." },
    ]);
  });
});

describe("plain text and words", () => {
  it("reads markdown as text", () => {
    expect(plainText("**Acme** needs it[^1]")).toBe("Acme needs it[1]");
    expect(wordCount("## Title\n\nOne two **three**.")).toBe(4);
  });
});

describe("editedBlockMarkdown", () => {
  it("writes an edited paragraph back as markdown, chips as their marks", () => {
    const element = document.createElement("p");
    element.innerHTML = 'Tell <b>Acme</b> now<span data-cite="4">4</span>.';
    const [block] = parseDocBlocks("Tell **Acme** now[^4].");
    expect(editedBlockMarkdown(block, element)).toBe("Tell **Acme** now[^4].");
  });

  it("numbers an edited list again", () => {
    const element = document.createElement("ol");
    element.innerHTML = "<li>First</li><li></li><li>Second</li>";
    const [block] = parseDocBlocks("1. a\n2. b");
    expect(editedBlockMarkdown(block, element)).toBe("1. First\n2. Second");
  });
});
