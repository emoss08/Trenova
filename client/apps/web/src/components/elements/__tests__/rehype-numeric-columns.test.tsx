import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AiMarkdown } from "../ai-markdown";

const TABLE = [
  "| Customer | Items | Amount | Note |",
  "| --- | --- | :-- | --- |",
  "| Acme | 3 | $1,486.00 | late |",
  "| Globex | — | $210.50 | 12 |",
].join("\n");

describe("number columns", () => {
  it("right-aligns columns of figures and leaves words and author alignment alone", () => {
    const { container } = render(<AiMarkdown content={TABLE} />);
    const head = [...container.querySelectorAll("th")];
    const isNum = (cell: Element) => cell.classList.contains("md-num");
    expect(head.map(isNum)).toEqual([false, true, false, false]);
    const firstRow = [...container.querySelectorAll("tbody tr:first-child td")];
    expect(firstRow.map(isNum)).toEqual([false, true, false, false]);
  });
});
