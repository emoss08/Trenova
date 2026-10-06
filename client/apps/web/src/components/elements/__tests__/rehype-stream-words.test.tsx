import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { StreamingAiMarkdown } from "../ai-markdown";

describe("StreamingAiMarkdown words", () => {
  it("wraps each word and ends the reply in a caret, leaving code as written", () => {
    const { container } = render(
      <StreamingAiMarkdown
        content={"First block here.\n\nSecond **bold** words and `code here`"}
        wordClassName="w"
        caretClassName="caret"
      />,
    );
    const words = [...container.querySelectorAll(".w")].map((node) => node.textContent);
    expect(words).toContain("First ");
    expect(words).toContain("bold");
    expect(words.some((word) => word?.includes("code"))).toBe(false);
    expect(container.querySelectorAll(".caret")).toHaveLength(1);
    expect(container.textContent).toBe("First block here.\nSecond bold words and code here");
  });

  it("leaves the text alone when no class is asked for", () => {
    const { container } = render(<StreamingAiMarkdown content="Plain reply." />);
    expect(container.querySelectorAll("span")).toHaveLength(0);
  });
});
