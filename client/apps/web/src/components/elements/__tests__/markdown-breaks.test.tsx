import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AiMarkdown } from "../ai-markdown";

describe("line breaks in a reply", () => {
  it("breaks the line at a single newline and at two trailing spaces", () => {
    const { container } = render(<AiMarkdown content={"First line\nSecond line  \nThird"} />);
    const paragraph = container.querySelector("p");
    expect(paragraph?.querySelectorAll("br")).toHaveLength(2);
    expect(paragraph?.textContent).toBe("First lineSecond lineThird");
  });

  it("breaks inside emphasis too", () => {
    const { container } = render(<AiMarkdown content={"**Bold one\nBold two**"} />);
    expect(container.querySelector("strong br")).not.toBeNull();
  });

  it("leaves code spans and code blocks alone", () => {
    const { container } = render(
      <AiMarkdown content={"Run `a\nb` now\n\n```sh\necho one\necho two\n```"} />,
    );
    expect(container.querySelector("p")?.querySelectorAll("br")).toHaveLength(0);
    expect(container.querySelector(".md-icode")?.textContent).toBe("a b");
    expect(container.querySelector(".md-code pre")?.textContent).toBe("echo one\necho two");
  });

  it("puts no break beside a formula on its own lines", () => {
    const { container } = render(<AiMarkdown content={"Where:\n$$x^2$$\nholds."} />);
    expect(container.querySelectorAll("br")).toHaveLength(0);
  });
});
