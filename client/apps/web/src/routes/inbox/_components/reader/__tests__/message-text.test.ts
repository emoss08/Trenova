import { describe, expect, it } from "vitest";
import {
  isRawEdi,
  splitQuotedBody,
  underlineParagraphs,
  type BodySegment,
  type UnderlineFact,
} from "../message-text";

describe("splitQuotedBody", () => {
  it("keeps the sender's own words and folds away the thread beneath a reply line", () => {
    const body = [
      "Driver is 20 minutes out.",
      "",
      "On Tue, Sep 22, 2026 at 9:14 AM Dispatch <d@x.com> wrote:",
      "> Where is the truck?",
    ].join("\n");

    expect(splitQuotedBody(body)).toEqual({
      own: "Driver is 20 minutes out.",
      quoted: "On Tue, Sep 22, 2026 at 9:14 AM Dispatch <d@x.com> wrote:\n> Where is the truck?",
    });
  });

  it("folds from a forwarded or original-message header", () => {
    expect(splitQuotedBody("FYI below\n---------- Forwarded message ---------\nFrom: a")).toEqual({
      own: "FYI below",
      quoted: "---------- Forwarded message ---------\nFrom: a",
    });
    expect(splitQuotedBody("See POD\n-----Original Message-----\nFrom: b").quoted).toBe(
      "-----Original Message-----\nFrom: b",
    );
  });

  it("folds a trailing run of quoted lines but keeps a quote the sender answers inline", () => {
    expect(splitQuotedBody("Yes.\n> Can you take it?\n> Thanks")).toEqual({
      own: "Yes.",
      quoted: "> Can you take it?\n> Thanks",
    });

    const inline = "> Rate?\n$2,400 all in.\n> Pickup?\nThursday 0800.";
    expect(splitQuotedBody(inline)).toEqual({ own: inline, quoted: "" });
  });

  it("keeps line breaks, since a body is shown as the sender laid it out", () => {
    expect(splitQuotedBody("Line one\r\nLine two\r\n\r\n").own).toBe("Line one\nLine two");
  });

  it("shows a message that is all quote rather than an empty body", () => {
    expect(splitQuotedBody("> only a quote")).toEqual({ own: "> only a quote", quoted: "" });
  });
});

const fact = (quote: string, key = quote, overrides: Partial<UnderlineFact> = {}): UnderlineFact => ({
  key,
  label: key,
  quote,
  tone: "Neutral",
  flagged: false,
  ...overrides,
});

const text = (paragraph: BodySegment[]) =>
  paragraph.map((segment) => (segment.kind === "text" ? segment.text : `[${segment.text}]`)).join("");

describe("underlineParagraphs", () => {
  it("splits the words into paragraphs at blank lines and marks each fact where its quote is", () => {
    const body = "Hello,\n\nWe are disputing the $185.00 detention on invoice INV-20931.\n\nRegards";
    const paragraphs = underlineParagraphs(body, [fact("INV-20931"), fact("$185.00 detention")]);

    expect(paragraphs.map(text)).toEqual([
      "Hello,",
      "We are disputing the [$185.00 detention] on invoice [INV-20931].",
      "Regards",
    ]);
    const marked = paragraphs[1].filter((segment) => segment.kind === "fact");
    expect(marked.map((segment) => segment.kind === "fact" && segment.fact.key)).toEqual([
      "$185.00 detention",
      "INV-20931",
    ]);
  });

  it("matches a quote across a line break or a run of spaces, as the server keeps it", () => {
    const body = "the driver arriving 40 minutes\nafter the  appointment window";
    expect(
      underlineParagraphs(body, [fact("40 minutes after the appointment window")]).map(text),
    ).toEqual(["the driver arriving [40 minutes\nafter the  appointment window]"]);
  });

  it("matches without regard to case, and keeps the message's own spelling", () => {
    expect(underlineParagraphs("Pickup Thursday 08:00", [fact("pickup thursday")]).map(text)).toEqual(
      ["[Pickup Thursday] 08:00"],
    );
  });

  it("marks only the first place a quote appears, and a later fact cannot take words already marked", () => {
    const body = "S-104251 rate con for S-104251";
    expect(
      underlineParagraphs(body, [fact("S-104251", "pro"), fact("S-104251 rate", "overlap")]).map(text),
    ).toEqual(["[S-104251] rate con for S-104251"]);
    expect(
      underlineParagraphs(body, [fact("S-104251", "pro"), fact("S-104251", "again")]).map(text),
    ).toEqual(["[S-104251] rate con for [S-104251]"]);
  });

  it("leaves out a fact whose quote is empty, is not in the words, or spans two paragraphs", () => {
    const body = "First part.\n\nSecond part.";
    expect(
      underlineParagraphs(body, [fact(""), fact("   "), fact("Third"), fact("part.\n\nSecond")]).map(text),
    ).toEqual(["First part.", "Second part."]);
  });

  it("treats a quote's regular-expression characters as plain text", () => {
    expect(underlineParagraphs("Rate $1,240 (all-in).", [fact("$1,240 (all-in)")]).map(text)).toEqual([
      "Rate [$1,240 (all-in)].",
    ]);
  });

  it("keeps a flagged fact marked, carrying the flag", () => {
    const [paragraph] = underlineParagraphs("PO 88213", [fact("88213", "po", { flagged: true })]);
    const marked = paragraph.find((segment) => segment.kind === "fact");
    expect(marked?.kind === "fact" && marked.fact.flagged).toBe(true);
  });

  it("returns no paragraphs for empty words", () => {
    expect(underlineParagraphs("", [fact("x")])).toEqual([]);
    expect(underlineParagraphs("\n\n  \n", [])).toEqual([]);
  });
});

describe("isRawEdi", () => {
  it("knows an X12 document by its segments", () => {
    expect(isRawEdi("ST*214*0441~B10*PC0441**SCAC~LX*1~AT7*X3*NS***20260919*0412~")).toBe(true);
  });

  it("does not take prose with an asterisk or a tilde for EDI", () => {
    expect(isRawEdi("Rate is *firm* ~ call me")).toBe(false);
    expect(isRawEdi("")).toBe(false);
  });
});
