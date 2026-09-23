import { afterEach, describe, expect, it } from "vitest";
import { openTurnReaderCount, registerTurnReader, releaseTurnReaders } from "../turn-readers";

afterEach(() => {
  releaseTurnReaders();
});

describe("turn readers", () => {
  it("lets go of every open reader at once", () => {
    const first = new AbortController();
    const second = new AbortController();
    registerTurnReader(first);
    registerTurnReader(second);

    expect(releaseTurnReaders()).toBe(2);
    expect(first.signal.aborted).toBe(true);
    expect(second.signal.aborted).toBe(true);
    expect(openTurnReaderCount()).toBe(0);
  });

  it("forgets a reader that let go of its own accord", () => {
    const reader = new AbortController();
    registerTurnReader(reader);
    reader.abort();

    expect(openTurnReaderCount()).toBe(0);
    expect(releaseTurnReaders()).toBe(0);
  });

  it("forgets a reader that finished and unregistered, and never aborts it later", () => {
    const reader = new AbortController();
    const unregister = registerTurnReader(reader);
    unregister();

    expect(releaseTurnReaders()).toBe(0);
    expect(reader.signal.aborted).toBe(false);
  });

  it("does not track a reader that was already released", () => {
    const reader = new AbortController();
    reader.abort();
    registerTurnReader(reader);

    expect(openTurnReaderCount()).toBe(0);
  });
});
