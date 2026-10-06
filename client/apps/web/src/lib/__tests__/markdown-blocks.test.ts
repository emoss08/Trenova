import { describe, expect, it } from "vitest";
import { splitMarkdownBlocks } from "@/lib/markdown-blocks";

const REPLY = [
  "## Loads at risk",
  "",
  "Three loads are late to pick up.",
  "",
  "- S1 at Dallas",
  "- S2 at Austin",
  "",
  "- S3 at Waco",
  "",
  "```sql",
  "select *",
  "",
  "from shipments",
  "```",
  "",
  "| Load | ETA |",
  "| --- | --- |",
  "| S1 | 14:00 |",
  "",
  "    indented after a blank",
  "",
  "1. call the shipper",
  "",
  "2. reschedule",
  "",
  "Done.",
].join("\n");

describe("splitMarkdownBlocks", () => {
  it("loses and reorders nothing: the blocks are the text", () => {
    expect(splitMarkdownBlocks(REPLY).join("")).toBe(REPLY);
    expect(splitMarkdownBlocks("")).toEqual([]);
  });

  it("splits at a blank line before a new top-level block", () => {
    expect(splitMarkdownBlocks("One.\n\nTwo.\n\nThree.")).toEqual([
      "One.\n\n",
      "Two.\n\n",
      "Three.",
    ]);
  });

  /**
   * A list split at the blank line between its items would render as two
   * lists, and a fence split at a blank line inside it would close the code
   * early and render the rest as prose.
   */
  it("never splits a fence, a list or an indented continuation", () => {
    const blocks = splitMarkdownBlocks(REPLY);

    expect(blocks).toContain("```sql\nselect *\n\nfrom shipments\n```\n\n");
    expect(blocks.find((block) => block.includes("S1 at Dallas"))).toContain("- S3 at Waco");
    expect(blocks.find((block) => block.includes("1. call"))).toContain("2. reschedule");
    expect(blocks.find((block) => block.includes("| S1 |"))).toContain(
      "    indented after a blank",
    );
  });

  it("keeps an unclosed fence open to the end", () => {
    expect(splitMarkdownBlocks("Run this:\n\n~~~\na\n\nb")).toEqual([
      "Run this:\n\n",
      "~~~\na\n\nb",
    ]);
  });

  /**
   * The point of splitting: as a reply streams, every block but the last is
   * already final, so it is parsed once rather than on every token.
   */
  it("settles every block but the last as the text grows", () => {
    const final = splitMarkdownBlocks(REPLY);
    for (let end = 1; end <= REPLY.length; end += 1) {
      const blocks = splitMarkdownBlocks(REPLY.slice(0, end));
      for (const block of blocks.slice(0, -1)) {
        expect(final).toContain(block);
      }
    }
  });
});
