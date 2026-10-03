import { describe, expect, it } from "vitest";
import { prepareMarkdown } from "../markdown-prepare";

describe("prepareMarkdown", () => {
  it("reads a dollar before a digit as money, not math", () => {
    expect(prepareMarkdown("$48,210.00 across six, $9,835.50 of it Acme")).toBe(
      "\\$48,210.00 across six, \\$9,835.50 of it Acme",
    );
  });

  it("turns paired bracket delimiters into math and leaves a lone one escaped", () => {
    expect(prepareMarkdown("\\[x^2\\] and \\(y\\)")).toBe("$$x^2$$ and $y$");
    expect(prepareMarkdown("a lone \\[ bracket")).toBe("a lone \\[ bracket");
  });

  it("leaves code as written", () => {
    const text = "Run `echo $5` then\n\n```sh\nprice=$9\n```\n";
    expect(prepareMarkdown(text)).toBe(text);
  });

  it("keeps an unclosed display block raw while the reply is arriving", () => {
    expect(prepareMarkdown("Rate:\n\n$$\nr = d / t", { streaming: true })).toBe(
      "Rate:\n\n\\$\\$\nr = d / t",
    );
    expect(prepareMarkdown("$$r = d / t$$", { streaming: true })).toBe("$$r = d / t$$");
  });
});
