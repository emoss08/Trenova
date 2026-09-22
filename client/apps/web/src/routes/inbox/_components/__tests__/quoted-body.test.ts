import { describe, expect, it } from "vitest";
import { splitQuotedBody } from "../quoted-body";

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
