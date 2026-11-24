import type { BlockNode, InlineNode } from "@tanstack/markdown";
import { parseMarkdown } from "@tanstack/markdown/parser";
import { describe, expect, it } from "vitest";
import { mathExtension } from "../markdown-math";

const parse = (text: string) => parseMarkdown(text, { extensions: [mathExtension()] }).children;

function formulas(nodes: (BlockNode | InlineNode)[]): [string, string][] {
  return nodes.flatMap((node): [string, string][] => {
    if ((node.type === "component" || node.type === "inlineComponent") && node.name === "math") {
      return [[node.attributes.tex, node.attributes.mode]];
    }
    return "children" in node ? formulas(node.children) : [];
  });
}

function text(nodes: (BlockNode | InlineNode)[]): string {
  return nodes
    .map((node) => {
      if (node.type === "text") return node.value;
      return "children" in node ? text(node.children) : "";
    })
    .join("");
}

describe("math in markdown", () => {
  it("reads $…$ as a formula in the line", () => {
    expect(formulas(parse("The rate is $r = d / t$ per mile."))).toEqual([["r = d / t", "inline"]]);
  });

  it("reads a $$ block over several lines as a block", () => {
    expect(formulas(parse("$$\na + b\n= c\n$$\n\nAfter."))).toEqual([["a + b\n= c", "block"]]);
    expect(text(parse("$$\na + b\n$$\n\nAfter."))).toBe("After.");
  });

  it("reads $$…$$ alone on its line as a displayed formula", () => {
    expect(formulas(parse("$$x^2$$"))).toEqual([["x^2", "block"]]);
    expect(formulas(parse("Where:\n$$x^2$$\nholds."))).toEqual([["x^2", "display"]]);
    expect(formulas(parse("Inline $$x^2$$ here."))).toEqual([["x^2", "inline"]]);
  });

  it("keeps an escaped dollar literal, so money stays money", () => {
    const nodes = parse("\\$48,210.00 across six, \\$9,835.50 of it Acme");
    expect(formulas(nodes)).toEqual([]);
    expect(text(nodes)).toBe("$48,210.00 across six, $9,835.50 of it Acme");
  });

  it("does not close a formula on an escaped dollar", () => {
    expect(formulas(parse("Cost $a \\$ b$ total"))).toEqual([["a \\$ b", "inline"]]);
  });

  it("leaves code and a lone dollar as written", () => {
    expect(formulas(parse("Run `echo $a$` now"))).toEqual([]);
    expect(text(parse("A lone $ sign"))).toBe("A lone $ sign");
  });

  it("takes an unclosed block to the end of the reply", () => {
    expect(formulas(parse("$$\nr = d / t"))).toEqual([["r = d / t", "block"]]);
  });
});
