import { describe, expect, it } from "vitest";
import { createSSEParser, type SSEMessage } from "../sse";

function feedAll(chunks: string[]): SSEMessage[] {
  const parser = createSSEParser();
  const out: SSEMessage[] = [];
  for (const chunk of chunks) {
    out.push(...parser.feed(chunk));
  }
  out.push(...parser.flush());
  return out;
}

/**
 * The parser follows the event-stream specification rather than the shape one
 * server happens to send: an event ends at a blank line, several data lines
 * join with newlines, a colon line is a comment, and a chunk boundary can fall
 * anywhere — mid-line, mid-multibyte character, or between the two newlines
 * that close an event.
 */
describe("createSSEParser", () => {
  it("emits one message per blank-line-terminated block", () => {
    expect(feedAll(["event: delta\ndata: {\"text\":\"a\"}\n\nevent: done\ndata: {}\n\n"])).toEqual([
      { event: "delta", data: '{"text":"a"}' },
      { event: "done", data: "{}" },
    ]);
  });

  it("reassembles a block split across chunks at any byte", () => {
    const whole = 'event: delta\ndata: {"text":"héllo"}\n\n';
    const bytes = new TextEncoder().encode(whole);
    // Split inside the multibyte "é" so a naive decoder would corrupt it.
    const cut = whole.indexOf("é") + 1;
    const decoder = new TextDecoder();
    const first = decoder.decode(bytes.slice(0, cut), { stream: true });
    const second = decoder.decode(bytes.slice(cut), { stream: true });

    expect(feedAll([first, second])).toEqual([{ event: "delta", data: '{"text":"héllo"}' }]);
  });

  it("joins multiple data lines with a newline and drops comments", () => {
    expect(feedAll([": keep-alive\nevent: x\ndata: line one\ndata: line two\n\n"])).toEqual([
      { event: "x", data: "line one\nline two" },
    ]);
  });

  it("defaults the event name to message when none is given", () => {
    expect(feedAll(["data: hello\n\n"])).toEqual([{ event: "message", data: "hello" }]);
  });

  it("ignores a block with no data at all", () => {
    expect(feedAll(["event: ping\n\n", "data: real\n\n"])).toEqual([
      { event: "message", data: "real" },
    ]);
  });

  it("accepts CRLF line endings", () => {
    expect(feedAll(["event: e\r\ndata: 1\r\n\r\n"])).toEqual([{ event: "e", data: "1" }]);
  });

  it("flushes a final block the server never terminated", () => {
    expect(feedAll(["event: done\ndata: {}"])).toEqual([{ event: "done", data: "{}" }]);
  });
});
