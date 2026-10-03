import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AiMarkdown, StreamingAiMarkdown } from "../ai-markdown";

describe("the Desk's markdown subset", () => {
  it("leaves a bare address as text but keeps written links", () => {
    const { container } = render(
      <AiMarkdown
        deskSubset
        content={
          "See https://example.com and [the docs](https://docs.example.com) or <https://a.example.com>."
        }
      />,
    );
    const links = [...container.querySelectorAll("a")].map((a) => a.textContent);
    expect(links).toEqual(["the docs", "https://a.example.com"]);
    expect(container.textContent).toContain("https://example.com");
  });

  it("shows images, footnotes and indented code as the words that were written", () => {
    const { container } = render(
      <AiMarkdown
        deskSubset
        content={
          "![truck](https://x.example/t.png)\n\nNote[^1].\n\n[^1]: A footnote.\n\n    indented code"
        }
      />,
    );
    expect(container.querySelector("img")).toBeNull();
    expect(container.querySelector("pre")).toBeNull();
    expect(container.querySelector("sup")).toBeNull();
    expect(container.textContent).toContain("![truck](https://x.example/t.png)");
    expect(container.textContent).toContain("[^1]");
    expect(container.textContent).toContain("indented code");
  });

  it("labels an unlabelled fence as text", () => {
    const { container } = render(<AiMarkdown deskSubset content={"```\nplain\n```"} />);
    expect(container.querySelector(".md-code-h span")?.textContent).toBe("text");
  });

  it("shows a math block that has not closed as its raw text in its own box", () => {
    const { container } = render(
      <StreamingAiMarkdown deskSubset content={"Rate:\n\n$$\nr = d / t"} />,
    );
    expect(container.querySelector(".md-math-raw")?.textContent).toContain("r = d / t");
    expect(container.querySelector(".katex")).toBeNull();
  });

  it("finds a reference link's definition in a later block while streaming", () => {
    const { container } = render(
      <StreamingAiMarkdown
        deskSubset
        content={"See [the queue][q].\n\nMore text.\n\n[q]: /billing/queue"}
      />,
    );
    expect(container.querySelector("a")?.textContent).toBe("the queue");
  });

  it("keeps images for the surfaces that are not the Desk", () => {
    const { container } = render(<AiMarkdown content={"![truck](https://x.example/t.png)"} />);
    expect(container.textContent).not.toContain("![truck]");
  });

  it("sets headings by level", () => {
    const { container } = render(
      <AiMarkdown deskSubset content={"# One\n\n## Two\n\n#### Four"} />,
    );
    expect([...container.querySelectorAll(".md-h")].map((h) => h.className)).toEqual([
      expect.stringContaining("md-h1"),
      expect.stringContaining("md-h2"),
      expect.stringContaining("md-h4"),
    ]);
  });
});
