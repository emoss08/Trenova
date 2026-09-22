/**
 * One server-sent event. `event` defaults to "message" when the server names
 * none, as the specification says it should.
 *
 * `id` is the server's resume cursor, empty when it sent none. A reader that
 * loses its connection sends the last id it *applied* back as Last-Event-ID —
 * not the last it received, which can be a frame that never made it out of the
 * socket.
 */
export type SSEMessage = {
  event: string;
  data: string;
  id: string;
};

export type SSEParser = {
  /** Feeds decoded text and returns every event completed by it. */
  feed: (chunk: string) => SSEMessage[];
  /** Returns the trailing event of a stream that closed without a blank line. */
  flush: () => SSEMessage[];
};

/**
 * An incremental parser for text/event-stream.
 *
 * Network chunks fall wherever the socket decides, so a line, an event or a
 * multibyte character can be split across two reads. The parser keeps the
 * unfinished remainder between calls; the caller keeps a streaming
 * TextDecoder so bytes are never decoded twice.
 */
export function createSSEParser(): SSEParser {
  let remainder = "";
  let eventName = "";
  let dataLines: string[] = [];
  // The id persists across events, as the specification says: a server that
  // sends one id and then several events means all of them to carry it.
  let lastId = "";

  const complete = (): SSEMessage[] => {
    if (dataLines.length === 0) {
      eventName = "";
      return [];
    }
    const message: SSEMessage = {
      event: eventName === "" ? "message" : eventName,
      data: dataLines.join("\n"),
      id: lastId,
    };
    eventName = "";
    dataLines = [];
    return [message];
  };

  const consumeLine = (raw: string): SSEMessage[] => {
    const line = raw.endsWith("\r") ? raw.slice(0, -1) : raw;
    if (line === "") {
      return complete();
    }
    if (line.startsWith(":")) {
      return [];
    }
    const colon = line.indexOf(":");
    const field = colon === -1 ? line : line.slice(0, colon);
    let value = colon === -1 ? "" : line.slice(colon + 1);
    if (value.startsWith(" ")) {
      value = value.slice(1);
    }
    if (field === "event") {
      eventName = value;
    } else if (field === "data") {
      dataLines.push(value);
    } else if (field === "id" && !value.includes("\u0000")) {
      // A NUL in an id is the one case the specification says to ignore
      // rather than store.
      lastId = value;
    }
    return [];
  };

  return {
    feed(chunk) {
      remainder += chunk;
      const lines = remainder.split("\n");
      remainder = lines.pop() ?? "";
      const out: SSEMessage[] = [];
      for (const line of lines) {
        out.push(...consumeLine(line));
      }
      return out;
    },
    flush() {
      const out: SSEMessage[] = [];
      if (remainder !== "") {
        out.push(...consumeLine(remainder));
        remainder = "";
      }
      out.push(...complete());
      return out;
    },
  };
}

/**
 * Reads a fetch response body as server-sent events until it closes or the
 * signal aborts. Each event is handed to `onMessage` in order; an abort resolves
 * quietly rather than throwing, because stopping a stream is the reader's
 * choice, not a failure.
 */
export async function readEventStream(
  body: ReadableStream<Uint8Array>,
  onMessage: (message: SSEMessage) => void,
  signal?: AbortSignal,
): Promise<void> {
  const reader = body.getReader();
  const decoder = new TextDecoder();
  const parser = createSSEParser();

  const cancel = () => {
    void reader.cancel().catch(() => undefined);
  };
  signal?.addEventListener("abort", cancel, { once: true });

  try {
    for (;;) {
      const { done, value } = await reader.read();
      if (done) {
        break;
      }
      for (const message of parser.feed(decoder.decode(value, { stream: true }))) {
        onMessage(message);
      }
    }
    for (const message of parser.feed(decoder.decode())) {
      onMessage(message);
    }
    for (const message of parser.flush()) {
      onMessage(message);
    }
  } catch (error) {
    if (signal?.aborted) {
      return;
    }
    throw error;
  } finally {
    signal?.removeEventListener("abort", cancel);
  }
}
