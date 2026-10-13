import { HighlightedCode } from "@trenova/shared/components/ui/highlighted-code";
import { cleanup, fireEvent, render } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

afterEach(cleanup);

function lines(container: HTMLElement): HTMLElement[] {
  return [...container.querySelectorAll<HTMLElement>("[data-line]")];
}

describe("HighlightedCode", () => {
  it("renders the code's exact text, one element per line, coloured by role tokens", () => {
    const code = '{\n  "id": 7,\n  "ok": true\n}';
    const { container } = render(<HighlightedCode code={code} language="json" />);

    expect(container.querySelector("pre")?.textContent).toBe(code);
    expect(container.querySelector("pre")?.dataset.language).toBe("json");
    expect(lines(container).map((line) => line.textContent)).toEqual([
      "{",
      '  "id": 7,',
      '  "ok": true',
      "}",
    ]);

    const property = [...container.querySelectorAll("span")].find((s) => s.textContent === '"id"');
    expect(property?.className).toBe("text-syntax-property");
    const number = [...container.querySelectorAll("span")].find((s) => s.textContent === "7");
    expect(number?.className).toBe("text-syntax-number");
  });

  it("shows code in a language it has no grammar for as plain text", () => {
    const { container } = render(<HighlightedCode code="PERFORM X." language="cobol" />);

    expect(container.querySelector("pre")?.dataset.language).toBeUndefined();
    expect(lines(container)[0].innerHTML).toBe("PERFORM X.");
  });

  it("never turns code into markup", () => {
    const code = '"<img src=x onerror=alert(1)>"';
    const { container } = render(<HighlightedCode code={code} language="json" query="img" />);

    expect(container.querySelector("img")).toBeNull();
    expect(container.querySelector("pre")?.textContent).toBe(code);
  });

  it("marks each occurrence of the query, ignoring case, across token boundaries", () => {
    const { container } = render(
      <HighlightedCode code={'{\n  "Ready": "ready"\n}'} language="json" query='dy": "re' />,
    );

    const marks = [...container.querySelectorAll("mark")].map((mark) => mark.textContent);
    expect(marks).toEqual(['dy"', ": ", '"re']);
    expect(container.querySelector("pre")?.textContent).toBe('{\n  "Ready": "ready"\n}');
  });

  it("draws what lineEnd returns after each line's code, given its text and index", () => {
    const { container } = render(
      <HighlightedCode
        code={"one\ntwo"}
        lineEnd={(text, index) => (text === "two" ? <b>{`end ${index}`}</b> : null)}
      />,
    );

    expect(lines(container).map((line) => line.textContent)).toEqual(["one", "twoend 1"]);
  });

  it("reports the index of the line clicked, from anywhere inside it", () => {
    const onLineClick = vi.fn();
    const { container } = render(
      <HighlightedCode code={'{\n  "a": 1\n}'} language="json" onLineClick={onLineClick} />,
    );

    const number = [...container.querySelectorAll("span")].find((s) => s.textContent === "1");
    fireEvent.click(number as HTMLElement);
    fireEvent.click(lines(container)[2]);

    expect(onLineClick.mock.calls).toEqual([[1], [2]]);
  });
});
