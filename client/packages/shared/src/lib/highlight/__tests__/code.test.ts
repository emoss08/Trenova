import {
  codeLanguage,
  countMatches,
  createMatcher,
  findMatches,
  highlightLines,
  lineText,
  markSegments,
} from "@trenova/shared/lib/highlight/code";
import { describe, expect, it } from "vitest";

describe("codeLanguage", () => {
  it("names the grammar for a fence label, its aliases and any case", () => {
    expect(codeLanguage("json")).toBe("json");
    expect(codeLanguage("JSONC")).toBe("json");
    expect(codeLanguage("javascript")).toBe("js");
    expect(codeLanguage("typescript")).toBe("ts");
    expect(codeLanguage("plsql")).toBe("sql");
    expect(codeLanguage("PostgreSQL")).toBe("sql");
    expect(codeLanguage("gql")).toBe("graphql");
    expect(codeLanguage("yml")).toBe("yaml");
    expect(codeLanguage("bash")).toBe("shell");
    expect(codeLanguage("patch")).toBe("diff");
    expect(codeLanguage("py")).toBe("python");
  });

  it("answers null for a language the app has no grammar for, or none at all", () => {
    expect(codeLanguage("cobol")).toBeNull();
    expect(codeLanguage("text")).toBeNull();
    expect(codeLanguage("")).toBeNull();
    expect(codeLanguage(undefined)).toBeNull();
  });
});

describe("highlightLines", () => {
  it("splits code into lines of segments, a token crossing a line break split with it", () => {
    const lines = highlightLines('{\n  "note": "a",\n  "n": 2\n}', "json");

    expect(lines).toEqual([
      [{ text: "{" }],
      [
        { text: "  " },
        { role: "property", text: '"note"' },
        { text: ": " },
        { role: "string", text: '"a"' },
        { text: "," },
      ],
      [
        { text: "  " },
        { role: "property", text: '"n"' },
        { text: ": " },
        { role: "number", text: "2" },
      ],
      [{ text: "}" }],
    ]);
  });

  it("keeps empty lines, including a trailing one, so line numbers match the text", () => {
    expect(highlightLines("a\n\nb\n", "text").map(lineText)).toEqual(["a", "", "b", ""]);
  });

  it("shows code in a language it has no grammar for as one plain segment per line", () => {
    expect(highlightLines("PERFORM X.\nSTOP RUN.", "cobol")).toEqual([
      [{ text: "PERFORM X." }],
      [{ text: "STOP RUN." }],
    ]);
  });

  it("colours the Python an analysis runs", () => {
    const line = highlightLines("import pandas as pd", "python")[0];
    expect(line[0]).toEqual({ role: "keyword", text: "import" });
  });

  it("colours SQL fenced as plsql the way it colours sql", () => {
    expect(highlightLines("select 1", "plsql")).toEqual(highlightLines("select 1", "sql"));
    expect(highlightLines("select 1", "plsql")[0][0]).toEqual({ role: "keyword", text: "select" });
  });
});

describe("matching", () => {
  it("finds every occurrence ignoring case, and treats the query as text, not a pattern", () => {
    expect(findMatches("Ship ship SHIP", createMatcher("ship"))).toEqual([
      [0, 4],
      [5, 9],
      [10, 14],
    ]);
    expect(findMatches("a.b axb", createMatcher("a.b"))).toEqual([[0, 3]]);
    expect(findMatches("(x)", createMatcher("("))).toEqual([[0, 1]]);
  });

  it("finds nothing for a blank query", () => {
    expect(createMatcher("")).toBeNull();
    expect(createMatcher("   ")).toBeNull();
    expect(findMatches("anything", null)).toEqual([]);
    expect(countMatches("anything", null)).toBe(0);
  });

  it("counts matches on every line, which is what the marks show", () => {
    const code = '{\n  "status": "Ready",\n  "readyAt": 1\n}';
    expect(countMatches(code, createMatcher("ready"))).toBe(2);
  });

  it("marks a match that spans a property and the punctuation after it, keeping roles", () => {
    const [, line] = highlightLines('{\n  "status": "Ready"\n}', "json");
    const marked = markSegments(line, findMatches(lineText(line), createMatcher('us": "re')));

    expect(marked).toEqual([
      { text: "  ", marked: false },
      { role: "property", text: '"stat', marked: false },
      { role: "property", text: 'us"', marked: true },
      { text: ": ", marked: true },
      { role: "string", text: '"Re', marked: true },
      { role: "string", text: 'ady"', marked: false },
    ]);
  });

  it("marks two matches inside one segment and leaves a line without one untouched", () => {
    const line = [{ role: "string" as const, text: "abcabc" }];
    expect(markSegments(line, findMatches("abcabc", createMatcher("b")))).toEqual([
      { role: "string", text: "a", marked: false },
      { role: "string", text: "b", marked: true },
      { role: "string", text: "ca", marked: false },
      { role: "string", text: "b", marked: true },
      { role: "string", text: "c", marked: false },
    ]);
    expect(markSegments(line, [])).toEqual([{ role: "string", text: "abcabc", marked: false }]);
  });
});
