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
    expect(container.textContent).toBe("First block here.Second bold words and code here");
  });

  it("puts the caret after the last word, past a closing code block", () => {
    const { container } = render(
      <StreamingAiMarkdown
        content={"Last **words** here\n```\nselect 1\n```"}
        wordClassName="w"
        caretClassName="caret"
      />,
    );
    const caret = container.querySelector(".caret");
    expect(caret?.getAttribute("aria-hidden")).toBe("true");
    expect(caret?.parentElement?.tagName).toBe("P");
    expect(caret?.previousElementSibling?.textContent).toBe("here");
  });

  it("leaves a formula whole", () => {
    const { container } = render(
      <StreamingAiMarkdown content={"Rate $r = d / t$ now"} wordClassName="w" />,
    );
    const words = [...container.querySelectorAll(".w")].map((node) => node.textContent);
    expect(words).toEqual(["Rate ", "now"]);
  });

  it("leaves the text alone when no class is asked for", () => {
    const { container } = render(<StreamingAiMarkdown content="Plain reply." />);
    expect(container.querySelectorAll("span")).toHaveLength(0);
  });
});
