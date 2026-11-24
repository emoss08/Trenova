import { cleanup, render } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { StreamingAiMarkdown } from "../ai-markdown";

const parsed = vi.hoisted(() => [] as string[]);

vi.mock("@tanstack/markdown/parser", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tanstack/markdown/parser")>();
  return {
    ...actual,
    parseMarkdown: (...args: Parameters<typeof actual.parseMarkdown>) => {
      parsed.push(args[0]);
      return actual.parseMarkdown(...args);
    },
  };
});

afterEach(() => {
  cleanup();
  parsed.length = 0;
});

/**
 * Every token of a reply re-parsed the whole reply, so the cost of drawing it
 * grew with the square of its length. A block the reply has moved past is
 * parsed once.
 */
describe("StreamingAiMarkdown", () => {
  it("parses again only the block still arriving", () => {
    const { rerender } = render(<StreamingAiMarkdown content={"First.\n\nSecond"} />);
    rerender(<StreamingAiMarkdown content={"First.\n\nSecond, longer"} />);
    rerender(<StreamingAiMarkdown content={"First.\n\nSecond, longer still."} />);

    expect(parsed.filter((content) => content === "First.\n\n")).toHaveLength(1);
    expect(parsed.filter((content) => content.startsWith("Second"))).toHaveLength(3);
  });
});
