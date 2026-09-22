import { describe, expect, it } from "vitest";
import { createSSEParser } from "@trenova/shared/lib/sse";

describe("createSSEParser", () => {
  it("surfaces the resume cursor the server sent", () => {
    const parser = createSSEParser();

    const [message] = parser.feed('id: 1738-0\nevent: delta\ndata: {"text":"hi"}\n\n');

    expect(message).toEqual({ event: "delta", data: '{"text":"hi"}', id: "1738-0" });
  });

  it("leaves the cursor empty when the server sent none", () => {
    const parser = createSSEParser();

    const [message] = parser.feed('event: delta\ndata: {"text":"hi"}\n\n');

    expect(message.id).toBe("");
  });

  // The specification says an id persists until replaced: a server that sends
  // one id and then several events means all of them to carry it.
  it("carries the cursor forward to later events", () => {
    const parser = createSSEParser();

    parser.feed("id: 7-0\nevent: delta\ndata: a\n\n");
    const [second] = parser.feed("event: delta\ndata: b\n\n");

    expect(second.id).toBe("7-0");
  });

  it("keeps the cursor across a chunk boundary", () => {
    const parser = createSSEParser();

    expect(parser.feed("id: 99-")).toEqual([]);
    const [message] = parser.feed("2\nevent: done\ndata: {}\n\n");

    expect(message.id).toBe("99-2");
  });

  it("ignores an id containing a NUL, as the specification requires", () => {
    const parser = createSSEParser();

    const [message] = parser.feed("id: 5-\u00000\nevent: delta\ndata: a\n\n");

    expect(message.id).toBe("");
  });
});
