import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { HighlightProvider, highlightRule, hoverProps, useSetHot } from "../highlight";

function sheetText(): string {
  return document.adoptedStyleSheets
    .flatMap((sheet) => Array.from(sheet.cssRules, (rule) => rule.cssText))
    .join("\n");
}

function Mark({ seriesKey, renders }: { seriesKey: string; renders: string[] }) {
  const setHot = useSetHot();
  renders.push(seriesKey);

  return (
    <button type="button" data-series={seriesKey} {...hoverProps(setHot, seriesKey)}>
      {seriesKey}
    </button>
  );
}

function renderMarks(keys: string[]) {
  const renders: string[] = [];
  const view = render(
    <HighlightProvider>
      {keys.map((key) => (
        <Mark key={key} seriesKey={key} renders={renders} />
      ))}
    </HighlightProvider>,
  );
  const scope = view.container.querySelector<HTMLElement>("[data-hl]");
  if (!scope) {
    throw new Error("the provider renders no scope element");
  }

  return { renders, scope, unmount: view.unmount };
}

describe("highlightRule", () => {
  it("quotes a series key that would otherwise end the attribute selector", () => {
    expect(highlightRule("_r_1_", 'gpt "x"')).toBe(
      '[data-hl="_r_1_"] [data-series="gpt\\ \\"x\\""]{--an-hot:1}',
    );
  });
});

describe("HighlightProvider", () => {
  it("lights the hovered series without re-rendering the marks", () => {
    const { renders, scope } = renderMarks(["claude", "llama"]);
    expect(renders).toEqual(["claude", "llama"]);
    expect(scope.hasAttribute("data-hot")).toBe(false);

    fireEvent.mouseEnter(screen.getByText("llama"));

    expect(scope.hasAttribute("data-hot")).toBe(true);
    expect(sheetText()).toContain(`[data-series="llama"]`);
    expect(sheetText()).not.toContain(`[data-series="claude"]`);
    expect(renders).toEqual(["claude", "llama"]);
  });

  it("clears the highlight when the pointer leaves", () => {
    const { scope } = renderMarks(["claude"]);

    fireEvent.mouseEnter(screen.getByText("claude"));
    fireEvent.mouseLeave(screen.getByText("claude"));

    expect(scope.hasAttribute("data-hot")).toBe(false);
    expect(sheetText()).not.toContain("data-series");
  });

  it("takes its stylesheet with it when it unmounts", () => {
    const before = document.adoptedStyleSheets.length;
    const { unmount } = renderMarks(["claude"]);
    expect(document.adoptedStyleSheets).toHaveLength(before + 1);

    unmount();

    expect(document.adoptedStyleSheets).toHaveLength(before);
  });
});
