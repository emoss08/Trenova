import { getSchema } from "@tiptap/core";
import { Node as ProseMirrorNode } from "@tiptap/pm/model";
import StarterKit from "@tiptap/starter-kit";
import { describe, expect, it } from "vitest";
import {
  cleanPastedText,
  docOf,
  mentionEndingAt,
  mentionRanges,
  offsetAt,
  PASTE_AS_FILE_CHARS,
  PASTE_AS_FILE_LINES,
  pastesAsFile,
  posAt,
  textOf,
  textOfHtml,
} from "../editor/editor-text";

const schema = getSchema([StarterKit]);
const doc = (text: string) => ProseMirrorNode.fromJSON(schema, docOf(text));

const shipment = { type: "shipment", id: "shp_1", label: "PRO-1001" };
const acme = { type: "customer", id: "cus_1", label: "Acme Freight" };
const acmeWest = { type: "customer", id: "cus_2", label: "Acme Freight West" };

describe("the composer's document as text", () => {
  it("is one paragraph a line, and reads back as the same text", () => {
    for (const text of ["", "one line", "two\nlines", "a\n\nb", "\nleading", "trailing\n"]) {
      expect(textOf(doc(text))).toBe(text);
    }
    expect(doc("a\n\nb").childCount).toBe(3);
  });

  it("maps every offset of the draft to a position and back", () => {
    const text = "ab\n\ncde";
    const node = doc(text);
    for (let offset = 0; offset <= text.length; offset += 1) {
      expect(offsetAt(node, posAt(node, offset))).toBe(offset);
    }
  });

  it("holds an offset past the end at the end", () => {
    const node = doc("ab\ncd");
    expect(offsetAt(node, posAt(node, 99))).toBe(5);
  });
});

describe("named records in the draft", () => {
  it("finds each @label where it stands", () => {
    expect(mentionRanges("Is @PRO-1001 late? @PRO-1001", [shipment])).toEqual([
      { from: 3, to: 12 },
      { from: 19, to: 28 },
    ]);
  });

  it("reads the longest label, not one it starts with", () => {
    expect(mentionRanges("Call @Acme Freight West", [acme, acmeWest])).toEqual([
      { from: 5, to: 23 },
    ]);
  });

  it("finds none when no record is named or the label was edited away", () => {
    expect(mentionRanges("Is PRO-1001 late?", [shipment])).toEqual([]);
    expect(mentionRanges("Is @PRO-100 late?", [shipment])).toEqual([]);
    expect(mentionRanges("@x", [])).toEqual([]);
  });

  it("names the record a caret sits at the end of, and no other caret", () => {
    const text = "Is @PRO-1001 late?";
    expect(mentionEndingAt(text, [shipment], 12)).toEqual({ from: 3, to: 12 });
    expect(mentionEndingAt(text, [shipment], 11)).toBeNull();
    expect(mentionEndingAt(text, [shipment], 13)).toBeNull();
  });
});

describe("pasted text", () => {
  it("arrives as a message wants it", () => {
    expect(cleanPastedText("PRO\tStatus\r\nS-1\tLate  \r\n\r\n\r\n\r\nS-2​\tOn time\r\n")).toBe(
      "PRO  Status\nS-1  Late\n\nS-2  On time",
    );
    expect(cleanPastedText("\n\n  indented stays\n")).toBe("  indented stays");
    expect(cleanPastedText("﻿")).toBe("");
  });

  it("rides as a file once it is longer than a message", () => {
    expect(pastesAsFile("x".repeat(PASTE_AS_FILE_CHARS))).toBe(false);
    expect(pastesAsFile("x".repeat(PASTE_AS_FILE_CHARS + 1))).toBe(true);
    expect(pastesAsFile(Array(PASTE_AS_FILE_LINES).fill("l").join("\n"))).toBe(false);
    expect(
      pastesAsFile(
        Array(PASTE_AS_FILE_LINES + 1)
          .fill("l")
          .join("\n"),
      ),
    ).toBe(true);
  });

  it("reads HTML that carries no plain text: blocks on lines, lists marked, cells apart, no script", () => {
    const html =
      "<style>p{}</style><p>Late loads:</p><ul><li>S-1</li><li>S-2</li></ul>" +
      "<ol><li>Call</li><li>Rebook</li></ol>" +
      "<table><tr><td>PRO</td><td>Status</td></tr><tr><td>S-1</td><td>Late</td></tr></table>" +
      "<script>alert(1)</script>line<br>break";

    expect(cleanPastedText(textOfHtml(html))).toBe(
      "Late loads:\n- S-1\n- S-2\n1. Call\n2. Rebook\nPRO  Status\nS-1  Late\nline\nbreak",
    );
  });
});
