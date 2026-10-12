import { describe, expect, it } from "vitest";
import { markdownHasMath, prepareMarkdown } from "../markdown-prepare";

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
      "Rate:\n\n\n\n```dk-math-raw\n$$\nr = d / t\n```\n",
    );
    expect(prepareMarkdown("$$r = d / t$$", { streaming: true })).toBe("$$r = d / t$$");
  });
});

/**
 * The math typesetter is fetched only for a reply that has math in it, so
 * money, code and plain prose must never be read as math.
 */
describe("markdownHasMath", () => {
  const has = (text: string) => markdownHasMath(prepareMarkdown(text));

  it("finds inline and display math, however it was delimited", () => {
    expect(has("The rate is $r = d / t$ here.")).toBe(true);
    expect(has("$$\\sum_i x_i$$")).toBe(true);
    expect(has("Solve \\(x^2 = 4\\).")).toBe(true);
    expect(has("\\[a + b\\]")).toBe(true);
  });

  it("does not take money for math", () => {
    expect(has("$48,210.00 across six customers, $9,835.50 of it Acme")).toBe(false);
  });

  it("does not look inside code", () => {
    expect(has("Run `echo $HOME` then\n\n```sh\nprice=$x\n```\n")).toBe(false);
  });

  it("is false for prose with no dollar sign at all", () => {
    expect(has("Three loads are late at Dallas.")).toBe(false);
  });

  it("finds math beside money in the same reply", () => {
    expect(has("$1,200 at $r$ per mile")).toBe(true);
  });
});
